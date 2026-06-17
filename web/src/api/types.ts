/**
 * Shared API domain types. snake_case to match the backend JSON contract.
 * Timestamps are ISO-8601 strings (Go time.Time serializes as RFC3339).
 *
 * These are authored from the Go handler response shapes (see the route study)
 * rather than generated, because many handlers return ad-hoc map[string]any.
 */

// ─── Auth & IAM ───
export interface TeamInfo {
  id: string;
  name: string;
  slug: string;
  icon?: string;
  role: string;
}

export interface User {
  id: string;
  email: string;
  name: string;
  active: boolean;
  teams: TeamInfo[];
}

export interface Permissions {
  role: string | null;
  team_id: string | null;
  permissions: string[];
}

export interface AuthTokens {
  access_token: string;
  refresh_token?: string;
}

export interface BootstrapResponse extends AuthTokens {
  user: { id: string; email: string; name: string };
  team: { id: string; name: string; role: string };
}

export interface SwitchTeamResponse extends AuthTokens {
  team_id?: string;
  role?: string;
}

export interface Session {
  id: string;
  created_at: string;
  ip_address?: string;
  user_agent?: string;
  current?: boolean;
}

export interface MessageResponse {
  message: string;
}

export interface ForgotPasswordResponse {
  message: string;
  dev_preview?: string;
  _dev_notice?: string;
}

// ─── Universal object spine ───
export interface ObjectDetail {
  id: string;
  team_id: string;
  object_type: string;
  title: string;
  summary: string;
  status: string;
  priority: string;
  owner_id: string | null;
  created_by: string | null;
  version: number;
  metadata: Record<string, unknown>;
  created_at: string;
  updated_at: string;
}

export interface CreateObjectResponse {
  id: string;
}

export interface Link {
  id: string;
  from_object_id: string;
  to_object_id: string;
  relation_type: string;
  created_at: string;
}

export interface Comment {
  id: string;
  author_id: string;
  body: string;
  created_at: string;
  updated_at: string | null;
}

// ─── Work items ───
export interface WorkItem {
  id: string;
  title: string;
  summary: string;
  status: string;
  priority: string;
  owner_id: string | null;
  version: number;
  work_item_type: string;
  assignee_id: string | null;
  project_id: string | null;
  created_at: string;
}

// ─── Incidents ───
export interface Incident {
  id: string;
  title: string;
  summary: string;
  status: string;
  severity: string;
  impact: string;
  resolved_at: string | null;
  created_at: string;
  version: number;
}

export interface IncidentPattern {
  pattern_id?: string;
  pattern_type?: string;
  pattern_description?: string;
  signature?: string;
  title?: string;
  count?: number;
  occurrence_count?: number;
  severity_mix?: Record<string, number> | string[];
  confidence?: number;
  incident_ids?: string[];
  asset_ids?: string[];
  affected_assets?: { asset_id: string; name?: string; provider?: string }[];
  advisory_only?: boolean;
  first_seen?: string;
  last_seen?: string;
  [k: string]: unknown;
}

export interface IncidentPatternsResponse {
  patterns: IncidentPattern[];
}

// ─── Projects ───
export interface Project {
  id: string;
  title: string;
  summary: string;
  status: string;
  version: number;
  created_at: string;
}

// ─── Agents ───
export type AutonomyLevel = 'A0' | 'A1' | 'A2' | 'A3' | 'A4' | 'A5';

export interface Agent {
  id: string;
  name: string;
  description: string;
  status: string;
  max_autonomy: string;
  created_at: string;
  updated_at?: string;
}

export interface AgentGrant {
  id: string;
  tool_name: string;
  max_autonomy_level: string;
  requires_approval: boolean;
  requires_mfa: boolean;
  expires_at: string | null;
  created_at: string;
  revoked_at: string | null;
}

export interface AgentRun {
  id: string;
  agent_id: string;
  status: string;
  triggered_by: string;
  created_at: string;
  started_at?: string;
  completed_at?: string | null;
  error_message?: string | null;
}

export interface AgentIntention {
  id: string;
  intention_type: string;
  requested_tool: string;
  confidence: number;
  risk_level: string;
  autonomy_level: string;
  status: string;
  blocked_reason: string | null;
  created_at: string;
}

