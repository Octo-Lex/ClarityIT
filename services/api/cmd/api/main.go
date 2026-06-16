package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/clarityit/api/internal/admin"
	"github.com/clarityit/api/internal/agent"
	"github.com/clarityit/api/internal/approval"
	"github.com/clarityit/api/internal/artifact"
	"github.com/clarityit/api/internal/config"
	"github.com/clarityit/api/internal/contextx"
	"github.com/clarityit/api/internal/database"
	"github.com/clarityit/api/internal/domain"
	"github.com/clarityit/api/internal/iam"
	"github.com/clarityit/api/internal/middleware"
	"github.com/clarityit/api/internal/presenton"
	"github.com/clarityit/api/internal/health"
	"github.com/clarityit/api/internal/mfa"
	"github.com/clarityit/api/internal/integration"
	"github.com/clarityit/api/internal/knowledge"
	"github.com/clarityit/api/internal/proxmox"
	"github.com/clarityit/api/internal/remediation"
	"github.com/clarityit/api/internal/storage"
	"github.com/clarityit/api/internal/team"
	"github.com/clarityit/api/internal/wsx"
	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/nats-io/nats.go"
	"github.com/redis/go-redis/v9"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Config error: %v", err)
	}
	config.InitLogger("clarityit-api", cfg.Version)
	if err := cfg.Validate(); err != nil {
		log.Fatalf("Config validation: %v", err)
	}

	log.Printf("ClarityIT API starting on :%s", cfg.Port)

	pool, err := database.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Database connection: %v", err)
	}
	defer pool.Close()
	log.Println("Database connected")

	iamHandler := iam.NewHandler(pool, cfg)

	// MFA handler (Track 1: Real MFA)
	mfaHandler, err := mfa.NewHandler(pool, cfg)
	if err != nil {
		log.Fatalf("failed to create MFA handler: %v", err)
	}
	teamHandler := team.NewHandler(pool, cfg)
	adminHandler := admin.NewHandler(pool, cfg)
	domainHandler := domain.NewHandler(pool, cfg)
	patternsHandler := domain.NewPatternsHandler(pool)

	wsHub := wsx.NewHub()

	// Wire Redis pub/sub for WebSocket fanout
	rdb := redis.NewClient(&redis.Options{Addr: cfg.RedisURLHost()})

	// Connect to NATS for deep health + event transport
	nc, err := nats.Connect(cfg.NATSURL)
	if err != nil {
		log.Printf("NATS not available: %v", err)
	}

	// MinIO health checker + real S3 client
	var minioChecker health.S3BucketChecker
	var s3Client storage.S3Client // nil unless MinIO configured
	if cfg.MinioEndpoint != "" {
		minioChecker = health.NewMinIOHealthChecker(cfg.MinioEndpoint, cfg.MinioUseSSL)
		// Create real MinIO S3 client
		mc, s3err := storage.NewMinIOClient(cfg.MinioEndpoint, cfg.MinioAccessKey, cfg.MinioSecretKey, cfg.MinioUseSSL)
		if s3err != nil {
			config.Warn("failed to create MinIO S3 client", map[string]any{"error": s3err.Error()})
		} else {
			// Ensure bucket exists
			if berr := mc.EnsureBucket(ctx, cfg.MinioBucket); berr != nil {
				config.Warn("failed to ensure MinIO bucket", map[string]any{"bucket": cfg.MinioBucket, "error": berr.Error()})
			} else {
				config.Info("MinIO S3 client connected", map[string]any{"endpoint": cfg.MinioEndpoint, "bucket": cfg.MinioBucket})
				s3Client = mc
			}
		}
	}
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Printf("Redis not available, WebSocket fanout disabled: %v", err)
	} else {
		log.Println("Redis connected, WebSocket fanout active")
		wsFanoutCh := make(chan string, 256)
		go func() {
			sub := rdb.Subscribe(ctx, "clarity:ws:events")
			for msg := range sub.Channel() {
				wsFanoutCh <- msg.Payload
			}
		}()
		wsHub.SubscribeRedis(wsFanoutCh)
	}

	r := chi.NewRouter()
	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)
	r.Use(chimw.Timeout(30 * time.Second))
	r.Use(middleware.ResolveAuth(cfg.JWTSecret))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "service": "clarityit", "version": "0.7.0"})
	})

	// Deep health (authenticated)
	healthHandler := health.NewHandlerWithDeps(pool, cfg.Version, "", nc, rdb, minioChecker, cfg.MinioBucket)
	r.Get("/metrics", healthHandler.Metrics)
	r.With(middleware.RequireAuth).Get("/api/health/deep", healthHandler.Deep)

	opsHandler := admin.NewOpsHandler(pool, healthHandler)

	// ─── WebSocket ───
	r.With(middleware.RequireAuth).Get("/api/ws", wsHub.HandleWS)

	// ─── Bootstrap ───
	r.With(middleware.Idempotency(middleware.IdempotencyConfig{Pool: pool, Scope: "system", Expiry: 24 * time.Hour})).
		Post("/api/bootstrap", iamHandler.Bootstrap)

	// ─── Auth ───
	r.With(middleware.Idempotency(middleware.IdempotencyConfig{Pool: pool, Scope: "user", Expiry: 1 * time.Hour})).
		Post("/api/auth/register", iamHandler.Register)
	r.Post("/api/auth/login", iamHandler.Login)
	r.Post("/api/auth/refresh", iamHandler.Refresh)
	r.With(middleware.Idempotency(middleware.IdempotencyConfig{Pool: pool, Scope: "user", Expiry: 1 * time.Hour})).
		Post("/api/auth/forgot-password", iamHandler.ForgotPassword)
	r.With(middleware.Idempotency(middleware.IdempotencyConfig{Pool: pool, Scope: "user", Expiry: 1 * time.Hour})).
		Post("/api/auth/reset-password", iamHandler.ResetPassword)

	r.Group(func(r chi.Router) {
		r.Use(middleware.RequireAuth)
		r.With(middleware.Idempotency(middleware.IdempotencyConfig{Pool: pool, Scope: "user", Expiry: 1 * time.Hour})).
			Post("/api/auth/change-password", iamHandler.ChangePassword)
		r.With(middleware.Idempotency(middleware.IdempotencyConfig{Pool: pool, Scope: "user", Expiry: 1 * time.Hour})).
			Post("/api/auth/logout", iamHandler.Logout)
		r.Get("/api/auth/me", iamHandler.Me)
		r.Post("/api/auth/switch-team", iamHandler.SwitchTeam)
		r.Get("/api/auth/permissions", iamHandler.Permissions)
		r.Get("/api/auth/sessions", iamHandler.ListSessions)
		r.With(middleware.Idempotency(middleware.IdempotencyConfig{Pool: pool, Scope: "user", Expiry: 1 * time.Hour})).
			Delete("/api/auth/sessions/{id}", iamHandler.RevokeSession)

		// MFA routes (Track 1: Real MFA)
		r.Mount("/api/auth/mfa", mfaHandler.Routes())

		// WebAuthn routes (v1.1 Track 5)
		webAuthnHandler, err := mfa.NewWebAuthnHandler(pool, cfg)
		if err != nil {
			log.Printf("WebAuthn init error: %v", err)
		}
		if webAuthnHandler != nil && webAuthnHandler.IsWebAuthnEnabled() {
			r.Mount("/api/auth/mfa", webAuthnHandler.Routes())
		}
	})

	// ─── Team Management ───
	// Phase 8: Webhook receiver (integration key auth, no JWT)
	integrationHandler := integration.NewHandlerWithEnv(pool, cfg.HMACKey, cfg.Env)
	// Phase 8: Webhook rate limiter (outside JWT auth)
	webhookRL := middleware.NewRateLimiter(middleware.RateLimiterConfig{HMACKey: cfg.HMACKey, MaxRequests: 60, Window: 1 * time.Minute})
	r.Post("/api/webhooks/{source}", webhookRL.Middleware(http.HandlerFunc(integrationHandler.ReceiveWebhook)).ServeHTTP)

	// v1.5 Knowledge handler (outer scope — used in both team and admin routes)
	knowledgeHandler := knowledge.NewHandler(pool)

	r.Route("/api/teams/{teamId}", func(r chi.Router) {
		r.Use(middleware.RequireAuth)
		r.With(middleware.RequirePermission(pool, "team.settings.read")).Get("/settings", teamHandler.GetSettings)
		r.With(middleware.RequirePermission(pool, "team.settings.update")).
			With(middleware.Idempotency(middleware.IdempotencyConfig{Pool: pool, Scope: "user", Expiry: 1 * time.Hour})).
			Patch("/settings", teamHandler.UpdateSettings)
		r.With(middleware.RequirePermission(pool, "team.members.read")).Get("/members", teamHandler.ListMembers)
		r.With(middleware.RequirePermission(pool, "team.members.update")).
			With(middleware.Idempotency(middleware.IdempotencyConfig{Pool: pool, Scope: "user", Expiry: 1 * time.Hour})).
			Patch("/members/{membershipId}", teamHandler.UpdateMemberRole)
		r.With(middleware.RequirePermission(pool, "team.members.remove")).
			With(middleware.Idempotency(middleware.IdempotencyConfig{Pool: pool, Scope: "user", Expiry: 1 * time.Hour})).
			Delete("/members/{membershipId}", teamHandler.RemoveMember)
		r.With(middleware.RequirePermission(pool, "team.invitations.create")).
			With(middleware.Idempotency(middleware.IdempotencyConfig{Pool: pool, Scope: "user", Expiry: 1 * time.Hour})).
			Post("/invitations", teamHandler.CreateInvitation)
		r.With(middleware.RequirePermission(pool, "team.invitations.read")).Get("/invitations", teamHandler.ListInvitations)
		r.Post("/invitations/{id}/accept", teamHandler.AcceptInvitation)
		r.With(middleware.RequirePermission(pool, "team.invitations.revoke")).
			With(middleware.Idempotency(middleware.IdempotencyConfig{Pool: pool, Scope: "user", Expiry: 1 * time.Hour})).
			Delete("/invitations/{id}", teamHandler.RevokeInvitation)
		r.With(middleware.RequirePermission(pool, "team.access_grants.read")).Get("/access-grants", teamHandler.ListAccessGrants)
		r.With(middleware.RequirePermission(pool, "team.access_grants.create")).
			With(middleware.Idempotency(middleware.IdempotencyConfig{Pool: pool, Scope: "user", Expiry: 1 * time.Hour})).
			Post("/access-grants", teamHandler.CreateAccessGrant)
		r.With(middleware.RequirePermission(pool, "team.access_grants.revoke")).
			With(middleware.Idempotency(middleware.IdempotencyConfig{Pool: pool, Scope: "user", Expiry: 1 * time.Hour})).
			Delete("/access-grants/{id}", teamHandler.RevokeAccessGrant)

		// ─── Domain: Objects ───
		r.With(middleware.RequirePermission(pool, "objects.create")).
			With(middleware.Idempotency(middleware.IdempotencyConfig{Pool: pool, Scope: "user", Expiry: 1 * time.Hour})).
			Post("/objects", domainHandler.CreateObject)
		r.With(middleware.RequirePermission(pool, "objects.read")).Get("/objects", domainHandler.ListObjects)
		r.With(middleware.RequirePermission(pool, "objects.read")).Get("/objects/{objectId}", domainHandler.GetObject)
		r.With(middleware.RequirePermission(pool, "objects.update")).
			With(middleware.Idempotency(middleware.IdempotencyConfig{Pool: pool, Scope: "user", Expiry: 1 * time.Hour})).
			Patch("/objects/{objectId}", domainHandler.UpdateObject)
		r.With(middleware.RequirePermission(pool, "objects.delete")).
			With(middleware.Idempotency(middleware.IdempotencyConfig{Pool: pool, Scope: "user", Expiry: 1 * time.Hour})).
			Delete("/objects/{objectId}", domainHandler.DeleteObject)

		// ─── Domain: Links ───
		r.With(middleware.RequirePermission(pool, "objects.links.create")).
			With(middleware.Idempotency(middleware.IdempotencyConfig{Pool: pool, Scope: "user", Expiry: 1 * time.Hour})).
			Post("/objects/{objectId}/links", domainHandler.CreateLink)
		r.With(middleware.RequirePermission(pool, "objects.links.read")).Get("/objects/{objectId}/links", domainHandler.ListLinks)
		r.With(middleware.RequirePermission(pool, "objects.links.delete")).
			With(middleware.Idempotency(middleware.IdempotencyConfig{Pool: pool, Scope: "user", Expiry: 1 * time.Hour})).
			Delete("/objects/{objectId}/links/{linkId}", domainHandler.DeleteLink)

		// ─── Domain: Comments ───
		r.With(middleware.RequirePermission(pool, "objects.comments.create")).
			With(middleware.Idempotency(middleware.IdempotencyConfig{Pool: pool, Scope: "user", Expiry: 1 * time.Hour})).
			Post("/objects/{objectId}/comments", domainHandler.CreateComment)
		r.With(middleware.RequirePermission(pool, "objects.comments.read")).Get("/objects/{objectId}/comments", domainHandler.ListComments)
		r.With(middleware.RequirePermission(pool, "objects.comments.update.own")).
			With(middleware.Idempotency(middleware.IdempotencyConfig{Pool: pool, Scope: "user", Expiry: 1 * time.Hour})).
			Patch("/objects/{objectId}/comments/{commentId}", domainHandler.UpdateComment)
		r.With(middleware.RequirePermission(pool, "objects.comments.delete.own")).
			With(middleware.Idempotency(middleware.IdempotencyConfig{Pool: pool, Scope: "user", Expiry: 1 * time.Hour})).
			Delete("/objects/{objectId}/comments/{commentId}", domainHandler.DeleteComment)

		// ─── Domain: Work Items ───
		r.With(middleware.RequirePermission(pool, "work.items.create")).
			With(middleware.Idempotency(middleware.IdempotencyConfig{Pool: pool, Scope: "user", Expiry: 1 * time.Hour})).
			Post("/work-items", domainHandler.CreateWorkItem)
		r.With(middleware.RequirePermission(pool, "work.items.view")).Get("/work-items", domainHandler.ListWorkItems)
		r.With(middleware.RequirePermission(pool, "work.items.view")).Get("/work-items/{objectId}", domainHandler.GetWorkItem)
		r.With(middleware.RequirePermission(pool, "work.items.update.own")).
			With(middleware.Idempotency(middleware.IdempotencyConfig{Pool: pool, Scope: "user", Expiry: 1 * time.Hour})).
			Patch("/work-items/{objectId}", domainHandler.UpdateWorkItem)
		r.With(middleware.RequirePermission(pool, "work.items.delete.own")).
			With(middleware.Idempotency(middleware.IdempotencyConfig{Pool: pool, Scope: "user", Expiry: 1 * time.Hour})).
			Delete("/work-items/{objectId}", domainHandler.DeleteWorkItem)
		r.With(middleware.RequirePermission(pool, "work.items.view")).Get("/work-items/board", domainHandler.BoardView)

		// ─── Domain: Incidents ───
		r.With(middleware.RequirePermission(pool, "incidents.create")).
			With(middleware.Idempotency(middleware.IdempotencyConfig{Pool: pool, Scope: "user", Expiry: 1 * time.Hour})).
			Post("/incidents", domainHandler.CreateIncident)
		r.With(middleware.RequirePermission(pool, "incidents.read")).Get("/incidents", domainHandler.ListIncidents)
		// v1.2 Track 2: Incident Pattern Detection — must be before {objectId}
		r.With(middleware.RequirePermission(pool, "incidents.read")).Get("/incidents/patterns", patternsHandler.GetPatterns)
		r.With(middleware.RequirePermission(pool, "incidents.read")).Get("/incidents/{objectId}", domainHandler.GetIncident)
		r.With(middleware.RequirePermission(pool, "incidents.update")).
			With(middleware.Idempotency(middleware.IdempotencyConfig{Pool: pool, Scope: "user", Expiry: 1 * time.Hour})).
			Patch("/incidents/{objectId}", domainHandler.UpdateIncident)
		r.With(middleware.RequirePermission(pool, "incidents.timeline.create")).
			With(middleware.Idempotency(middleware.IdempotencyConfig{Pool: pool, Scope: "user", Expiry: 1 * time.Hour})).
			Post("/incidents/{objectId}/timeline", domainHandler.AddTimeline)

		// ─── Domain: Projects ───
		r.With(middleware.RequirePermission(pool, "projects.create")).
			With(middleware.Idempotency(middleware.IdempotencyConfig{Pool: pool, Scope: "user", Expiry: 1 * time.Hour})).
			Post("/projects", domainHandler.CreateProject)
		r.With(middleware.RequirePermission(pool, "projects.view")).Get("/projects", domainHandler.ListProjects)
		r.With(middleware.RequirePermission(pool, "projects.view")).Get("/projects/{objectId}", domainHandler.GetProject)
		r.With(middleware.RequirePermission(pool, "projects.update")).
			With(middleware.Idempotency(middleware.IdempotencyConfig{Pool: pool, Scope: "user", Expiry: 1 * time.Hour})).
			Patch("/projects/{objectId}", domainHandler.UpdateProject)
		r.With(middleware.RequirePermission(pool, "projects.delete")).
			With(middleware.Idempotency(middleware.IdempotencyConfig{Pool: pool, Scope: "user", Expiry: 1 * time.Hour})).
			Delete("/projects/{objectId}", domainHandler.DeleteProject)

		// ─── Agent Runtime ───
		agentHandler := agent.NewHandler(pool)
		r.Route("/agents", func(r chi.Router) {
			r.With(middleware.RequirePermission(pool, "agents.create")).
				With(middleware.Idempotency(middleware.IdempotencyConfig{Pool: pool, Scope: "user", Expiry: 1 * time.Hour})).
				Post("/", agentHandler.CreateAgent)
			r.With(middleware.RequirePermission(pool, "agents.read")).Get("/", agentHandler.ListAgents)
			r.Route("/{agentId}", func(r chi.Router) {
				r.With(middleware.RequirePermission(pool, "agents.read")).Get("/", agentHandler.GetAgent)
				r.With(middleware.RequirePermission(pool, "agents.update")).
					With(middleware.Idempotency(middleware.IdempotencyConfig{Pool: pool, Scope: "user", Expiry: 1 * time.Hour})).
					Patch("/", agentHandler.UpdateAgent)
				r.With(middleware.RequirePermission(pool, "agents.disable")).
					With(middleware.Idempotency(middleware.IdempotencyConfig{Pool: pool, Scope: "user", Expiry: 1 * time.Hour})).
					Delete("/", agentHandler.DisableAgent)
				r.With(middleware.RequirePermission(pool, "agents.grants.create")).
					With(middleware.Idempotency(middleware.IdempotencyConfig{Pool: pool, Scope: "user", Expiry: 1 * time.Hour})).
					Post("/grants", agentHandler.CreateGrant)
				r.With(middleware.RequirePermission(pool, "agents.grants.read")).Get("/grants", agentHandler.ListGrants)
				r.With(middleware.RequirePermission(pool, "agents.grants.revoke")).
					With(middleware.Idempotency(middleware.IdempotencyConfig{Pool: pool, Scope: "user", Expiry: 1 * time.Hour})).
					Delete("/grants/{grantId}", agentHandler.RevokeGrant)
			})
		})

		// Agent Runs
		r.With(middleware.RequirePermission(pool, "agents.runs.create")).
			With(middleware.Idempotency(middleware.IdempotencyConfig{Pool: pool, Scope: "user", Expiry: 1 * time.Hour})).
			Post("/agent-runs", agentHandler.CreateRun)
		r.With(middleware.RequirePermission(pool, "agents.runs.read")).Get("/agent-runs", agentHandler.ListRuns)
		r.With(middleware.RequirePermission(pool, "agents.runs.read")).Get("/agent-runs/{runId}", agentHandler.GetRun)

		// Agent Intentions
		r.With(middleware.RequirePermission(pool, "agents.intentions.create")).
			With(middleware.Idempotency(middleware.IdempotencyConfig{Pool: pool, Scope: "user", Expiry: 1 * time.Hour})).
			Post("/agent-runs/{runId}/intentions", agentHandler.CreateIntention)
		r.With(middleware.RequirePermission(pool, "agents.intentions.read")).Get("/agent-runs/{runId}/intentions", agentHandler.ListIntentions)

		// Tool Gateway
		r.With(middleware.RequirePermission(pool, "agents.tools.execute")).
			With(middleware.Idempotency(middleware.IdempotencyConfig{Pool: pool, Scope: "tool-gateway", Expiry: 1 * time.Hour})).
			Post("/tool-gateway/execute", agentHandler.ExecuteTool)


		// ─── v1.0 Track 2: Approval Workflow ───
		approvalHandler := approval.NewHandler(pool, cfg)
		r.Route("/approvals", func(r chi.Router) {
			r.With(middleware.RequirePermission(pool, "approvals.create")).Post("/", approvalHandler.Create)
			r.With(middleware.RequirePermission(pool, "approvals.read")).Get("/", approvalHandler.List)
			r.With(middleware.RequirePermission(pool, "approvals.read")).Get("/{approvalId}", approvalHandler.Get)
			r.With(middleware.RequirePermission(pool, "approvals.approve")).
				With(middleware.Idempotency(middleware.IdempotencyConfig{Pool: pool, Scope: "user", Expiry: 1 * time.Hour})).
				Post("/{approvalId}/approve", approvalHandler.Approve)
			r.With(middleware.RequirePermission(pool, "approvals.approve")).
				With(middleware.Idempotency(middleware.IdempotencyConfig{Pool: pool, Scope: "user", Expiry: 1 * time.Hour})).
				Post("/{approvalId}/reject", approvalHandler.Reject)
			r.With(middleware.RequirePermission(pool, "approvals.create")).
				With(middleware.Idempotency(middleware.IdempotencyConfig{Pool: pool, Scope: "user", Expiry: 1 * time.Hour})).
				Post("/{approvalId}/cancel", approvalHandler.Cancel)
		})

		// ─── Phase 8: Integration Keys ───
		r.With(middleware.RequirePermission(pool, "integrations.keys.create")).
			With(middleware.Idempotency(middleware.IdempotencyConfig{Pool: pool, Scope: "user", Expiry: 1 * time.Hour})).
			Post("/integration-keys", integrationHandler.CreateKey)
		r.With(middleware.RequirePermission(pool, "integrations.keys.read")).Get("/integration-keys", integrationHandler.ListKeys)
		r.With(middleware.RequirePermission(pool, "integrations.keys.revoke")).
			With(middleware.Idempotency(middleware.IdempotencyConfig{Pool: pool, Scope: "user", Expiry: 1 * time.Hour})).
			Delete("/integration-keys/{keyId}", integrationHandler.RevokeKey)
		r.With(middleware.RequirePermission(pool, "integrations.keys.revoke")).
			With(middleware.Idempotency(middleware.IdempotencyConfig{Pool: pool, Scope: "user", Expiry: 1 * time.Hour})).
			Post("/integration-keys/{keyId}/rotate", integrationHandler.RotateKey)

		// ─── Phase 8: Proxmox Integration ───
		var pxCtxClient proxmox.ProxmoxClient
		if cfg.ProxmoxEnabled {
			pxCtxClient = proxmox.NewRealProxmoxClient(cfg.ProxmoxURL, cfg.ProxmoxTokenID, cfg.ProxmoxSecret, cfg.ProxmoxVerifyTLS)
		} else {
			pxCtxClient = &proxmox.FakeProxmoxClient{}
		}
		proxmoxHandler := proxmox.NewHandler(pool, pxCtxClient)
		r.Route("/integrations/proxmox", func(r chi.Router) {
			r.With(middleware.RequirePermission(pool, "integrations.proxmox.read")).Get("/status", proxmoxHandler.Status)
			r.With(middleware.RequirePermission(pool, "integrations.proxmox.sync")).Post("/sync", proxmoxHandler.Sync)
		})

		// Phase 8 + v1.0: Assets + Proxmox Controlled Mutation Pipeline
		actionHandler := proxmox.NewActionHandler(pool, pxCtxClient, cfg)
		r.Route("/assets", func(r chi.Router) {
			// Asset list and detail (inside route group to avoid chi shadowing)
			r.With(middleware.RequirePermission(pool, "assets.read")).Get("/", func(w http.ResponseWriter, r *http.Request) {
				rows, _ := pool.Query(r.Context(), `SELECT o.id::text, a.asset_type, a.provider, a.external_id, a.hostname, o.status, o.created_at FROM assets a JOIN objects o ON a.object_id=o.id WHERE o.team_id=$1`, chi.URLParam(r, "teamId"))
				defer rows.Close()
				var out []map[string]any
				for rows.Next() {
					var id, at, prov, eid, host, st string; var c time.Time
					rows.Scan(&id, &at, &prov, &eid, &host, &st, &c)
					out = append(out, map[string]any{"id": id, "asset_type": at, "provider": prov, "external_id": eid, "hostname": host, "status": st, "created_at": c})
				}
				if out == nil { out = []map[string]any{} }
				writeJSON(w, 200, out)
			})
			r.With(middleware.RequirePermission(pool, "assets.read")).Get("/{assetId}", func(w http.ResponseWriter, r *http.Request) {
				var id, at, prov, eid, host, st string; var c time.Time
				err := pool.QueryRow(r.Context(), `SELECT o.id::text, a.asset_type, a.provider, a.external_id, a.hostname, o.status, o.created_at FROM assets a JOIN objects o ON a.object_id=o.id WHERE o.id=$1 AND o.team_id=$2`, chi.URLParam(r, "assetId"), chi.URLParam(r, "teamId")).Scan(&id, &at, &prov, &eid, &host, &st, &c)
				if err != nil { writeJSON(w, 404, map[string]string{"detail": "not found"}); return }
				writeJSON(w, 200, map[string]any{"id": id, "asset_type": at, "provider": prov, "external_id": eid, "hostname": host, "status": st, "created_at": c})
			})
			// Proxmox mutation actions
			r.With(middleware.RequirePermission(pool, "assets.actions.create")).
				Post("/{assetId}/actions/proxmox/start", actionHandler.CreateAction)
			r.With(middleware.RequirePermission(pool, "assets.actions.create")).
				Post("/{assetId}/actions/proxmox/shutdown", actionHandler.CreateAction)
			r.With(middleware.RequirePermission(pool, "assets.actions.create")).
				Post("/{assetId}/actions/proxmox/stop", actionHandler.CreateAction)
			r.With(middleware.RequirePermission(pool, "assets.actions.create")).
				Post("/{assetId}/actions/proxmox/snapshot", actionHandler.CreateAction)
			// v1.2 Track 4: Change-Risk Scoring
			riskScoreHandler := proxmox.NewRiskScoreHandler(pool, cfg)
			r.With(middleware.RequirePermission(pool, "assets.read")).
				Get("/{assetId}/risk-score", riskScoreHandler.GetRiskScore)

			// v1.2 Track 5: Post-Action Outcome Tracking
			outcomeHandler := proxmox.NewOutcomeHandler(pool, cfg)
			r.With(middleware.RequirePermission(pool, "assets.actions.read")).
				Post("/asset-actions/{actionId}/outcome", outcomeHandler.CreateOrUpdateAssetActionOutcome)
			r.With(middleware.RequirePermission(pool, "assets.actions.read")).
				Get("/asset-actions/{actionId}/outcome", outcomeHandler.GetAssetActionOutcome)

			r.With(middleware.RequirePermission(pool, "assets.actions.read")).Get("/asset-actions", actionHandler.ListActions)
			r.With(middleware.RequirePermission(pool, "assets.actions.read")).Get("/asset-actions/{actionId}", actionHandler.GetAction)
			r.With(middleware.RequirePermission(pool, "assets.actions.execute")).
				With(middleware.Idempotency(middleware.IdempotencyConfig{Pool: pool, Scope: "tool-gateway", Expiry: 1 * time.Hour})).
				Post("/asset-actions/{actionId}/execute", actionHandler.ExecuteAction)
		})

		// ─── Phase 8: Object Attachments ───
		storageHandler := storage.NewHandler(pool, s3Client, cfg.MinioBucket)
		r.Route("/objects/{objectId}/attachments", func(r chi.Router) {
			r.With(middleware.RequirePermission(pool, "objects.attachments.create")).Post("/", storageHandler.Upload)
			r.With(middleware.RequirePermission(pool, "objects.attachments.read")).Get("/", storageHandler.List)
			r.With(middleware.RequirePermission(pool, "objects.attachments.read")).Get("/{attachmentId}/download-url", storageHandler.DownloadURL)
		})

		// ─── v1.0 Track 5: Remediation ───
		remediationHandler := remediation.NewHandler(pool)
		r.Route("/remediations", func(r chi.Router) {
			r.With(middleware.RequirePermission(pool, "remediations.create")).
				With(middleware.Idempotency(middleware.IdempotencyConfig{Pool: pool, Scope: "user", Expiry: 1 * time.Hour})).
				Post("/", remediationHandler.Create)
			r.With(middleware.RequirePermission(pool, "remediations.read")).Get("/", remediationHandler.List)
			r.With(middleware.RequirePermission(pool, "remediations.read")).Get("/{remediationId}", remediationHandler.Get)
			r.With(middleware.RequirePermission(pool, "remediations.approve")).
				With(middleware.Idempotency(middleware.IdempotencyConfig{Pool: pool, Scope: "user", Expiry: 1 * time.Hour})).
				Post("/{remediationId}/approve", remediationHandler.Approve)
			r.With(middleware.RequirePermission(pool, "remediations.execute")).
				With(middleware.Idempotency(middleware.IdempotencyConfig{Pool: pool, Scope: "user", Expiry: 1 * time.Hour})).
				Post("/{remediationId}/execute", remediationHandler.Execute)
			r.With(middleware.RequirePermission(pool, "remediations.cancel")).
				With(middleware.Idempotency(middleware.IdempotencyConfig{Pool: pool, Scope: "user", Expiry: 1 * time.Hour})).
				Post("/{remediationId}/cancel", remediationHandler.Cancel)

			// v1.2 Track 5: Post-Action Outcome Tracking (remediation)
			remOutcomeHandler := proxmox.NewOutcomeHandler(pool, cfg)
			r.With(middleware.RequirePermission(pool, "remediations.read")).
				Post("/{remediationId}/outcome", remOutcomeHandler.CreateOrUpdateRemediationOutcome)
			r.With(middleware.RequirePermission(pool, "remediations.read")).
				Get("/{remediationId}/outcome", remOutcomeHandler.GetRemediationOutcome)
		})

		// v1.2 Track 1: Recommendation Evidence Packs
		evidenceHandler := remediation.NewEvidenceHandler(pool)
		r.Route("/recommendations", func(r chi.Router) {
			r.With(middleware.RequirePermission(pool, "remediations.read")).
				Get("/{recommendationId}/evidence", evidenceHandler.GetEvidence)
		})

		// v1.2 Track 6: Context Graph Quality Controls
		qualityHandler := contextx.NewQualityHandler(pool)
		r.Route("/context", func(r chi.Router) {
			r.With(middleware.RequirePermission(pool, "assets.read")).
				Get("/quality", qualityHandler.GetQuality)
			r.With(middleware.RequirePermission(pool, "assets.read")).
				Post("/relations/{relationId}/confirm", qualityHandler.ConfirmRelation)
			r.With(middleware.RequirePermission(pool, "assets.read")).
				Post("/relations/{relationId}/dismiss", qualityHandler.DismissRelation)
		})

		// v1.5 Knowledge (handler declared in outer scope)
		artifactHandler := artifact.NewHandler(pool)
		// Wire S3 storage for download/export endpoints (Track 7)
		if s3Client != nil {
			artifactHandler.SetStorage(s3Client, cfg.MinioBucket)
		}
		// v1.4 Track 3: Wire worker assist for document-assist
		workerAssistURL := os.Getenv("WORKER_ASSIST_URL")
		if workerAssistURL != "" {
			artifactHandler.SetWorkerAssist(artifact.WorkerAssistConfig{
				URL:   workerAssistURL,
				Token: os.Getenv("WORKER_TOKEN"),
			})
		}

		// v1.5 Knowledge index hook: re-extract and index on mutation
		kIndexer := knowledge.NewIndexer(pool)
		artifactHandler.SetIndexHook(func(ctx context.Context, teamID, sourceType, sourceID string) {
			go func() {
				var docs []knowledge.SourceDocument
				var err error
				switch sourceType {
				case "clarity_document":
					docs, err = knowledge.ExtractClarityDocuments(ctx, pool, teamID)
				case "meeting_summary":
					docs, err = knowledge.ExtractMeetingSummaries(ctx, pool, teamID)
				case "template":
					docs, err = knowledge.ExtractTemplates(ctx, pool, teamID)
				default:
					docs, err = knowledge.ExtractArtifacts(ctx, pool, teamID)
				}
				if err != nil {
					return
				}
				for _, doc := range docs {
					if doc.SourceID == sourceID {
						kIndexer.IndexSource(ctx, doc)
						break
					}
				}
			}()
		})
		// v1.3 Track 2: Presenton handler
		presentonClient := presenton.NewClient(cfg.PresentonURL, cfg.PresentonAdminUser, cfg.PresentonAdminPass, cfg.PresentonGenerationTimeout)
		presentonCfg := presenton.Config{
			Enabled:      cfg.PresentonEnabled,
			URL:          cfg.PresentonURL,
			AdminUser:    cfg.PresentonAdminUser,
			AdminPass:    cfg.PresentonAdminPass,
			Timeout:      cfg.PresentonGenerationTimeout,
			MaxFileBytes: cfg.PresentonMaxFileBytes,
		}
		presentonHandler := presenton.NewHandler(pool, presentonClient, s3Client, cfg.MinioBucket, presentonCfg)
		r.Route("/artifacts", func(r chi.Router) {
			r.With(middleware.RequirePermission(pool, "artifacts.create")).
				Post("/", artifactHandler.Create)
			r.With(middleware.RequirePermission(pool, "artifacts.read")).
				Get("/", artifactHandler.List)
			r.With(middleware.RequirePermission(pool, "artifacts.read")).
				Get("/{artifactId}", artifactHandler.Get)
			r.With(middleware.RequirePermission(pool, "artifacts.update")).
				Patch("/{artifactId}", artifactHandler.Patch)
			r.With(middleware.RequirePermission(pool, "artifacts.delete")).
				Delete("/{artifactId}", artifactHandler.Delete)

			// v1.3 Track 2: Presenton
			r.With(middleware.RequirePermission(pool, "artifacts.read")).
				Get("/presenton/status", presentonHandler.Status)
			r.With(middleware.RequirePermission(pool, "artifacts.create")).
				Post("/generate-presentation", presentonHandler.Generate)

			// v1.3 Track 3: Meeting Summaries
			r.With(middleware.RequirePermission(pool, "artifacts.create")).
				Post("/meeting-summaries", artifactHandler.CreateMeetingSummary)
			r.With(middleware.RequirePermission(pool, "artifacts.read")).
				Get("/meeting-summaries", artifactHandler.ListMeetingSummaries)
			r.With(middleware.RequirePermission(pool, "artifacts.read")).
				Get("/meeting-summaries/{id}", artifactHandler.GetMeetingSummary)
			r.With(middleware.RequirePermission(pool, "artifacts.update")).
				Patch("/meeting-summaries/{id}", artifactHandler.PatchMeetingSummary)

			// v1.3 Track 4: Status Report Generator
			r.With(middleware.RequirePermission(pool, "artifacts.create")).
				Post("/status-reports/generate", artifactHandler.GenerateStatusReport)

			// v1.3 Track 5: Template Library
			r.With(middleware.RequirePermission(pool, "artifacts.read")).
				Get("/artifact-templates", artifactHandler.ListTemplates)
			r.With(middleware.RequirePermission(pool, "artifacts.create")).
				Post("/artifact-templates", artifactHandler.CreateTemplate)
			r.With(middleware.RequirePermission(pool, "artifacts.create")).
				Post("/artifact-templates/{templateId}/instantiate", artifactHandler.InstantiateTemplate)

			// v1.3 Track 6: Artifact Storage and Recent Files
			r.With(middleware.RequirePermission(pool, "artifacts.read")).
				Get("/recent", artifactHandler.Recent)
			r.With(middleware.RequirePermission(pool, "artifacts.read")).
				Get("/search", artifactHandler.Search)
			r.With(middleware.RequirePermission(pool, "artifacts.read")).
				Get("/storage-summary", artifactHandler.StorageSummary)

			// v1.3 Track 7: Download and Export
			r.With(middleware.RequirePermission(pool, "artifacts.read")).
				Get("/{artifactId}/download", artifactHandler.Download)
			r.With(middleware.RequirePermission(pool, "artifacts.read")).
				Get("/{artifactId}/export/markdown", artifactHandler.ExportMarkdown)
			r.With(middleware.RequirePermission(pool, "artifacts.read")).
				Get("/{artifactId}/export/pdf", artifactHandler.ExportPDF)

			// v1.4 Track 1: Native Document Artifacts
			r.With(middleware.RequirePermission(pool, "artifacts.create")).
				Post("/documents", artifactHandler.CreateDocument)
			r.With(middleware.RequirePermission(pool, "artifacts.read")).
				Get("/documents", artifactHandler.ListDocuments)
			r.With(middleware.RequirePermission(pool, "artifacts.read")).
				Get("/documents/{artifactId}", artifactHandler.GetDocument)
			r.With(middleware.RequirePermission(pool, "artifacts.update")).
				Patch("/documents/{artifactId}", artifactHandler.PatchDocument)
			// v1.4 Track 3: Agent Assist
			r.With(middleware.RequirePermission(pool, "artifacts.update")).
				Post("/documents/{artifactId}/document-assist", artifactHandler.DocumentAssist)
			// v1.4 Track 4: Document Generation
			r.With(middleware.RequirePermission(pool, "artifacts.create")).
				Post("/artifacts/generate-document", artifactHandler.GenerateDocument)
			// v1.4 Track 6: DOCX Export
			r.With(middleware.RequirePermission(pool, "artifacts.read")).
				Get("/{artifactId}/export/docx", artifactHandler.ExportDOCX)
			// v1.4 Track 7: Document Version History
			r.With(middleware.RequirePermission(pool, "artifacts.read")).
				Get("/documents/{artifactId}/versions", artifactHandler.ListVersions)
			r.With(middleware.RequirePermission(pool, "artifacts.read")).
				Get("/documents/{artifactId}/versions/{versionId}", artifactHandler.GetVersion)
			r.With(middleware.RequirePermission(pool, "artifacts.update")).
				Post("/documents/{artifactId}/versions/{versionId}/restore", artifactHandler.RestoreVersion)
		})

		// v1.5 Knowledge
		r.With(middleware.RequirePermission(pool, "knowledge.search")).
			Get("/knowledge/search", knowledgeHandler.SearchHTTP)
		r.With(middleware.RequirePermission(pool, "knowledge.read")).
			Get("/knowledge/index-status", knowledgeHandler.IndexStatusHTTP)
		r.With(middleware.RequirePermission(pool, "knowledge.read")).
			Get("/knowledge/{itemId}", knowledgeHandler.GetHTTP)
	})

	// ─── Platform Admin ───
	r.Route("/api/admin", func(r chi.Router) {
		r.Use(middleware.RequireAuth)
		r.Use(middleware.RequirePlatformRole(pool, "platform_owner"))
		r.Get("/users", adminHandler.ListUsers)
		r.Get("/users/{id}", adminHandler.GetUser)
		r.With(middleware.Idempotency(middleware.IdempotencyConfig{Pool: pool, Scope: "user", Expiry: 1 * time.Hour})).
			Patch("/users/{id}", adminHandler.UpdateUser)
		r.Get("/teams", adminHandler.ListTeams)
		r.Get("/audit", adminHandler.ListAudit)
		r.Get("/settings", adminHandler.GetSettings)
		r.Get("/setup-status", adminHandler.SetupStatus)
		r.With(middleware.Idempotency(middleware.IdempotencyConfig{Pool: pool, Scope: "user", Expiry: 1 * time.Hour})).
			Patch("/settings", adminHandler.UpdateSettings)

		// v1.5 Knowledge admin
		r.Post("/knowledge/reindex", knowledgeHandler.AdminReindexHTTP)
		r.Get("/knowledge/index-status", knowledgeHandler.AdminIndexStatusAllHTTP)

		// v1.1 Track 2: Proxmox Mutation Change-Window
		mwHandler := proxmox.NewMutationWindowHandler(pool, cfg)
		r.Post("/proxmox/mutation-window", mwHandler.OpenWindow)
		r.Get("/proxmox/mutation-window", mwHandler.GetActiveWindow)
		r.Post("/proxmox/mutation-window/{windowId}/close", mwHandler.CloseWindow)

		// Ops dashboard (read-only)
		r.Route("/ops", func(r chi.Router) {
			r.Get("/summary", opsHandler.Summary)
			r.Get("/outbox", opsHandler.Outbox)
			r.Get("/dead-letters", opsHandler.DeadLetters)
			r.Get("/workers", opsHandler.Workers)
			r.Get("/webhooks/rejections", opsHandler.WebhookRejections)
			r.Get("/agent-blocks", opsHandler.AgentBlocks)
		})

		// v1.1 Track 3: Backup Status (read-only)
		backupStatusHandler := admin.NewBackupStatusHandler(pool)
		r.Get("/backup-status", backupStatusHandler.GetBackupStatus)

		// v1.1 Track 7: Operational Metrics (read-only)
		metricsHandler := admin.NewMetricsHandler(pool)
		r.Get("/metrics", metricsHandler.Metrics)

		// v1.2 Track 3: Approval Policy Simulation
		simHandler := approval.NewSimulationHandler(pool)
		r.Post("/approval-policy/simulate", simHandler.Simulate)

		// v1.2 Track 7: Agent Recommendation Evaluation Harness
		evalHandler := agent.NewEvalHandler(pool)
		r.Route("/agent-evaluation", func(r chi.Router) {
			r.Post("/run", evalHandler.RunEvaluation)
			r.Get("/results", evalHandler.GetLatestResults)
			r.Get("/runs/{runId}", evalHandler.GetRunDetail)
		})
	})

	srv := &http.Server{Addr: fmt.Sprintf(":%s", cfg.Port), Handler: r}

	// Start approval expiry monitor
	approvalMonitor := approval.NewMonitor(pool, cfg)
	go approvalMonitor.Start(context.Background())

	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		log.Println("Shutting down...")
		srv.Shutdown(context.Background())
	}()
	log.Printf("ClarityIT API listening on :%s", cfg.Port)
	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatalf("Server error: %v", err)
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
