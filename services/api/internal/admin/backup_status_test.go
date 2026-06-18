package admin

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestBackupStatus_RecentBackupGreen(t *testing.T) {
	dir := t.TempDir()

	// Create a recent PostgreSQL backup file
	pgPath := filepath.Join(dir, "postgresql_20260614_100000.sql.gz")
	os.WriteFile(pgPath, []byte("fake pg backup"), 0644)

	// Create a recent MinIO backup file
	minioPath := filepath.Join(dir, "minio_20260614_100200.tar.gz")
	os.WriteFile(minioPath, []byte("fake minio backup"), 0644)

	pgBasename, pgModTime, pgSize, pgFound := findLatestBackup(dir, "postgresql_", ".sql.gz")
	if !pgFound {
		t.Fatal("expected to find PostgreSQL backup")
	}
	if pgBasename != "postgresql_20260614_100000.sql.gz" {
		t.Errorf("expected basename, got %s", pgBasename)
	}
	if pgSize != 14 {
		t.Errorf("expected size 15, got %d", pgSize)
	}
	status := backupAgeStatus(pgModTime, pgFound)
	if status != "green" {
		t.Errorf("expected green, got %s", status)
	}

	_, minioModTime, _, minioFound := findLatestBackup(dir, "minio_", ".tar.gz")
	if !minioFound {
		t.Fatal("expected to find MinIO backup")
	}
	minioStatus := backupAgeStatus(minioModTime, minioFound)
	if minioStatus != "green" {
		t.Errorf("expected green, got %s", minioStatus)
	}
}

func TestBackupStatus_YellowAge(t *testing.T) {
	// Create a file with modification time 48 hours ago
	dir := t.TempDir()
	pgPath := filepath.Join(dir, "postgresql_old.sql.gz")
	os.WriteFile(pgPath, []byte("old backup"), 0644)

	// Set modification time to 48h ago
	oldTime := time.Now().Add(-48 * time.Hour)
	os.Chtimes(pgPath, oldTime, oldTime)

	_, pgModTime, _, pgFound := findLatestBackup(dir, "postgresql_", ".sql.gz")
	status := backupAgeStatus(pgModTime, pgFound)
	if status != "yellow" {
		t.Errorf("expected yellow for 48h old backup, got %s", status)
	}
}

func TestBackupStatus_RedAge(t *testing.T) {
	dir := t.TempDir()
	pgPath := filepath.Join(dir, "postgresql_ancient.sql.gz")
	os.WriteFile(pgPath, []byte("ancient backup"), 0644)

	// Set modification time to 96 hours ago
	ancientTime := time.Now().Add(-96 * time.Hour)
	os.Chtimes(pgPath, ancientTime, ancientTime)

	_, pgModTime, _, pgFound := findLatestBackup(dir, "postgresql_", ".sql.gz")
	status := backupAgeStatus(pgModTime, pgFound)
	if status != "red" {
		t.Errorf("expected red for 96h old backup, got %s", status)
	}
}

func TestBackupStatus_MissingBackup(t *testing.T) {
	dir := t.TempDir()

	_, _, _, pgFound := findLatestBackup(dir, "postgresql_", ".sql.gz")
	if pgFound {
		t.Error("expected no PostgreSQL backup in empty dir")
	}

	status := backupAgeStatus(time.Time{}, false)
	if status != "missing" {
		t.Errorf("expected missing, got %s", status)
	}
}

func TestBackupStatus_NoRawPathExposed(t *testing.T) {
	dir := "/opt/clarityit/test/deep/nested/path"

	pgBasename, _, _, pgFound := findLatestBackup(dir, "postgresql_", ".sql.gz")
	if pgFound {
		t.Error("expected no backup in non-existent dir")
	}

	// Even if found, basename should not contain the directory path
	if containsRawPath(pgBasename) {
		t.Errorf("raw path leaked in basename: %s", pgBasename)
	}
}

func TestBackupStatus_RestoreDrillReport(t *testing.T) {
	dir := t.TempDir()

	// Create a restore drill report marker
	drillPath := filepath.Join(dir, "restore-drill-report-20260614.md")
	os.WriteFile(drillPath, []byte("# Restore Drill Report\n\nStatus: PASSED"), 0644)

	verifiedAt, status, found := findRestoreDrillReport(dir)
	if !found {
		t.Fatal("expected to find restore drill report")
	}
	if status != "passed" {
		t.Errorf("expected status 'passed', got '%s'", status)
	}
	if verifiedAt.IsZero() {
		t.Error("expected non-zero verified time")
	}
}

func TestBackupStatus_RestoreDrillMissing(t *testing.T) {
	dir := t.TempDir()

	_, status, found := findRestoreDrillReport(dir)
	if found {
		t.Error("expected no restore drill report")
	}
	if status != "unknown" {
		t.Errorf("expected status 'unknown', got '%s'", status)
	}
}

func TestBackupStatus_ResponseShape(t *testing.T) {
	dir := t.TempDir()

	// Create minimal backups
	os.WriteFile(filepath.Join(dir, "postgresql_20260614.sql.gz"), []byte("pg"), 0644)
	os.WriteFile(filepath.Join(dir, "minio_20260614.tar.gz"), []byte("minio"), 0644)

	pgBasename, pgModTime, pgSize, pgFound := findLatestBackup(dir, "postgresql_", ".sql.gz")
	minioBasename, minioModTime, minioSize, minioFound := findLatestBackup(dir, "minio_", ".tar.gz")

	response := map[string]any{
		"postgres": map[string]any{
			"last_backup_at": pgModTime,
			"size_bytes":     pgSize,
			"path":           pgBasename,
			"age_status":     backupAgeStatus(pgModTime, pgFound),
		},
		"minio": map[string]any{
			"last_backup_at": minioModTime,
			"size_bytes":     minioSize,
			"path":           minioBasename,
			"age_status":     backupAgeStatus(minioModTime, minioFound),
		},
	}

	// Verify response can be marshaled
	respJSON, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("failed to marshal response: %v", err)
	}

	// Verify no raw paths in response
	respStr := string(respJSON)
	if containsRawPath(respStr) {
		t.Errorf("raw path in response JSON: %s", respStr)
	}

	// Verify expected keys exist
	var resp map[string]any
	json.Unmarshal(respJSON, &resp)
	if resp["postgres"] == nil {
		t.Error("missing 'postgres' key")
	}
	if resp["minio"] == nil {
		t.Error("missing 'minio' key")
	}
}
