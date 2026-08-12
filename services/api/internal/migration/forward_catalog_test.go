package migration

import (
	"errors"
	"testing"

	"github.com/clarityit/api/internal/migration/assets"
)

func testForwardCatalog() []ForwardRevision {
	return []ForwardRevision{
		{Version: "0002", Name: "wp01-kernel-foundation", Asset: assets.AssetForward0002, Checksum: "a"},
		{Version: "0003", Name: "wp01-kernel-integrity-hardening", Asset: assets.AssetForward0003, Checksum: "b"},
		{Version: "0004", Name: "wp01-packet-immutability-barrier", Asset: assets.AssetForward0004, Checksum: "c"},
		{Version: "0005", Name: "wp01-lineage-and-message-integrity", Asset: assets.AssetForward0005, Checksum: "d"},
	}
}

func TestValidateForwardHistoryFoundation(t *testing.T) {
	state, err := validateForwardHistory(
		[]forwardLedgerRow{{Version: "0001", Name: "baseline", Checksum: BaselineChecksum, Success: true}},
		testForwardCatalog(),
	)
	if err != nil || state != "foundation" {
		t.Fatalf("state=%q err=%v", state, err)
	}
}

func TestValidateForwardHistoryCurrent(t *testing.T) {
	rows := []forwardLedgerRow{
		{Version: "0001", Name: "baseline", Checksum: BaselineChecksum, Success: true},
		{Version: "0002", Name: "wp01-kernel-foundation", Checksum: "a", Success: true},
		{Version: "0003", Name: "wp01-kernel-integrity-hardening", Checksum: "b", Success: true},
		{Version: "0004", Name: "wp01-packet-immutability-barrier", Checksum: "c", Success: true},
		{Version: "0005", Name: "wp01-lineage-and-message-integrity", Checksum: "d", Success: true},
	}
	state, err := validateForwardHistory(rows, testForwardCatalog())
	if err != nil || state != "current" {
		t.Fatalf("state=%q err=%v", state, err)
	}
}

func TestValidateForwardHistoryRejectsIntermediatePrefix(t *testing.T) {
	rows := []forwardLedgerRow{
		{Version: "0001", Checksum: BaselineChecksum, Success: true},
		{Version: "0002", Name: "wp01-kernel-foundation", Checksum: "a", Success: true},
	}
	_, err := validateForwardHistory(rows, testForwardCatalog())
	if !errors.Is(err, ErrForwardIntermediate) {
		t.Fatalf("expected ErrForwardIntermediate, got %v", err)
	}
}

func TestValidateForwardHistoryRejectsChecksumMutation(t *testing.T) {
	rows := []forwardLedgerRow{
		{Version: "0001", Checksum: BaselineChecksum, Success: true},
		{Version: "0002", Name: "wp01-kernel-foundation", Checksum: "MUTATED", Success: true},
		{Version: "0003", Name: "wp01-kernel-integrity-hardening", Checksum: "b", Success: true},
		{Version: "0004", Name: "wp01-packet-immutability-barrier", Checksum: "c", Success: true},
		{Version: "0005", Name: "wp01-lineage-and-message-integrity", Checksum: "d", Success: true},
	}
	_, err := validateForwardHistory(rows, testForwardCatalog())
	if !errors.Is(err, ErrForwardHistory) {
		t.Fatalf("expected ErrForwardHistory, got %v", err)
	}
}

func TestValidateForwardHistoryRejectsUnknownRevision(t *testing.T) {
	rows := []forwardLedgerRow{
		{Version: "0001", Checksum: BaselineChecksum, Success: true},
		{Version: "0002", Name: "wp01-kernel-foundation", Checksum: "a", Success: true},
		{Version: "0003", Name: "wp01-kernel-integrity-hardening", Checksum: "b", Success: true},
		{Version: "0004", Name: "wp01-packet-immutability-barrier", Checksum: "c", Success: true},
		{Version: "9999", Name: "unknown", Checksum: "d", Success: true},
	}
	_, err := validateForwardHistory(rows, testForwardCatalog())
	if !errors.Is(err, ErrForwardHistory) {
		t.Fatalf("expected ErrForwardHistory, got %v", err)
	}
}

func TestValidateForwardSQLRejectsClientOrTransactionControl(t *testing.T) {
	for _, sql := range []string{
		"\\set ON_ERROR_STOP on\nSELECT 1;",
		"BEGIN;\nSELECT 1;",
		"SELECT 1;\nCOMMIT;",
	} {
		if err := validateForwardSQL([]byte(sql)); err == nil {
			t.Fatalf("expected rejection for %q", sql)
		}
	}
	if err := validateForwardSQL([]byte("SET LOCAL ROLE clarityit_owner;\nCREATE TABLE x(id int);\n")); err != nil {
		t.Fatalf("unexpected valid SQL rejection: %v", err)
	}
}
