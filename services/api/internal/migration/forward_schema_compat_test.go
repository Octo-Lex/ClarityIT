package migration

import (
	"errors"
	"testing"
)

func TestWP01AuthoritativeWriteSchemaCompatibility(t *testing.T) {
	if WP01SchemaCompatibility.MinimumAuthoritativeWriteVersion != "0005" ||
		WP01SchemaCompatibility.MaximumAuthoritativeWriteVersion != "0005" {
		t.Fatalf("unexpected WP-01 write range: %+v", WP01SchemaCompatibility)
	}

	cases := []struct {
		name    string
		version string
		wantErr error
	}{
		{name: "exact_target", version: "0005"},
		{name: "foundation_too_old", version: "0001", wantErr: ErrSchemaVersionTooOld},
		{name: "future_too_new", version: "0006", wantErr: ErrSchemaVersionTooNew},
		{name: "malformed", version: "5", wantErr: ErrSchemaVersionMalformed},
		{name: "non_numeric", version: "v005", wantErr: ErrSchemaVersionMalformed},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := CheckWP01AuthoritativeWriteVersion(tc.version)
			if tc.wantErr == nil {
				if err != nil {
					t.Fatalf("version %s unexpectedly rejected: %v", tc.version, err)
				}
				return
			}
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("version %s error=%v want=%v", tc.version, err, tc.wantErr)
			}
		})
	}
}
