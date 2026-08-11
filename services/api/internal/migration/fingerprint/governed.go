package fingerprint

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/clarityit/api/internal/migration/canonicalize"
)

// GovernedAlgorithm and GovernedDomain are transcribed verbatim from
// governed_fingerprint.py. The domain includes a trailing NUL byte.
const (
	GovernedAlgorithm = "clarityit-g3-governed-v1"
	GovernedDomain    = "clarityit-g3-governed-v1\x00"
)

// SignedG2Manifest is the subset of the G2 product manifest the projection
// needs to derive the governed inventory (relations, sequence, app functions).
// The full manifest is large; the runner parses it from the embedded asset.
type SignedG2Manifest struct {
	Tables    map[string]json.RawMessage `json:"tables"`
	Sequences []struct {
		Schema string `json:"schema"`
		Name   string `json:"name"`
	} `json:"sequences"`
	TargetGrants struct {
		ApplicationFunctions []struct {
			Schema string `json:"schema"`
			Name   string `json:"name"`
			Args   string `json:"args"`
		} `json:"application_functions"`
	} `json:"target_grants"`
}

// ControlManifestFunctions is the subset of the control manifest the projection
// needs: the platform function inventory (names like "platform.fn(args)").
type ControlManifestFunctions struct {
	Functions []string                   `json:"functions"`
	Tables    map[string]json.RawMessage `json:"tables"`
}

// GovernedCapture builds the governed projection from the live catalog. It is
// the Go port of governed_fingerprint.governed_capture. All catalog
// incompleteness is an error (never a partial profile).
//
// signed and control are parsed from the embedded G2 and control manifests.
func GovernedCapture(ctx context.Context, q pgxQuerier, signed *SignedG2Manifest, control *ControlManifestFunctions) (map[string]any, error) {
	// --- Derive the governed inventory from the signed manifests ---
	// Governed relations: signed product tables + sequences + platform tables.
	var governedRelations [][2]string
	for key := range signed.Tables {
		schema, name := splitTableKey(key)
		governedRelations = append(governedRelations, [2]string{schema, name})
	}
	for _, seq := range signed.Sequences {
		governedRelations = append(governedRelations, [2]string{seq.Schema, seq.Name})
	}
	for tname := range control.Tables {
		governedRelations = append(governedRelations, [2]string{"platform", tname})
	}

	// App signatures: signed application functions + platform functions.
	appSignatures := map[funcKey]struct{}{}
	for _, af := range signed.TargetGrants.ApplicationFunctions {
		appSignatures[funcKey{af.Schema, af.Name, af.Args}] = struct{}{}
	}
	for _, fn := range control.Functions {
		schema, name, args := splitPlatformFunction(fn)
		appSignatures[funcKey{schema, name, args}] = struct{}{}
	}

	// --- Catalog extraction (each query treats incompleteness as an error) ---
	roles, err := queryRoles(ctx, q)
	if err != nil {
		return nil, err
	}
	memberships, err := queryMemberships(ctx, q)
	if err != nil {
		return nil, err
	}
	relations, err := queryRelations(ctx, q)
	if err != nil {
		return nil, err
	}
	relationsProjected := make([]map[string]any, 0, len(relations))
	for _, r := range relations {
		// Project to the 4-field signed contract (drop the type field).
		relationsProjected = append(relationsProjected, map[string]any{
			"schema": r.Schema, "name": r.Name, "kind": r.Kind, "persistence": r.Persistence,
		})
	}
	columns, err := queryColumns(ctx, q)
	if err != nil {
		return nil, err
	}
	// The signed G2 product contract is column-order independent.
	// G3 fresh installs already emit these tables in column-name order;
	// normalize inherited P1/P2 attnum order to that same canonical order.
	// Only sort signed product tables — leave platform tables untouched.
	for tableKey := range signed.Tables {
		if colList, ok := columns[tableKey]; ok {
			sort.Slice(colList, func(i, j int) bool {
				return colList[i].Name < colList[j].Name
			})
		}
	}
	constraints, err := queryConstraints(ctx, q)
	if err != nil {
		return nil, err
	}
	indexes, err := queryIndexes(ctx, q)
	if err != nil {
		return nil, err
	}
	triggers, err := queryTriggers(ctx, q)
	if err != nil {
		return nil, err
	}
	sequences, err := querySequences(ctx, q)
	if err != nil {
		return nil, err
	}
	appFunctions, err := queryAppFunctions(ctx, q, appSignatures)
	if err != nil {
		return nil, err
	}
	grants, err := queryGrantMaterial(ctx, q, governedRelations, appSignatures)
	if err != nil {
		return nil, err
	}
	ownership, err := queryProjectedOwnership(ctx, q, appSignatures)
	if err != nil {
		return nil, err
	}
	defaultPrivs, err := queryDefaultPrivilegesEffective(ctx, q)
	if err != nil {
		return nil, err
	}
	extInvariant, err := queryExtensionOwnerInvariant(ctx, q)
	if err != nil {
		return nil, err
	}

	// --- Assemble the projection (keys MUST match governed_fingerprint.py) ---
	rolesAny := make([]any, len(roles))
	for i, r := range roles {
		// Python shape: {"name":..., "flags":{"superuser":...,"inherit":...,...}}
		rolesAny[i] = map[string]any{
			"name": r.Name,
			"flags": map[string]any{
				"superuser":   r.RolSuper,
				"inherit":     r.RolInherit,
				"createrole":  r.RolCreateRole,
				"createdb":    r.RolCreateDB,
				"canlogin":    r.RolCanLogin,
				"replication": r.RolReplication,
				"bypassrls":   r.RolBypassRLS,
			},
		}
	}
	membershipsAny := make([]any, len(memberships))
	for i, m := range memberships {
		membershipsAny[i] = map[string]any{
			"member": m.Member, "role_of": m.RoleOf,
			"admin_option": m.AdminOption, "inherit_option": m.InheritOption, "set_option": m.SetOption,
		}
	}
	appFunctionsAny := make([]any, len(appFunctions))
	for i, f := range appFunctions {
		appFunctionsAny[i] = map[string]any{
			"schema": f.Schema, "name": f.Name, "args": f.Args, "body": f.Body,
		}
	}
	sequencesAny := make([]any, len(sequences))
	for i, s := range sequences {
		sequencesAny[i] = map[string]any{
			"schema": s.Schema, "name": s.Name, "type": s.Type, "start": s.Start,
			"increment": s.Increment, "max": s.Max, "min": s.Min, "cache": s.Cache, "cycle": s.Cycle,
		}
	}
	grantsAny := make([]any, len(grants))
	for i, g := range grants {
		grantsAny[i] = g
	}
	defaultPrivsAny := make([]any, len(defaultPrivs))
	for i, d := range defaultPrivs {
		defaultPrivsAny[i] = d
	}

	return map[string]any{
		"algorithm":             GovernedAlgorithm,
		"schemas":               sortedStrings(GovernedSchemas),
		"relations":             relationsProjected,
		"columns":               columns,
		"constraints":           constraints,
		"indexes":               indexes,
		"triggers":              triggers,
		"sequences":             sequencesAny,
		"application_functions": appFunctionsAny,
		"roles":                 rolesAny,
		"memberships":           membershipsAny,
		"roles_digest":          rolesDigest(roles, memberships),
		"grants":                grantsAny,
		"default_privileges":    defaultPrivsAny,
		"ownership":             ownership,
		"extension_owners":      extInvariant,
	}, nil
}

