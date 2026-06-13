package clarityit

#ViewName: "command_center" | "queue" | "project" | "wiki" | "hub" | "grid" | "agent_console" | "approval_inbox" | "audit_timeline" | "context_graph" | "kanban" | "table" | "timeline" | "detail"

#ViewCompatibility: {
  object_type: string
  views: [...#ViewName]
  required_permission: string
  masked_fields?: [...string]
}

view_registry: [...#ViewCompatibility]