// ─── Approvals ───
export interface Approval {
  id: string;
  action_type: string;
  risk_level: string;
  description?: string;
  status: string;
  action_target?: Record<string, unknown> | null;
  team_id?: string;
  created_at: string;
  expires_at?: string | null;
  executed_at?: string | null;
  approved_by?: string | null;
  [k: string]: unknown;
}

export interface CreateApprovalRequest {
  action_type: string;
  risk_level: string;
  description?: string;
  action_target?: Record<string, unknown>;
}

// ─── Team ───
export interface TeamSettings {
  [key: string]: unknown;
}

export interface Member {
  user_id: string;
  name: string;
  email: string;
  role: string;
  joined_at: string;
}

export interface Invitation {
  id: string;
  email: string;
  role: string;
  invited_by: string;
  expires_at: string;
  accepted_at: string | null;
  created_at: string;
  status: string;
}

export interface AccessGrant {
  id: string;
  [key: string]: unknown;
}

// ─── Integrations ───
export interface IntegrationKey {
  id: string;
  name: string;
  allowed_sources: string[];
  allowed_scopes: string[];
  allow_unsigned_dev?: boolean;
  prefix?: string;
  key?: string;
  signing_secret?: string;
  created_at?: string;
  rotated_at?: string | null;
  [k: string]: unknown;
}

/** Returned when a key is created or rotated — includes the plaintext secret. */
export interface IntegrationKeySecret extends IntegrationKey {
  id: string;
  key: string;
  signing_secret: string;
  prefix: string;
  name: string;
}

export interface RotatedKey {
  id: string;
  key: string;
  signing_secret: string;
  prefix: string;
  name: string;
}

export interface ProxmoxStatus {
  [key: string]: unknown;
}

// ─── Assets & actions ───
export interface Asset {
  id: string;
  name?: string;
  hostname?: string;
  provider?: string;
  node?: string;
  vmid?: number;
  vm_type?: string;
  status?: string;
  team_id?: string;
  [key: string]: unknown;
}

export interface AssetAction {
  id: string;
  asset_id: string;
  action: string;
  status: string;
  [k: string]: unknown;
}

export interface DryRunPreview {
  dry_run: boolean;
  action_type: string;
  target: {
    asset_id: string;
    name: string;
    provider: string;
    node: string;
    vmid: number;
    vm_type: string;
    [k: string]: unknown;
  };
  risk_level: string;
  requires_approval: boolean;
  requires_mfa: boolean;
  min_approvers: number;
  mutation_window_required: boolean;
  mutation_window_active: boolean;
  feature_flag_enabled: boolean;
  would_create_approval: boolean;
  would_create_asset_action: boolean;
  would_call_proxmox: boolean;
  validation: {
    asset_valid: boolean;
    target_valid: boolean;
    snapshot_name_valid: boolean;
    policy_valid: boolean;
    [k: string]: unknown;
  };
  risk_score?: number;
  [k: string]: unknown;
}

export interface RiskScoreResponse {
  [key: string]: unknown;
}

// ─── Remediations ───
export interface Remediation {
  id: string;
  status: string;
  [k: string]: unknown;
}

export interface RecommendationEvidence {
  recommendation_id?: string;
  available?: boolean;
  recommendation_summary?: string;
  supporting_evidence?: { type?: string; description?: string; source?: string; [k: string]: unknown }[];
  conflicting_evidence?: { type?: string; description?: string; source?: string; [k: string]: unknown }[];
  missing_info?: { type?: string; description?: string; source?: string; [k: string]: unknown }[];
  confidence_score?: number;
  confidence_level?: string;
  risk_notes?: string;
  is_stale?: boolean;
  message?: string;
  [k: string]: unknown;
}

export interface ContextQuality {
  quality_score?: number;
  advisory_only?: boolean;
  summary?: {
    total_nodes: number;
    total_relations: number;
    stale_nodes: number;
    low_confidence_relations: number;
    conflicting_relations: number;
    confirmed_relations: number;
    dismissed_relations: number;
    [k: string]: unknown;
  };
  stale_nodes?: { node_id: string; node_type: string; label: string; days_stale: number; reason: string }[];
  low_confidence_relations?: { relation_id: string; relation_type: string; confidence: number; reason: string }[];
  conflicting_relations?: { relation_id: string; relation_type: string; conflict_reason: string }[];
  [k: string]: unknown;
}

export interface ActionOutcome {
  [key: string]: unknown;
}

