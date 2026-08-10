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

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/Octo-Lex/ClarityIT/services/api/cmd/api/handlers/admin"
	"github.com/Octo-Lex/ClarityIT/services/api/cmd/api/handlers/agent"
	"github.com/Octo-Lex/ClarityIT/services/api/cmd/api/handlers/approval"
	"github.com/Octo-Lex/ClarityIT/services/api/cmd/api/handlers/artifact"
	"github.com/Octo-Lex/ClarityIT/services/api/cmd/api/handlers/knowledge"
	"github.com/Octo-Lex/ClarityIT/services/api/cmd/api/handlers/ops"
	"github.com/Octo-Lex/ClarityIT/services/api/cmd/api/handlers/proxmox"
	"github.com/Octo-Lex/ClarityIT/services/api/cmd/api/middleware"
	"github.com/Octo-Lex/ClarityIT/services/api/internal/config"
	"github.com/Octo-Lex/ClarityIT/services/api/internal/storage"
)

 artifactHandler.Download)
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
		Get("/knowledge/related", knowledgeHandler.RelatedHTTP)
	r.With(middleware.RequirePermission(pool, "knowledge.ask")).
		Post("/knowledge/ask", knowledgeHandler.AskHTTP)

	// v1.5 Track 6: Knowledge Collections
	r.With(middleware.RequirePermission(pool, "knowledge.collections.read")).
		Get("/knowledge/collections", knowledgeHandler.ListCollections)
	r.With(middleware.RequirePermission(pool, "knowledge.collections.create")).
		Post("/knowledge/collections", knowledgeHandler.CreateCollection)
	r.With(middleware.RequirePermission(pool, "knowledge.collections.read")).
		Get("/knowledge/collections/{collectionId}", knowledgeHandler.GetCollection)
	r.With(middleware.RequirePermission(pool, "knowledge.collections.update")).
		Patch("/knowledge/collections/{collectionId}", knowledgeHandler.PatchCollection)
	r.With(middleware.RequirePermission(pool, "knowledge.collections.delete")).
		Delete("/knowledge/collections/{collectionId}", knowledgeHandler.DeleteCollection)
	r.With(middleware.RequirePermission(pool, "knowledge.collections.update")).
		Post("/knowledge/collections/{collectionId}/items", knowledgeHandler.AddItem)
	r.With(middleware.RequirePermission(pool, "knowledge.collections.update")).
		Delete("/knowledge/collections/{collectionId}/items/{itemId}", knowledgeHandler.RemoveItem)

	// v1.5 Track 6: Saved Answers
	r.With(middleware.RequirePermission(pool, "knowledge.collections.update")).
		Post("/knowledge/saved-answers", knowledgeHandler.SaveAnswer)
	r.With(middleware.RequirePermission(pool, "knowledge.collections.read")).
		Get("/knowledge/saved-answers", knowledgeHandler.ListSavedAnswers)
	r.With(middleware.RequirePermission(pool, "knowledge.collections.read")).
		Get("/knowledge/saved-answers/{answerId}", knowledgeHandler.GetSavedAnswer)
	r.With(middleware.RequirePermission(pool, "knowledge.collections.delete")).
		Delete("/knowledge/saved-answers/{answerId}", knowledgeHandler.DeleteSavedAnswer)

	// v1.5 Track 7: Quality Controls (read-only)
	r.With(middleware.RequirePermission(pool, "knowledge.read")).
		Get("/knowledge/quality", knowledgeHandler.QualityReportHTTP)
	r.With(middleware.RequirePermission(pool, "knowledge.read")).
		Get("/knowledge/quality/stale", knowledgeHandler.StaleItemsHTTP)
	r.With(middleware.RequirePermission(pool, "knowledge.read")).
		Get("/knowledge/quality/duplicates", knowledgeHandler.DuplicateItemsHTTP)
	r.With(middleware.RequirePermission(pool, "knowledge.read")).
		Get("/knowledge/quality/orphans", knowledgeHandler.OrphanItemsHTTP)

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
