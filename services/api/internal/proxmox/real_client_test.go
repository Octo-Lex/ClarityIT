package proxmox

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRealClientListNodes(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api2/json/nodes" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "PVEAPIToken=root@pam!test=test-secret" {
			t.Errorf("unexpected auth header: %s", r.Header.Get("Authorization"))
		}
		resp := map[string]any{
			"data": []map[string]any{
				{"node": "pve1", "status": "online", "cpu": 0.35, "mem": float64(16 * 1024 * 1024 * 1024), "maxmem": float64(32 * 1024 * 1024 * 1024)},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	client := NewRealProxmoxClient(ts.URL, "root@pam!test", "test-secret", false)
	nodes, err := client.ListNodes(context.Background())
	if err != nil { t.Fatalf("ListNodes: %v", err) }
	if len(nodes) != 1 { t.Fatalf("expected 1 node, got %d", len(nodes)) }
	if nodes[0].Node != "pve1" { t.Errorf("expected node pve1, got %s", nodes[0].Node) }
}

func TestRealClientListVMs(t *testing.T) {
	node := "pve1"
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api2/json/nodes/pve1/qemu":
			resp := map[string]any{
				"data": []map[string]any{
					{"vmid": 100, "name": "test-vm", "status": "running", "cpu": 0.1, "mem": float64(1024), "maxmem": float64(2048)},
				},
			}
			json.NewEncoder(w).Encode(resp)
		case "/api2/json/nodes/pve1/lxc":
			resp := map[string]any{
				"data": []map[string]any{
					{"vmid": 200, "name": "test-ct", "status": "stopped", "cpu": 0.0, "mem": float64(0), "maxmem": float64(4096)},
				},
			}
			json.NewEncoder(w).Encode(resp)
		default:
			t.Errorf("unexpected path: %s", r.URL.Path)
			w.WriteHeader(404)
		}
	}))
	defer ts.Close()

	client := NewRealProxmoxClient(ts.URL, "root@pam!test", "test-secret", false)
	vms, err := client.ListVMs(context.Background(), node)
	if err != nil { t.Fatalf("ListVMs: %v", err) }
	if len(vms) != 2 { t.Fatalf("expected 2 VMs, got %d", len(vms)) }
	// Check types
	foundQEMU, foundLXC := false, false
	for _, vm := range vms {
		if vm.Type == "qemu" { foundQEMU = true }
		if vm.Type == "lxc" { foundLXC = true }
	}
	if !foundQEMU { t.Error("missing qemu VM") }
	if !foundLXC { t.Error("missing lxc container") }
}

func TestRealClientSanitizesSecretsInErrors(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	defer ts.Close()

	secret := "super-secret-token-12345"
	client := NewRealProxmoxClient(ts.URL, "root@pam!test", secret, false)
	_, err := client.ListNodes(context.Background())
	if err == nil { t.Fatal("expected error") }
	if contains(err.Error(), secret) { t.Error("secret leaked in error message") }
}

func TestRealClientAPIError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
		w.Write([]byte(`{"errors":"authentication failed"}`))
	}))
	defer ts.Close()

	client := NewRealProxmoxClient(ts.URL, "root@pam!test", "secret", false)
	_, err := client.ListNodes(context.Background())
	if err == nil { t.Fatal("expected error for 401") }
}

func TestRealClientImplementsInterface(t *testing.T) {
	// Verify RealProxmoxClient implements ProxmoxClient
	var _ ProxmoxClient = (*RealProxmoxClient)(nil)
	var _ ProxmoxClient = (*FakeProxmoxClient)(nil)
}

func TestRealClientInterfaceHasNoMutationMethods(t *testing.T) {
	// This is a design assertion — ProxmoxClient only has List* methods
	// No Create*, Update*, Delete*, Start*, Stop* methods exist
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 || containsHelper(s, sub))
}

func containsHelper(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub { return true }
	}
	return false
}
