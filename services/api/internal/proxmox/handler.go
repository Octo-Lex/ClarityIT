package proxmox

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/clarityit/api/internal/audit"
	"github.com/clarityit/api/internal/database"
	"github.com/clarityit/api/internal/iam"
	"github.com/clarityit/api/internal/outbox"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// MutationTarget carries the identifying information for a Proxmox VM/LXC
// that is the target of a mutation action.
type MutationTarget struct {
	Node   string
	VMID   int
	VMType string // "qemu" or "lxc"
}

// TaskStatus represents the result of a Proxmox async task.
type TaskStatus struct {
	Status   string // "running", "stopped"
	ExitCode string // "OK", "ERROR", etc.
	Output   string
}

// ProxmoxClient is the interface for Proxmox API operations.
// v1.0 adds controlled mutation methods (start, shutdown, stop, snapshot).
type ProxmoxClient interface {
	// Read methods
	ListNodes(ctx context.Context) ([]ProxmoxNode, error)
	ListVMs(ctx context.Context, node string) ([]ProxmoxVM, error)

	// Mutation methods (v1.0 — all require approved approval_request + MFA)
	StartVM(ctx context.Context, target MutationTarget) (taskUPID string, err error)
	ShutdownVM(ctx context.Context, target MutationTarget) (taskUPID string, err error)
	StopVM(ctx context.Context, target MutationTarget) (taskUPID string, err error)
	SnapshotVM(ctx context.Context, target MutationTarget, snapName string) (taskUPID string, err error)
	GetTaskStatus(ctx context.Context, node string, taskID string) (*TaskStatus, error)
}

// ProxmoxNode represents a Proxmox node.
type ProxmoxNode struct {
	Node   string `json:"node"`
	Status string `json:"status"`
	CPU    float64 `json:"cpu"`
	Mem    int64   `json:"mem"`
	MaxMem int64   `json:"maxmem"`
}

// ProxmoxVM represents a VM or container.
type ProxmoxVM struct {
	VMID   int    `json:"vmid"`
	Name   string `json:"name"`
	Status string `json:"status"`
	Type   string `json:"type"` // "qemu" or "lxc"
	Node   string `json:"node"`
	CPU    float64 `json:"cpu"`
	Mem    int64   `json:"mem"`
	MaxMem int64   `json:"maxmem"`
}

// FakeProxmoxClient returns test data without connecting to Proxmox.
type FakeProxmoxClient struct{}

func (f *FakeProxmoxClient) ListNodes(ctx context.Context) ([]ProxmoxNode, error) {
	return []ProxmoxNode{
		{Node: "pve1", Status: "online", CPU: 0.35, Mem: 16 * 1024 * 1024 * 1024, MaxMem: 32 * 1024 * 1024 * 1024},
	}, nil
}

func (f *FakeProxmoxClient) ListVMs(ctx context.Context, node string) ([]ProxmoxVM, error) {
	return []ProxmoxVM{
		{VMID: 100, Name: "clarityit", Status: "running", Type: "lxc", Node: node, CPU: 0.12, Mem: 2 * 1024 * 1024 * 1024, MaxMem: 4 * 1024 * 1024 * 1024},
		{VMID: 101, Name: "monitoring", Status: "running", Type: "qemu", Node: node, CPU: 0.08, Mem: 1 * 1024 * 1024 * 1024, MaxMem: 2 * 1024 * 1024 * 1024},
	}, nil
}

// ─── Fake mutation methods ───

func (f *FakeProxmoxClient) StartVM(ctx context.Context, target MutationTarget) (string, error) {
	return fmt.Sprintf("UPID:pve:%s:fake:start:%d::", target.Node, target.VMID), nil
}

func (f *FakeProxmoxClient) ShutdownVM(ctx context.Context, target MutationTarget) (string, error) {
	return fmt.Sprintf("UPID:pve:%s:fake:shutdown:%d::", target.Node, target.VMID), nil
}

func (f *FakeProxmoxClient) StopVM(ctx context.Context, target MutationTarget) (string, error) {
	return fmt.Sprintf("UPID:pve:%s:fake:stop:%d::", target.Node, target.VMID), nil
}

func (f *FakeProxmoxClient) SnapshotVM(ctx context.Context, target MutationTarget, snapName string) (string, error) {
	return fmt.Sprintf("UPID:pve:%s:fake:snapshot:%d:%s:", target.Node, target.VMID, snapName), nil
}

func (f *FakeProxmoxClient) GetTaskStatus(ctx context.Context, node string, taskID string) (*TaskStatus, error) {
	return &TaskStatus{Status: "stopped", ExitCode: "OK"}, nil
}

// Handler for Proxmox integration
type Handler struct {
	pool   *pgxpool.Pool
	client ProxmoxClient
}

func NewHandler(pool *pgxpool.Pool, client ProxmoxClient) *Handler {
	return &Handler{pool: pool, client: client}
}

func claims(r *http.Request) (*iam.TokenClaims, bool) { return iam.GetClaims(r) }

func Routes(h *Handler) chi.Router {
	r := chi.NewRouter()
	r.Get("/status", h.Status)
	r.Post("/sync", h.Sync)
	return r
}

