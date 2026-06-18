package admin

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// BackupStatusHandler provides read-only backup verification visibility.
type BackupStatusHandler struct {
	pool        *pgxpool.Pool
	backupDir   string
}

func NewBackupStatusHandler(pool *pgxpool.Pool) *BackupStatusHandler {
	backupDir := os.Getenv("BACKUP_DIR")
	if backupDir == "" {
		backupDir = "/opt/clarityit/backups"
	}
	return &BackupStatusHandler{pool: pool, backupDir: backupDir}
}

// backupAgeStatus computes the age status of a backup.
// green: <=24h, yellow: >24h and <=72h, red: >72h, missing: no backup
func backupAgeStatus(modTime time.Time, exists bool) string {
	if !exists {
		return "missing"
	}
	age := time.Since(modTime)
	if age <= 24*time.Hour {
		return "green"
	}
	if age <= 72*time.Hour {
		return "yellow"
	}
	return "red"
}

// findLatestBackup finds the most recent file matching the prefix in the backup directory.
// Returns the basename, modtime, size, and whether a file was found.
// Never returns the full filesystem path — only basename for safety.
func findLatestBackup(dir, prefix, suffix string) (basename string, modTime time.Time, size int64, found bool) {
	pattern := prefix + "*" + suffix
	matches, err := filepath.Glob(filepath.Join(dir, pattern))
	if err != nil || len(matches) == 0 {
		return "", time.Time{}, 0, false
	}

	var latestPath string
	var latestTime time.Time

	for _, m := range matches {
		info, err := os.Stat(m)
		if err != nil {
			continue
		}
		if info.IsDir() {
			continue
		}
		if latestPath == "" || info.ModTime().After(latestTime) {
			latestPath = m
			latestTime = info.ModTime()
			size = info.Size()
		}
	}

	if latestPath == "" {
		return "", time.Time{}, 0, false
	}

	// Return basename only — never expose raw filesystem paths
	basename = filepath.Base(latestPath)
	modTime = latestTime
	found = true
	return
}

// findRestoreDrillReport looks for a restore drill report marker.
func findRestoreDrillReport(dir string) (verifiedAt time.Time, status string, found bool) {
	// Look for restore-drill-report.* files
	matches, err := filepath.Glob(filepath.Join(dir, "restore-drill-report*"))
	if err != nil || len(matches) == 0 {
		return time.Time{}, "unknown", false
	}

	var latestTime time.Time
	for _, m := range matches {
		info, err := os.Stat(m)
		if err != nil {
			continue
		}
		if latestTime.IsZero() || info.ModTime().After(latestTime) {
			latestTime = info.ModTime()
		}
	}

	if latestTime.IsZero() {
		return time.Time{}, "unknown", false
	}

	return latestTime, "passed", true
}

// GetBackupStatus returns backup freshness and restore drill status.
func (h *BackupStatusHandler) GetBackupStatus(w http.ResponseWriter, r *http.Request) {
	// Check if backup directory exists
	_, err := os.Stat(h.backupDir)
	backupDirExists := err == nil

	response := map[string]any{}

	// PostgreSQL backup
	var pgBasename string
	var pgModTime time.Time
	var pgSize int64
	var pgFound bool

	if backupDirExists {
		pgBasename, pgModTime, pgSize, pgFound = findLatestBackup(h.backupDir, "postgresql_", ".sql.gz")
	}

	response["postgres"] = map[string]any{
		"last_backup_at": pgModTime,
		"size_bytes":     pgSize,
		"path":           pgBasename, // basename only — no raw path
		"age_status":     backupAgeStatus(pgModTime, pgFound),
	}

	// MinIO backup
	var minioBasename string
	var minioModTime time.Time
	var minioSize int64
	var minioFound bool

	if backupDirExists {
		minioBasename, minioModTime, minioSize, minioFound = findLatestBackup(h.backupDir, "minio_", ".tar.gz")
	}

	response["minio"] = map[string]any{
		"last_backup_at": minioModTime,
		"size_bytes":     minioSize,
		"path":           minioBasename,
		"age_status":     backupAgeStatus(minioModTime, minioFound),
	}

	// Restore drill
	var drillTime time.Time
	var drillStatus string
	var drillFound bool
	if backupDirExists {
		drillTime, drillStatus, drillFound = findRestoreDrillReport(h.backupDir)
	}

	if !drillFound {
		drillStatus = "unknown"
	}

	response["restore_drill"] = map[string]any{
		"last_verified_at": drillTime,
		"status":           drillStatus,
		"source":           "restore-drill-report",
	}

	writeJSON(w, 200, response)
}

// writeJSON and writeErr are shared helpers defined in ops_handler.go
// (we reuse them — same package)

// Helper to verify no raw paths leak
func containsRawPath(s string) bool {
	return strings.Contains(s, "/opt/") || strings.Contains(s, "/var/lib/docker/")
}
