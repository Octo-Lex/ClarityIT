package clarityit

#ToolRisk: "low" | "medium" | "high" | "critical"
#AutonomyLevel: "A0" | "A1" | "A2" | "A3" | "A4" | "A5"

#Tool: {
  name: =~"^[a-z][a-z0-9_]*$"
  description: string & !=""
  required_permission: string
  risk_level: #ToolRisk
  max_autonomy_level: #AutonomyLevel
  requires_approval: bool
  requires_mfa: bool
  input_schema_ref: string
  output_schema_ref: string
  emits_events?: [...string]
}

tools: [...#Tool]
