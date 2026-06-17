import React from 'react';

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

const SOURCE_ICONS: Record<string, string> = {
  clarity_document: '📄',
  artifact: '📎',
  meeting_summary: '🗓️',
  status_report: '📊',
  presentation: '📽️',
  template: '📋',
  work_item: '✅',
  incident: '🔥',
  asset: '🖥️',
  remediation: '🔧',
  approval: '🔑',
  context_node: '🔗',
  project: '📁',
};

export function KnowledgeSourceBadge({ sourceType }: { sourceType: string }) {
  const label = SOURCE_LABELS[sourceType] ?? sourceType;
  const icon = SOURCE_ICONS[sourceType] ?? '📦';
  return (
    <span
      data-testid="knowledge-source-badge"
      className="inline-flex items-center gap-1 px-2 py-0.5 text-xs font-medium rounded-full bg-slate-100 text-slate-700 dark:bg-slate-800 dark:text-slate-300"
    >
      <span>{icon}</span> {label}
    </span>
  );
}

export { SOURCE_LABELS, SOURCE_ICONS };