// ─── Artifacts & documents ───
export interface Artifact {
  id: string;
  artifact_type: string;
  title: string;
  description?: string;
  status: string;
  content_markdown?: string;
  source_data?: Record<string, unknown>;
  team_id?: string;
  created_at?: string;
  updated_at?: string;
  files?: unknown[];
  [k: string]: unknown;
}

export interface DocumentArtifact extends Artifact {
  document_type?: string;
  document_json?: { schema_version?: number; title?: string; document_type?: string; blocks?: unknown[]; [k: string]: unknown };
}

export interface MeetingSummary extends Artifact {
  [k: string]: unknown;
}

export interface ArtifactTemplate {
  id: string;
  template_type: string;
  name: string;
  description?: string;
  content_markdown?: string;
  template_format?: string;
  document_json?: unknown;
  schema_version?: number;
  metadata?: Record<string, unknown>;
  [k: string]: unknown;
}

export interface StorageSummary {
  [key: string]: unknown;
}

export interface DocumentVersion {
  id: string;
  version_number: number;
  word_count: number;
  source: string;
  change_summary?: string;
  created_by?: string;
  created_at: string;
  document_json?: unknown;
  [k: string]: unknown;
}

export interface RestoreVersionResponse {
  artifact_id: string;
  restored_from_version: number;
  new_version_number: number;
  document_json: unknown;
  word_count: number;
}

export interface DocumentVersionsResponse {
  versions: DocumentVersion[];
}

export interface PresentonStatus {
  [key: string]: unknown;
}

// ─── Knowledge (v1.5) ───
export interface KnowledgeSearchResult {
  source_type: string;
  source_id: string;
  title: string;
  summary: string;
  snippet: string;
  rank: number;
  updated_at: string;
}

export interface KnowledgeSearchResponse {
  results: KnowledgeSearchResult[];
  total: number;
  query: string;
}

export interface KnowledgeItem {
  id: string;
  team_id: string;
  source_type: string;
  source_id: string;
  title: string;
  summary: string;
  metadata: Record<string, unknown>;
  visibility: string;
  indexed_at: string;
  updated_at: string;
}

export interface RelatedKnowledgeItem {
  source_type: string;
  source_id: string;
  title: string;
  summary: string;
  snippet: string;
  rank: number;
  reason: string;
  updated_at: string;
}

export interface RelatedKnowledgeResponse {
  source: { source_type: string; source_id: string };
  related: RelatedKnowledgeItem[];
}

export interface AskClaritySource {
  source_type: string;
  source_id: string;
  knowledge_item_id: string;
  chunk_id: string;
  title: string;
  snippet: string;
}

export interface AskClarityResponse {
  answer: string;
  sources: AskClaritySource[];
  confidence: 'low' | 'medium' | 'high';
  missing_info: string[];
}

export interface KnowledgeCollection {
  id: string;
  team_id: string;
  name: string;
  description: string;
  created_by: string | null;
  created_at: string;
  updated_at: string;
  archived_at: string | null;
  item_count: number;
}

export interface CollectionItem {
  id: string;
  collection_id: string;
  team_id: string;
  source_type: string;
  source_id: string;
  knowledge_item_id?: string;
  title?: string;
  summary?: string;
  note?: string | null;
  added_by: string | null;
  added_at: string;
}

export interface KnowledgeCollectionDetail extends KnowledgeCollection {
  items: CollectionItem[];
}

export interface CollectionsListResponse {
  collections: KnowledgeCollection[];
}

export interface AddCollectionItemResponse {
  item: CollectionItem;
  duplicate: boolean;
}

export interface SavedKnowledgeAnswer {
  id: string;
  team_id: string;
  collection_id?: string | null;
  question: string;
  answer: string;
  confidence: 'low' | 'medium' | 'high';
  sources: AskClaritySource[];
  created_by: string | null;
  created_at: string;
}

export interface SavedAnswersListResponse {
  answers: SavedKnowledgeAnswer[];
}

export interface QualityItem {
  knowledge_item_id: string;
  source_type: string;
  source_id: string;
  title: string;
  summary: string;
  indexed_at: string;
  stale_after?: string;
  days_stale?: number;
}

export interface DupGroup {
  content_hash: string;
  count: number;
  items: QualityItem[];
}

export interface KnowledgeQualityReport {
  team_id: string;
  total_items: number;
  stale_count: number;
  duplicate_count: number;
  orphan_count: number;
  by_type: Record<string, number>;
  stale_items: QualityItem[];
  duplicate_groups: DupGroup[];
  orphan_items: QualityItem[];
  generated_at: string;
}

