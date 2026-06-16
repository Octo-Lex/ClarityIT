-- 037_document_templates.sql
-- v1.4 Track 5: Native Document Templates.
-- Extends artifact_templates to support structured document_json templates
-- alongside existing markdown templates.

-- Make content_markdown nullable (was NOT NULL) for document_json templates
ALTER TABLE artifact_templates ALTER COLUMN content_markdown DROP NOT NULL;

-- Drop old constraint requiring content_markdown for all templates
ALTER TABLE artifact_templates DROP CONSTRAINT IF EXISTS atpl_content_nonempty;

-- Add new columns
ALTER TABLE artifact_templates
  ADD COLUMN IF NOT EXISTS template_format TEXT NOT NULL DEFAULT 'markdown',
  ADD COLUMN IF NOT EXISTS document_json JSONB NULL,
  ADD COLUMN IF NOT EXISTS schema_version INTEGER NULL;

-- template_format allowlist
ALTER TABLE artifact_templates
  DROP CONSTRAINT IF EXISTS atpl_format_check;
ALTER TABLE artifact_templates
  ADD CONSTRAINT atpl_format_check CHECK (
    template_format IN ('markdown', 'document_json')
  );

-- Conditional: markdown templates need content_markdown; document_json templates need document_json
ALTER TABLE artifact_templates
  DROP CONSTRAINT IF EXISTS atpl_content_or_json;
ALTER TABLE artifact_templates
  ADD CONSTRAINT atpl_content_or_json CHECK (
    (template_format = 'markdown' AND length(btrim(content_markdown)) > 0) OR
    (template_format = 'document_json' AND document_json IS NOT NULL)
  );

-- document_json must be a JSON object
ALTER TABLE artifact_templates
  DROP CONSTRAINT IF EXISTS atpl_docjson_object;
ALTER TABLE artifact_templates
  ADD CONSTRAINT atpl_docjson_object CHECK (
    template_format != 'document_json' OR jsonb_typeof(document_json) = 'object'
  );

-- ─── Seed 7 System Document Templates ───
-- All use template_type='document', template_format='document_json', schema_version=1

INSERT INTO artifact_templates (id, team_id, template_type, name, description, content_markdown, document_json, schema_version, template_format, metadata, is_system) VALUES
('a0000000-0000-0000-0000-000000000010', NULL, 'document', 'Decision Memo (Structured)',
 'Structured decision memo with context, options, recommendation, and impact sections.',
 NULL,
 '{
   "schema_version": 1,
   "title": "Decision Memo",
   "document_type": "decision_memo",
   "blocks": [
     {"id": "blk_001", "type": "heading", "level": 1, "text": "Decision: [Title]"},
     {"id": "blk_002", "type": "callout", "variant": "info", "text": "Date: [YYYY-MM-DD] — Decided By: [name/role]"},
     {"id": "blk_003", "type": "heading", "level": 2, "text": "Context"},
     {"id": "blk_004", "type": "paragraph", "text": "What problem are we solving? What constraints exist?"},
     {"id": "blk_005", "type": "heading", "level": 2, "text": "Options Considered"},
     {"id": "blk_006", "type": "heading", "level": 3, "text": "Option A"},
     {"id": "blk_007", "type": "bullets", "items": ["Pros:", "Cons:"]},
     {"id": "blk_008", "type": "heading", "level": 3, "text": "Option B"},
     {"id": "blk_009", "type": "bullets", "items": ["Pros:", "Cons:"]},
     {"id": "blk_010", "type": "heading", "level": 2, "text": "Recommendation"},
     {"id": "blk_011", "type": "paragraph", "text": "What do we recommend and why?"},
     {"id": "blk_012", "type": "heading", "level": 2, "text": "Impact"},
     {"id": "blk_013", "type": "bullets", "items": ["Systems affected:", "Teams affected:", "Follow-up actions:"]}
   ]
 }'::jsonb,
 1, 'document_json', '{"doc_type": "decision_memo"}'::jsonb, true)
ON CONFLICT (id) DO NOTHING;

INSERT INTO artifact_templates (id, team_id, template_type, name, description, content_markdown, document_json, schema_version, template_format, metadata, is_system) VALUES
('a0000000-0000-0000-0000-000000000011', NULL, 'document', 'Implementation Plan (Structured)',
 'Structured implementation plan with overview, scope, architecture, risks, and timeline.',
 NULL,
 '{
   "schema_version": 1,
   "title": "Implementation Plan",
   "document_type": "implementation_plan",
   "blocks": [
     {"id": "blk_001", "type": "heading", "level": 1, "text": "Implementation Plan: [Title]"},
     {"id": "blk_002", "type": "callout", "variant": "info", "text": "Status: Draft — Last Updated: [YYYY-MM-DD]"},
     {"id": "blk_003", "type": "heading", "level": 2, "text": "Overview"},
     {"id": "blk_004", "type": "paragraph", "text": "Brief description of what is being implemented and why."},
     {"id": "blk_005", "type": "heading", "level": 2, "text": "Scope"},
     {"id": "blk_006", "type": "bullets", "items": ["In scope:", "Out of scope:"]},
     {"id": "blk_007", "type": "heading", "level": 2, "text": "Architecture"},
     {"id": "blk_008", "type": "paragraph", "text": "Key architectural decisions and components involved."},
     {"id": "blk_009", "type": "heading", "level": 2, "text": "Risks"},
     {"id": "blk_010", "type": "bullets", "items": ["Risk 1: [description and mitigation]", "Risk 2: [description and mitigation]"]},
     {"id": "blk_011", "type": "heading", "level": 2, "text": "Timeline"},
     {"id": "blk_012", "type": "paragraph", "text": "Milestones and target dates."},
     {"id": "blk_013", "type": "heading", "level": 2, "text": "Next Steps"},
     {"id": "blk_014", "type": "bullets", "items": ["Action 1", "Action 2"]}
   ]
 }'::jsonb,
 1, 'document_json', '{"doc_type": "implementation_plan"}'::jsonb, true)
