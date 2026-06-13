package clarityit

#ActorType: "user" | "agent" | "system"
#AutonomyLevel: "A0" | "A1" | "A2" | "A3" | "A4" | "A5"

#EventEnvelope: {
  event_id: string
  event_type: =~"^[a-z]+\\.[a-z_]+\\.[a-z_]+$"
  event_version: int & >=1
  occurred_at: string

  actor: {
    actor_id?: string
    actor_type: #ActorType
    delegated_by_user_id?: string | null
    agent_run_id?: string | null
  }

  tenant: {
    team_id: string
    workspace_id?: string | null
  }

  aggregate: {
    aggregate_type: string
    aggregate_id: string
  }

  trace: {
    request_id: string
    correlation_id: string
    causation_id?: string | null
    idempotency_key?: string | null
  }

  security: {
    permission_checked?: string | null
    autonomy_level?: #AutonomyLevel | null
    approval_id?: string | null
    mfa_verified?: bool
  }

  payload: {...}

  schema: {
    schema_name: string
    schema_version: int & >=1
  }
}

#NatsSubject: =~"^clarity\\.v[0-9]+\\.[a-z]+\\.[a-z_]+\\.[a-z_]+$"
