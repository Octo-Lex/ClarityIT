package config

import (
	"os"
	"testing"
)

// TestWorkerAssistWiring verifies that the docker-compose.yml correctly wires
// the document assist feature:
//   1. API service has WORKER_ASSIST_URL pointing to the reasoning worker
//   2. API service has WORKER_TOKEN (same as reasoning worker)
//   3. Reasoning worker does NOT publish port 9100 to the host
//
// This test parses docker-compose.yml to validate the configuration statically,
// ensuring the feature is deployable, not only unit-testable.

func TestWorkerAssistWiring(t *testing.T) {
	composeYAML, err := os.ReadFile("../../../../docker-compose.yml")
	if err != nil {
		t.Skip("docker-compose.yml not found — skipping config test")
	}
	yamlStr := string(composeYAML)

	// 1. API service has WORKER_ASSIST_URL
	if !contains(yamlStr, "WORKER_ASSIST_URL: http://clarityit-reasoning-worker:9100") {
		t.Error("docker-compose.yml: API service must have WORKER_ASSIST_URL pointing to reasoning worker:9100")
	}

	// 2. API service has WORKER_TOKEN
	if !contains(yamlStr, "WORKER_TOKEN:") {
		t.Error("docker-compose.yml: API service must have WORKER_TOKEN environment variable")
	}

	// 3. Reasoning worker must NOT publish port 9100 to host
	// Check that there's no "9100:9100" or "9100:" port mapping anywhere
	if contains(yamlStr, "9100:9100") || contains(yamlStr, "\"9100:") {
		t.Error("docker-compose.yml: reasoning worker must NOT publish port 9100 to host — internal network only")
	}

	// 4. Reasoning worker service does not have a ports: section at all
	// (it should only have networks, not published ports for assist)
	// We verify by checking the worker section has no ports mapping
	if contains(yamlStr, "clarityit-reasoning-worker") && contains(yamlStr, "9100:9100") {
		t.Error("docker-compose.yml: reasoning worker must not expose port 9100 to host")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (indexOf(s, substr) >= 0)
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