ON CONFLICT (id) DO NOTHING;

INSERT INTO artifact_templates (id, team_id, template_type, name, description, content_markdown, document_json, schema_version, template_format, metadata, is_system) VALUES
('a0000000-0000-0000-0000-000000000012', NULL, 'document', 'Incident Summary (Structured)',
 'Structured post-incident summary with timeline, impact, root cause, and lessons learned.',
 NULL,
 '{
   "schema_version": 1,
   "title": "Incident Summary",
   "document_type": "incident_summary",
   "blocks": [
     {"id": "blk_001", "type": "heading", "level": 1, "text": "Incident: [Title]"},
     {"id": "blk_002", "type": "callout", "variant": "warning", "text": "Severity: [sev1-sev5] — Duration: [hours] — Date: [YYYY-MM-DD]"},
     {"id": "blk_003", "type": "heading", "level": 2, "text": "Timeline"},
     {"id": "blk_004", "type": "bullets", "items": ["[time] — [event]", "[time] — [event]"]},
     {"id": "blk_005", "type": "heading", "level": 2, "text": "Impact"},
     {"id": "blk_006", "type": "bullets", "items": ["Services affected:", "Users impacted:", "Data impact:"]},
     {"id": "blk_007", "type": "heading", "level": 2, "text": "Root Cause"},
     {"id": "blk_008", "type": "paragraph", "text": "What caused the incident?"},
     {"id": "blk_009", "type": "heading", "level": 2, "text": "Resolution"},
     {"id": "blk_010", "type": "paragraph", "text": "How was the incident resolved?"},
     {"id": "blk_011", "type": "heading", "level": 2, "text": "Lessons Learned"},
     {"id": "blk_012", "type": "bullets", "items": ["What went well:", "What went poorly:", "Action items for prevention:"]}
   ]
 }'::jsonb,
 1, 'document_json', '{"doc_type": "incident_summary"}'::jsonb, true)
ON CONFLICT (id) DO NOTHING;

INSERT INTO artifact_templates (id, team_id, template_type, name, description, content_markdown, document_json, schema_version, template_format, metadata, is_system) VALUES
('a0000000-0000-0000-0000-000000000013', NULL, 'document', 'Architecture Document (Structured)',
 'Structured architecture document with overview, components, data flow, and security.',
 NULL,
 '{
   "schema_version": 1,
   "title": "Architecture Document",
   "document_type": "architecture_doc",
   "blocks": [
     {"id": "blk_001", "type": "heading", "level": 1, "text": "Architecture: [System Name]"},
     {"id": "blk_002", "type": "callout", "variant": "info", "text": "Last Updated: [YYYY-MM-DD] — Status: [Draft/Review/Current]"},
     {"id": "blk_003", "type": "heading", "level": 2, "text": "Overview"},
     {"id": "blk_004", "type": "paragraph", "text": "Brief description of the system and its purpose."},
     {"id": "blk_005", "type": "heading", "level": 2, "text": "Components"},
     {"id": "blk_006", "type": "bullets", "items": ["Component: Purpose, Technology, Dependencies"]},
     {"id": "blk_007", "type": "heading", "level": 2, "text": "Data Flow"},
     {"id": "blk_008", "type": "paragraph", "text": "Describe how data moves through the system."},
     {"id": "blk_009", "type": "heading", "level": 2, "text": "Security"},
     {"id": "blk_010", "type": "bullets", "items": ["Trust boundaries", "Access controls"]},
     {"id": "blk_011", "type": "heading", "level": 2, "text": "Scalability"},
     {"id": "blk_012", "type": "paragraph", "text": "How does the system scale? What are the limits?"}
   ]
 }'::jsonb,
 1, 'document_json', '{"doc_type": "architecture_doc"}'::jsonb, true)
ON CONFLICT (id) DO NOTHING;

