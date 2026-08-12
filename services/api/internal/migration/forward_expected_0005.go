package migration

func init() {
	expectedKernelTables = append(expectedKernelTables, "evidence_manifest_refs")
	expectedKernelFunctions = append(expectedKernelFunctions, "require_evidence_manifest_typed_lineage")
}