export interface StaleItemsResponse {
  stale_items: QualityItem[];
  count: number;
}

export interface DuplicateItemsResponse {
  duplicate_groups: DupGroup[];
  count: number;
}

export interface OrphanItemsResponse {
  orphan_items: QualityItem[];
  count: number;
}

// ─── MFA ───
export interface MFaEnrollResponse {
  factor_id: string;
  secret: string;
  otpauth_uri: string;
}

export interface MFAChallengeResponse {
  challenge_token: string;
}

export interface MFARegenerateResponse {
  recovery_codes: string[];
}

export interface MFAFactor {
  id: string;
  type: string;
  verified: boolean;
  created_at: string;
}

/** WebAuthn credentials have a different shape than TOTP factors. */
export interface WebAuthnCredential {
  id: string;
  label: string;
  status: string;
  created_at: string;
  last_used_at?: string;
  [k: string]: unknown;
}

export interface MFAStatusResponse {
  enabled: boolean;
  verified_factors: number;
  pending_factors: number;
}

// ─── Admin ───
export interface AdminUser {
  id: string;
  email: string;
  name: string;
  active: boolean;
  [k: string]: unknown;
}

export interface AdminTeam {
  id: string;
  name: string;
  slug: string;
  [k: string]: unknown;
}

export interface AuditEvent {
  id: string;
  actor_id: string;
  action: string;
  entity_type: string;
  entity_id: string;
  new_value: Record<string, unknown>;
  created_at: string;
}

export interface OpsSummary {
  outbox_pending: number;
  dead_letters: number;
  agent_runs_pending: number;
  agent_runs_running: number;
  webhook_rejections_24h: number;
  agent_blocks_24h: number;
  security_events_24h: number;
  integration_keys_active: number;
  integration_keys_rotation_required: number;
  total_users: number;
  total_teams: number;
  [k: string]: number;
}

export interface WorkerInfo {
  name: string;
  status: string;
  last_seen?: string | null;
}

export interface SetupStatus {
  [key: string]: unknown;
}

export interface AdminSettings {
  [key: string]: unknown;
}

export interface DeepHealth {
  postgres?: { status: string; latency?: string; [k: string]: unknown };
  nats?: { status: string; latency?: string; [k: string]: unknown };
  redis?: { status: string; latency?: string; [k: string]: unknown };
  minio?: { status: string; latency?: string; [k: string]: unknown };
  outbox?: { pending: number; dead_letter: number; [k: string]: unknown };
  uptime?: string;
  [k: string]: unknown;
}

export interface BackupStatus {
  [key: string]: unknown;
}

export interface AdminMetrics {
  approvals?: {
    pending: number; approved: number; rejected: number; expired: number;
    executed: number; failed: number; avg_time_to_decision_seconds: number;
  };
  remediations?: {
    draft: number; proposed: number; approved: number; executing: number;
    completed: number; failed: number; cancelled: number;
  };
  asset_actions?: {
    by_status: Record<string, number>;
    by_type: Record<string, number>;
    success_rate_percent: number;
  };
  agents?: {
    runs_pending: number; runs_running: number; runs_completed: number;
    runs_failed: number; avg_reasoning_time_seconds: number;
  };
  [k: string]: unknown;
}

export interface EvaluationResults {
  run_id?: string | null;
  run_status?: string;
  scenario_count?: number;
  passed_count?: number;
  failed_count?: number;
  average_score?: number;
  safety_score?: number;
  explainability_score?: number;
  correctness_score?: number;
  quality_score?: number;
  evaluation_only?: boolean;
  created_at?: string;
  completed_at?: string;
  scenarios?: unknown[];
  [k: string]: unknown;
}

export interface MutationWindow {
  [key: string]: unknown;
}

// Approval policy simulation (advisory-only)
export interface PolicySimulationResponse {
  simulation_only: boolean;
  live_policy_changed: boolean;
  results: { scenario_id?: string; action_type?: string; risk_level?: string; [k: string]: unknown }[];
  policy_diff: { changed: boolean; changes: { level?: string; field?: string; old?: unknown; new?: unknown }[] };
  [k: string]: unknown;
}

// ─── Bootstrap ───
export interface BootstrapStatus {
  bootstrapped: boolean;
}
