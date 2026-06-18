import { StatusBadge } from '@/components/ui/status-badge';

const SOURCE_LABELS: Record<string, string> = {
  clarity_document: 'Document',
  artifact: 'Artifact',
  meeting_summary: 'Meeting',
  status_report: 'Status Report',
  presentation: 'Presentation',
  template: 'Template',
  work_item: 'Work Item',
  incident: 'Incident',
  asset: 'Asset',
  remediation: 'Remediation',
  approval: 'Approval',
  context_node: 'Context',
  project: 'Project',
};

export function KnowledgeSourceBadge({ sourceType }: { sourceType: string }) {
  const label = SOURCE_LABELS[sourceType] ?? sourceType;
  return (
    <StatusBadge data-testid="knowledge-source-badge" tone="neutral">
      {label}
    </StatusBadge>
  );
}

export { SOURCE_LABELS };
