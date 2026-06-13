package proxmox

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// RealProxmoxClient connects to a real Proxmox VE API.
// Only exposes read-only methods as defined by the ProxmoxClient interface.
type RealProxmoxClient struct {
	baseURL    string
	tokenID    string
	secret     string
	verifyTLS  bool
	httpClient *http.Client
}

// NewRealProxmoxClient creates a client for a real Proxmox VE instance.
// The token secret is never logged or exposed in API responses.
func NewRealProxmoxClient(url, tokenID, secret string, verifyTLS bool) *RealProxmoxClient {
	return &RealProxmoxClient{
		baseURL:   strings.TrimRight(url, "/"),
		tokenID:   tokenID,
		secret:    secret,
		verifyTLS: verifyTLS,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: !verifyTLS,
				},
			},
		},
	}
}

// proxmoxAPIResponse is the standard PVE API response wrapper.
type proxmoxAPIResponse struct {
	Data json.RawMessage `json:"data"`
}

// ListNodes returns all cluster nodes.
// API: GET /api2/json/nodes
func (c *RealProxmoxClient) ListNodes(ctx context.Context) ([]ProxmoxNode, error) {
	var nodes []ProxmoxNode
	err := c.get(ctx, "/api2/json/nodes", &nodes)
	if err != nil {
		return nil, fmt.Errorf("list nodes: %w", c.sanitizeError(err))
	}
	return nodes, nil
}

// ListVMs returns all VMs and containers on a node.
// API: GET /api2/json/nodes/{node}/qemu + /lxc
func (c *RealProxmoxClient) ListVMs(ctx context.Context, node string) ([]ProxmoxVM, error) {
	var all []ProxmoxVM

	// QEMU VMs
	var qemu []struct {
		VMID   int     `json:"vmid"`
		Name   string  `json:"name"`
		Status string  `json:"status"`
		CPU    float64 `json:"cpu"`
		Mem    int64   `json:"mem"`
		MaxMem int64   `json:"maxmem"`
	}
	if err := c.get(ctx, fmt.Sprintf("/api2/json/nodes/%s/qemu", node), &qemu); err != nil {
		return nil, fmt.Errorf("list qemu: %w", c.sanitizeError(err))
	}
	for _, vm := range qemu {
		all = append(all, ProxmoxVM{
			VMID: vm.VMID, Name: vm.Name, Status: vm.Status,
			Type: "qemu", Node: node, CPU: vm.CPU, Mem: vm.Mem, MaxMem: vm.MaxMem,
		})
	}

	// LXC containers
	var lxc []struct {
		VMID   int     `json:"vmid"`
		Name   string  `json:"name"`
		Status string  `json:"status"`
		CPU    float64 `json:"cpu"`
		Mem    int64   `json:"mem"`
		MaxMem int64   `json:"maxmem"`
	}
	if err := c.get(ctx, fmt.Sprintf("/api2/json/nodes/%s/lxc", node), &lxc); err != nil {
		return nil, fmt.Errorf("list lxc: %w", c.sanitizeError(err))
	}
	for _, ct := range lxc {
		all = append(all, ProxmoxVM{
			VMID: ct.VMID, Name: ct.Name, Status: ct.Status,
			Type: "lxc", Node: node, CPU: ct.CPU, Mem: ct.Mem, MaxMem: ct.MaxMem,
		})
	}

	return all, nil
}

// get performs an authenticated GET request against the PVE API.
func (c *RealProxmoxClient) get(ctx context.Context, path string, target any) error {
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "PVEAPIToken="+c.tokenID+"="+c.secret)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("PVE API returned %d", resp.StatusCode)
	}

	var apiResp proxmoxAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return fmt.Errorf("decode PVE response: %w", err)
	}

	return json.Unmarshal(apiResp.Data, target)
}

// ─── Mutation methods (v1.0) ───

// post performs an authenticated POST request against the PVE API.
func (c *RealProxmoxClient) post(ctx context.Context, path string, form url.Values) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+path, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "PVEAPIToken="+c.tokenID+"="+c.secret)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", c.sanitizeError(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("PVE API returned %d", resp.StatusCode)
	}

	var apiResp proxmoxAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return "", fmt.Errorf("decode PVE response: %w", err)
	}

	// PVE returns the UPID as a string in the data field
	var upid string
	if err := json.Unmarshal(apiResp.Data, &upid); err != nil {
		return "", fmt.Errorf("decode UPID: %w", err)
	}
	return upid, nil
}

// StartVM starts a stopped VM/LXC.
func (c *RealProxmoxClient) StartVM(ctx context.Context, target MutationTarget) (string, error) {
	path := fmt.Sprintf("/api2/json/nodes/%s/%s/%d/status/start", target.Node, target.VMType, target.VMID)
	upid, err := c.post(ctx, path, url.Values{})
	if err != nil {
		return "", fmt.Errorf("start VM: %w", c.sanitizeError(err))
	}
	return upid, nil
}

// ShutdownVM gracefully shuts down a running VM/LXC.
func (c *RealProxmoxClient) ShutdownVM(ctx context.Context, target MutationTarget) (string, error) {
	path := fmt.Sprintf("/api2/json/nodes/%s/%s/%d/status/shutdown", target.Node, target.VMType, target.VMID)
	upid, err := c.post(ctx, path, url.Values{})
	if err != nil {
		return "", fmt.Errorf("shutdown VM: %w", c.sanitizeError(err))
	}
	return upid, nil
}

// StopVM forcefully stops a running VM/LXC.
func (c *RealProxmoxClient) StopVM(ctx context.Context, target MutationTarget) (string, error) {
	path := fmt.Sprintf("/api2/json/nodes/%s/%s/%d/status/stop", target.Node, target.VMType, target.VMID)
	upid, err := c.post(ctx, path, url.Values{})
	if err != nil {
		return "", fmt.Errorf("stop VM: %w", c.sanitizeError(err))
	}
	return upid, nil
}

// SnapshotVM creates a snapshot of a VM/LXC.
func (c *RealProxmoxClient) SnapshotVM(ctx context.Context, target MutationTarget, snapName string) (string, error) {
	path := fmt.Sprintf("/api2/json/nodes/%s/%s/%d/snapshot", target.Node, target.VMType, target.VMID)
	form := url.Values{}
	form.Set("snapname", snapName)
	upid, err := c.post(ctx, path, form)
	if err != nil {
		return "", fmt.Errorf("snapshot VM: %w", c.sanitizeError(err))
	}
	return upid, nil
}

// GetTaskStatus checks the status of a Proxmox async task.
func (c *RealProxmoxClient) GetTaskStatus(ctx context.Context, node string, taskID string) (*TaskStatus, error) {
	path := fmt.Sprintf("/api2/json/nodes/%s/tasks/%s/status", node, taskID)
	var raw struct {
		Status   string `json:"status"`
		ExitCode string `json:"exitcode"`
		Output   string `json:"output"`
	}
	if err := c.get(ctx, path, &raw); err != nil {
		return nil, fmt.Errorf("task status: %w", c.sanitizeError(err))
	}
	return &TaskStatus{Status: raw.Status, ExitCode: raw.ExitCode, Output: raw.Output}, nil
}
func (c *RealProxmoxClient) sanitizeError(err error) error {
	if err == nil {
		return nil
	}
	msg := err.Error()
	// Remove any accidental token/secret leakage
	if c.secret != "" {
		msg = strings.ReplaceAll(msg, c.secret, "[REDACTED]")
	}
	return fmt.Errorf("%s", msg)
}
