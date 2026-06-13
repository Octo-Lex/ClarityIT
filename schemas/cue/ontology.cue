package clarityit

#ObjectType: "work_item" | "ticket" | "incident" | "change" | "problem" | "alert" | "doc" | "runbook" | "message" | "project" | "asset" | "service" | "approval" | "agent_run"

#Priority: "critical" | "high" | "medium" | "low" | "none"
#Status: string & !=""

#Object: {
  id?: string
  team_id: string
  object_type: #ObjectType
  title: string & !=""
  summary?: string
  status: #Status
  priority?: #Priority
  owner_user_id?: string
  created_by: string
  version: int & >=1
}

#WorkItemType: "task" | "ticket" | "incident" | "change" | "problem" | "project_task" | "alert_work_item"

#WorkItem: #Object & {
  object_type: "work_item" | "ticket" | "incident" | "change" | "problem"
  work_item_type: #WorkItemType
  due_at?: string
  sla_policy_id?: string
  queue_id?: string
  project_id?: string
}
