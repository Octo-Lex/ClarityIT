-- 035_artifact_templates.sql
-- v1.3 Track 5: Reusable artifact templates.
-- System templates (team_id NULL) are available to all teams.
-- Team templates (team_id NOT NULL) are team-scoped custom templates.

CREATE TABLE IF NOT EXISTS artifact_templates (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    team_id         UUID REFERENCES teams(id) ON DELETE CASCADE,
    template_type   TEXT NOT NULL,
    name            TEXT NOT NULL,
    description     TEXT,
    content_markdown TEXT NOT NULL,
    metadata        JSONB NOT NULL DEFAULT '{}',
    is_system       BOOLEAN NOT NULL DEFAULT false,
    created_by      UUID,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT atpl_template_type_check CHECK (
        template_type IN ('document', 'report', 'meeting_summary', 'status_report',
                          'decision_memo', 'training_deck', 'presentation')
    ),
    CONSTRAINT atpl_metadata_object CHECK (jsonb_typeof(metadata) = 'object'),
    CONSTRAINT atpl_name_nonempty CHECK (length(btrim(name)) > 0),
    CONSTRAINT atpl_content_nonempty CHECK (length(btrim(content_markdown)) > 0),
    CONSTRAINT atpl_system_no_team CHECK (
        (is_system = true AND team_id IS NULL) OR
        (is_system = false AND team_id IS NOT NULL)
    )
);

CREATE INDEX IF NOT EXISTS idx_atpl_team ON artifact_templates(team_id);
CREATE INDEX IF NOT EXISTS idx_atpl_type ON artifact_templates(template_type);
CREATE INDEX IF NOT EXISTS idx_atpl_system ON artifact_templates(is_system) WHERE is_system = true;

-- ─── Seed System Templates ───
-- Use fixed UUIDs for deterministic system templates.

INSERT INTO artifact_templates (id, team_id, template_type, name, description, content_markdown, metadata, is_system) VALUES
('a0000000-0000-0000-0000-000000000001', NULL, 'status_report', 'Weekly Status Report',
 'Standard weekly status report template with summary, milestones, risks, and metrics sections.',
 '# Weekly Status Report

**Period:** [Week of YYYY-MM-DD]

## Summary

- Key accomplishments this week:
- Items in progress:
- Blockers:

## Milestones

- [Achieved] [Milestone name] — [date]
- [Upcoming] [Milestone name] — [target date]

## Risks

- [Risk description and mitigation]

## Metrics

- Work items completed:
- Incidents:
- Open action items:

## Next Week

- Planned priorities:
',
 '{"sections": ["summary", "milestones", "risks", "metrics"]}', true)
ON CONFLICT DO NOTHING;

INSERT INTO artifact_templates (id, team_id, template_type, name, description, content_markdown, metadata, is_system) VALUES
('a0000000-0000-0000-0000-000000000002', NULL, 'meeting_summary', 'Meeting Summary',
 'Standard meeting summary template with attendees, agenda, decisions, and action items.',
 '# Meeting: [Title]

**Date:** [YYYY-MM-DD]
**Duration:** [minutes]
**Attendees:**

## Agenda

1. [Agenda item]

## Decisions

- [Decision text]

## Action Items

- [ ] [Action] — Assignee: [name] — Due: [date]

## Notes

[Optional narrative notes]
',
 '{"sections": ["attendees", "agenda", "decisions", "actions"]}', true)
ON CONFLICT DO NOTHING;

INSERT INTO artifact_templates (id, team_id, template_type, name, description, content_markdown, metadata, is_system) VALUES
('a0000000-0000-0000-0000-000000000003', NULL, 'decision_memo', 'Decision Memo',
 'Decision memo template for documenting key architectural or operational decisions.',
 '# Decision: [Title]

**Date:** [YYYY-MM-DD]
**Decided By:** [name/role]

## Context

[What problem are we solving? What constraints exist?]

## Options Considered

### Option A: [Name]
- Pros:
- Cons:

### Option B: [Name]
- Pros:
- Cons:

## Decision

[What did we decide and why?]

## Impact

- Systems affected:
- Teams affected:
- Follow-up actions:
',
 '{"format": "ADR-lite"}', true)
ON CONFLICT DO NOTHING;

INSERT INTO artifact_templates (id, team_id, template_type, name, description, content_markdown, metadata, is_system) VALUES
('a0000000-0000-0000-0000-000000000004', NULL, 'report', 'Incident Summary',
 'Post-incident summary template for documenting what happened and lessons learned.',
 '# Incident Summary: [Title]

**Severity:** [sev1-sev5]
**Date:** [YYYY-MM-DD]
**Duration:** [hours/minutes]

## Timeline

- [time] — [event]
- [time] — [event]

## Impact

- Services affected:
- Users impacted:
- Data impact:

## Root Cause

[What caused the incident?]

## Resolution

[How was it resolved?]

## Lessons Learned

- What went well:
- What went poorly:
- Action items for prevention:
',
 '{"type": "post-mortem"}', true)
ON CONFLICT DO NOTHING;

INSERT INTO artifact_templates (id, team_id, template_type, name, description, content_markdown, metadata, is_system) VALUES
('a0000000-0000-0000-0000-000000000005', NULL, 'document', 'Architecture Walkthrough',
 'Document template for describing system architecture and design decisions.',
 '# Architecture: [System Name]

**Last Updated:** [YYYY-MM-DD]
**Status:** [Draft/Review/Current]

## Overview

[Brief description of the system and its purpose]

## Components

### [Component Name]
- Purpose:
- Technology:
- Dependencies:

## Data Flow

[Describe how data moves through the system]

## Key Design Decisions

- [Decision and rationale]

## Security Boundaries

- [Trust boundaries and access controls]

## Operational Notes

- Deployment:
- Monitoring:
- Backup/Recovery:
',
 '{"format": "architecture-doc"}', true)
ON CONFLICT DO NOTHING;

INSERT INTO artifact_templates (id, team_id, template_type, name, description, content_markdown, metadata, is_system) VALUES
('a0000000-0000-0000-0000-000000000006', NULL, 'training_deck', 'Training Deck Prompt',
 'Template for creating training presentation content with structured sections.',
 '# Training: [Topic]

**Audience:** [Who is this for?]
**Duration:** [estimated minutes]

## Learning Objectives

By the end of this session, participants will be able to:
1. [Objective 1]
2. [Objective 2]
3. [Objective 3]

## Outline

### Introduction (5 min)
- Overview of [topic]
- Why it matters

### Core Concepts (15 min)
- [Key concept 1]
- [Key concept 2]

### Hands-On Exercise (20 min)
- [Exercise description]

### Q&A and Wrap-up (10 min)

## Materials Needed

- [Materials list]

## Assessment

- [How to verify learning]
',
 '{"format": "training-prompt"}', true)
ON CONFLICT DO NOTHING;