func (h *Handler) Status(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	teamID := chi.URLParam(r, "teamId")

	var syncCount int
	h.pool.QueryRow(ctx, `SELECT COUNT(*) FROM audit_logs WHERE action='integration.proxmox.sync_completed' AND team_id=$1`, teamID).Scan(&syncCount)

	writeJSON(w, 200, map[string]any{
		"connected":   h.client != nil,
		"sync_count":  syncCount,
		"mode":         h.clientMode(),
	})
}

func (h *Handler) clientMode() string {
	switch h.client.(type) {
	case *RealProxmoxClient:
		return "real"
	case *FakeProxmoxClient:
		return "fake"
	default:
		return "unknown"
	}
}

func (h *Handler) Sync(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	teamID := chi.URLParam(r, "teamId")
	cl, ok := claims(r)
	if !ok { writeErr(w, 401, "unauthorized"); return }
	actorID, _ := uuid.Parse(cl.UserID)
	tid, _ := uuid.Parse(teamID)

	nodes, err := h.client.ListNodes(ctx)
	if err != nil { writeErr(w, 502, "Proxmox connection failed"); return }

	synced := 0
	err = database.WithTx(ctx, h.pool, func(ctx context.Context, tx pgx.Tx) error {
		// Sync event
		meta, _ := json.Marshal(map[string]any{"nodes": len(nodes)})
		if err := audit.Write(ctx, tx, audit.Event{TeamID: &tid, ActorID: actorID, Action: "integration.proxmox.sync_started", EntityType: "integration_proxmox", NewValue: meta}); err != nil { return fmt.Errorf("audit write: %w", err) }
		if err := outbox.Write(ctx, tx, &teamID, outbox.Event{EventType: "clarity.v1.integration.proxmox.sync_started", AggregateType: "integration_proxmox", AggregateID: "00000000-0000-0000-0000-000000000001", Payload: meta}); err != nil { return fmt.Errorf("outbox: %w", err) }

		for _, node := range nodes {
			vms, vmErr := h.client.ListVMs(ctx, node.Node)
			if vmErr != nil { continue }

			for _, vm := range vms {
				// Upsert asset via objects table
				var objID string
				extID := fmt.Sprintf("pve:%s:%d", vm.Node, vm.VMID)

				// Check if asset exists
				err := tx.QueryRow(ctx, `SELECT object_id FROM assets WHERE external_id=$1 AND object_id IN (SELECT id FROM objects WHERE team_id=$2)`, extID, tid).Scan(&objID)
				if err != nil {
					// Create new
										if err := tx.QueryRow(ctx, `INSERT INTO objects (team_id, object_type, title, status) VALUES ($1,'asset',$2,'active') RETURNING id::text`, tid, vm.Name).Scan(&objID); err != nil { continue }
					oid, _ := uuid.Parse(objID)
					tx.Exec(ctx, `INSERT INTO assets (object_id, asset_type, provider, external_id, hostname) VALUES ($1,$4,'proxmox',$2,$3)`, oid, extID, vm.Name, vm.Type)

					evMeta, _ := json.Marshal(map[string]any{"hostname": vm.Name, "type": vm.Type, "provider": "proxmox"})
					_ = audit.Write(ctx, tx, audit.Event{TeamID: &tid, ActorID: actorID, Action: "asset.discovered", EntityType: "asset", EntityID: oid, NewValue: evMeta})
					_ = outbox.Write(ctx, tx, &teamID, outbox.Event{EventType: "clarity.v1.asset.discovered", AggregateType: "asset", AggregateID: objID, Payload: evMeta})
				} else {
					// Update existing
					oid, _ := uuid.Parse(objID)
					tx.Exec(ctx, `UPDATE assets SET hostname=$1, asset_type=$3 WHERE object_id=$2`, vm.Name, oid, vm.Type)
					evMeta, _ := json.Marshal(map[string]any{"hostname": vm.Name, "object_id": objID})
					_ = outbox.Write(ctx, tx, &teamID, outbox.Event{EventType: "clarity.v1.asset.updated", AggregateType: "asset", AggregateID: objID, Payload: evMeta})
				}
				synced++
			}
		}

		// Sync completed event
		compMeta, _ := json.Marshal(map[string]any{"synced": synced, "nodes": len(nodes)})
		_ = audit.Write(ctx, tx, audit.Event{TeamID: &tid, ActorID: actorID, Action: "integration.proxmox.sync_completed", EntityType: "integration_proxmox", NewValue: compMeta})
		_ = outbox.Write(ctx, tx, &teamID, outbox.Event{EventType: "clarity.v1.integration.proxmox.sync_completed", AggregateType: "integration_proxmox", AggregateID: "00000000-0000-0000-0000-000000000002", Payload: compMeta})
		return nil
	})
	if err != nil { writeErr(w, 500, fmt.Sprintf("Sync failed: %v", err)); return }
	writeJSON(w, 200, map[string]any{"synced": synced, "nodes": len(nodes)})
}

func writeJSON(w http.ResponseWriter, s int, v any) { w.Header().Set("Content-Type", "application/json"); w.WriteHeader(s); json.NewEncoder(w).Encode(v) }
func writeErr(w http.ResponseWriter, s int, m string) { writeJSON(w, s, map[string]string{"detail": m}) }