// GovernedFingerprint computes SHA-256(domain || canonical(capture)).hex.
func GovernedFingerprint(capture map[string]any) (string, error) {
	payload, err := canonicalize.Marshal(capture)
	if err != nil {
		return "", fmt.Errorf("governed canonicalize: %w", err)
	}
	h := sha256.New()
	h.Write([]byte(GovernedDomain))
	h.Write(payload)
	return hex.EncodeToString(h.Sum(nil)), nil
}

// rolesDigest reproduces governed_fingerprint.roles_digest: SHA-256 over the
// canonical serialization of [canonical(roles), canonical(memberships)]. The
// role rows MUST use the same nested {"name","flags":{...}} shape as the
// projection's roles field — roles_digest is called on the same row list.
func rolesDigest(roles []roleRow, memberships []membershipRow) string {
	rolesAny := make([]any, len(roles))
	for i, r := range roles {
		rolesAny[i] = map[string]any{
			"name": r.Name,
			"flags": map[string]any{
				"superuser":   r.RolSuper,
				"inherit":     r.RolInherit,
				"createrole":  r.RolCreateRole,
				"createdb":    r.RolCreateDB,
				"canlogin":    r.RolCanLogin,
				"replication": r.RolReplication,
				"bypassrls":   r.RolBypassRLS,
			},
		}
	}
	membershipsAny := make([]any, len(memberships))
	for i, m := range memberships {
		membershipsAny[i] = map[string]any{
			"member": m.Member, "role_of": m.RoleOf,
			"admin_option": m.AdminOption, "inherit_option": m.InheritOption, "set_option": m.SetOption,
		}
	}
	rb, err := canonicalize.Marshal(rolesAny)
	if err != nil {
		panic("roles canonicalize: " + err.Error())
	}
	mb, err := canonicalize.Marshal(membershipsAny)
	if err != nil {
		panic("memberships canonicalize: " + err.Error())
	}
	// Python: _normalize(payload) where payload = [_normalize(roles), _normalize(memberships)]
	// i.e. a JSON array of two strings.
	payload := []any{string(rb), string(mb)}
	pb, err := canonicalize.Marshal(payload)
	if err != nil {
		panic("payload canonicalize: " + err.Error())
	}
	h := sha256.Sum256(pb)
	return hex.EncodeToString(h[:])
}

// splitTableKey splits "schema.table" from the manifest's table dict key.
func splitTableKey(key string) (string, string) {
	for i := 0; i < len(key); i++ {
		if key[i] == '.' {
			return key[:i], key[i+1:]
		}
	}
	return key, ""
}

// splitPlatformFunction splits "platform.fn(args)" into (schema, name, args).
func splitPlatformFunction(fn string) (string, string, string) {
	dot := -1
	paren := -1
	for i := 0; i < len(fn); i++ {
		if fn[i] == '.' && dot == -1 {
			dot = i
		}
		if fn[i] == '(' {
			paren = i
			break
		}
	}
	if dot == -1 || paren == -1 {
		return fn, "", ""
	}
	schema := fn[:dot]
	name := fn[dot+1 : paren]
	args := fn[paren+1:]
	if len(args) > 0 && args[len(args)-1] == ')' {
		args = args[:len(args)-1]
	}
	return schema, name, args
}

func sortedStrings(in []string) []any {
	out := make([]any, len(in))
	s := append([]string(nil), in...)
	sort.Strings(s)
	for i, v := range s {
		out[i] = v
	}
	return out
}
