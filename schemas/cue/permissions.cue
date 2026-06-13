package clarityit

#RoleName: "owner" | "admin" | "manager" | "member" | "viewer" | "on_call_engineer" | "infrastructure_engineer" | "security_admin" | "auditor" | "automation_operator"

#Permission: {
  resource: string & !=""
  action: string & !=""
  risk_level: "low" | "medium" | "high" | "critical"
  requires_mfa?: bool
}

permissions: [...#Permission]

roles: [...{
  name: #RoleName
  permissions: [...string]
}]
