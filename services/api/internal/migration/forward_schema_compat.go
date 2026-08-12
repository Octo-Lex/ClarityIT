package migration

import (
	"errors"
	"fmt"
	"regexp"
)

// SchemaCompatibilityRange is the machine-readable schema compatibility
// contract for authoritative WP-01 kernel writes. G1 introduces the schema and
// migration package, but no domain/authority/effect service writer is activated;
// those future writers must pass this contract before issuing authoritative SQL.
type SchemaCompatibilityRange struct {
	MinimumAuthoritativeWriteVersion string `json:"minimum_authoritative_write_version"`
	MaximumAuthoritativeWriteVersion string `json:"maximum_authoritative_write_version"`
}

// WP01SchemaCompatibility is intentionally exact for the G1 binary: the
// authoritative WP-01 write surface is compatible only with the complete
// atomic G1 target. Exact 0001 remains a supported diagnostic/migration
// foundation, not an authoritative WP-01 write target.
var WP01SchemaCompatibility = SchemaCompatibilityRange{
	MinimumAuthoritativeWriteVersion: ForwardTargetVersion,
	MaximumAuthoritativeWriteVersion: ForwardTargetVersion,
}

var (
	ErrSchemaVersionMalformed = errors.New("schema version is malformed")
	ErrSchemaVersionTooOld    = errors.New("schema version is older than the WP-01 authoritative-write range")
	ErrSchemaVersionTooNew    = errors.New("schema version is newer than the WP-01 authoritative-write range")
)

var fixedSchemaVersionRE = regexp.MustCompile(`^[0-9]{4}$`)

// CheckWP01AuthoritativeWriteVersion is the fail-closed compatibility boundary
// for any WP-01 service that performs authoritative kernel writes. G1 itself
// activates no such service writer; schema presence and clarityit_app grants do
// not constitute cutover authority. Future writer code must call this boundary
// (or an equivalent server-side check bound to this exact range) before writes.
func CheckWP01AuthoritativeWriteVersion(version string) error {
	if !fixedSchemaVersionRE.MatchString(version) {
		return fmt.Errorf("%w: %q", ErrSchemaVersionMalformed, version)
	}
	if version < WP01SchemaCompatibility.MinimumAuthoritativeWriteVersion {
		return fmt.Errorf("%w: current=%s minimum=%s", ErrSchemaVersionTooOld, version, WP01SchemaCompatibility.MinimumAuthoritativeWriteVersion)
	}
	if version > WP01SchemaCompatibility.MaximumAuthoritativeWriteVersion {
		return fmt.Errorf("%w: current=%s maximum=%s", ErrSchemaVersionTooNew, version, WP01SchemaCompatibility.MaximumAuthoritativeWriteVersion)
	}
	return nil
}
