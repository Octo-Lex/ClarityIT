package clarityit

#AutonomyLevel: "A0" | "A1" | "A2" | "A3" | "A4" | "A5"

#AutonomyPolicy: {
  capability: string & !=""
  default_level: #AutonomyLevel
  max_level: #AutonomyLevel
  requires_approval: bool
  requires_mfa: bool
  allowed_roles?: [...string]
}

policies: [...#AutonomyPolicy]