INSERT INTO artifact_templates (id, team_id, template_type, name, description, content_markdown, document_json, schema_version, template_format, metadata, is_system) VALUES
('a0000000-0000-0000-0000-000000000014', NULL, 'document', 'Training Document (Structured)',
 'Structured training document with objectives, prerequisites, steps, and troubleshooting.',
 NULL,
 '{
   "schema_version": 1,
   "title": "Training Document",
   "document_type": "training_doc",
   "blocks": [
     {"id": "blk_001", "type": "heading", "level": 1, "text": "Training: [Topic]"},
     {"id": "blk_002", "type": "callout", "variant": "info", "text": "Audience: [Who] — Duration: [minutes]"},
     {"id": "blk_003", "type": "heading", "level": 2, "text": "Learning Objectives"},
     {"id": "blk_004", "type": "numbered_list", "items": ["Objective 1", "Objective 2", "Objective 3"]},
     {"id": "blk_005", "type": "heading", "level": 2, "text": "Prerequisites"},
     {"id": "blk_006", "type": "bullets", "items": ["Required knowledge", "Required tools"]},
     {"id": "blk_007", "type": "heading", "level": 2, "text": "Steps"},
     {"id": "blk_008", "type": "paragraph", "text": "Step-by-step instructions go here."},
     {"id": "blk_009", "type": "heading", "level": 2, "text": "Best Practices"},
     {"id": "blk_010", "type": "bullets", "items": ["Best practice 1", "Best practice 2"]},
     {"id": "blk_011", "type": "heading", "level": 2, "text": "Troubleshooting"},
     {"id": "blk_012", "type": "paragraph", "text": "Common issues and solutions."}
   ]
 }'::jsonb,
 1, 'document_json', '{"doc_type": "training_doc"}'::jsonb, true)
ON CONFLICT (id) DO NOTHING;

INSERT INTO artifact_templates (id, team_id, template_type, name, description, content_markdown, document_json, schema_version, template_format, metadata, is_system) VALUES
('a0000000-0000-0000-0000-000000000015', NULL, 'document', 'Project Report (Structured)',
 'Structured project report with summary, objectives, progress, challenges, and next steps.',
 NULL,
 '{
   "schema_version": 1,
   "title": "Project Report",
   "document_type": "project_report",
   "blocks": [
     {"id": "blk_001", "type": "heading", "level": 1, "text": "Project Report: [Name]"},
     {"id": "blk_002", "type": "callout", "variant": "info", "text": "Period: [date range] — Status: [On Track/At Risk/Off Track]"},
     {"id": "blk_003", "type": "heading", "level": 2, "text": "Executive Summary"},
     {"id": "blk_004", "type": "paragraph", "text": "High-level summary of project status and key highlights."},
     {"id": "blk_005", "type": "heading", "level": 2, "text": "Objectives"},
     {"id": "blk_006", "type": "bullets", "items": ["Objective 1: status", "Objective 2: status"]},
     {"id": "blk_007", "type": "heading", "level": 2, "text": "Progress"},
     {"id": "blk_008", "type": "paragraph", "text": "What was accomplished in this period?"},
     {"id": "blk_009", "type": "heading", "level": 2, "text": "Challenges"},
     {"id": "blk_010", "type": "bullets", "items": ["Challenge 1: [description and mitigation]"]},
     {"id": "blk_011", "type": "heading", "level": 2, "text": "Next Steps"},
     {"id": "blk_012", "type": "numbered_list", "items": ["Action 1", "Action 2"]}
   ]
 }'::jsonb,
 1, 'document_json', '{"doc_type": "project_report"}'::jsonb, true)
ON CONFLICT (id) DO NOTHING;

INSERT INTO artifact_templates (id, team_id, template_type, name, description, content_markdown, document_json, schema_version, template_format, metadata, is_system) VALUES
('a0000000-0000-0000-0000-000000000016', NULL, 'document', 'Executive Brief (Structured)',
 'Structured executive brief with summary, key findings, strategic implications, and recommendations.',
 NULL,
 '{
   "schema_version": 1,
   "title": "Executive Brief",
   "document_type": "executive_brief",
   "blocks": [
     {"id": "blk_001", "type": "heading", "level": 1, "text": "Executive Brief: [Title]"},
     {"id": "blk_002", "type": "callout", "variant": "info", "text": "Date: [YYYY-MM-DD] — Prepared for: [Audience]"},
     {"id": "blk_003", "type": "heading", "level": 2, "text": "Summary"},
     {"id": "blk_004", "type": "paragraph", "text": "One-paragraph summary of the key message and recommendation."},
     {"id": "blk_005", "type": "heading", "level": 2, "text": "Key Findings"},
     {"id": "blk_006", "type": "bullets", "items": ["Finding 1", "Finding 2", "Finding 3"]},
     {"id": "blk_007", "type": "heading", "level": 2, "text": "Strategic Implications"},
     {"id": "blk_008", "type": "paragraph", "text": "What do these findings mean for the organization?"},
     {"id": "blk_009", "type": "heading", "level": 2, "text": "Recommendations"},
     {"id": "blk_010", "type": "numbered_list", "items": ["Recommendation 1", "Recommendation 2"]}
   ]
 }'::jsonb,
 1, 'document_json', '{"doc_type": "executive_brief"}'::jsonb, true)
ON CONFLICT (id) DO NOTHING;
