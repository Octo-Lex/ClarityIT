import React from 'react';

const REASON_LABELS: Record<string, string> = {
  explicit_link: 'Linked',
  context_edge: 'Connected',
  shared_reference: 'Shared Ref',
  content_similarity: 'Similar',
  same_source_family: 'Same Type',
  recent_related: 'Recent',
};

const REASON_COLORS: Record<string, string> = {
  explicit_link: 'bg-blue-100 text-blue-700 dark:bg-blue-900 dark:text-blue-300',
  context_edge: 'bg-purple-100 text-purple-700 dark:bg-purple-900 dark:text-purple-300',
  shared_reference: 'bg-cyan-100 text-cyan-700 dark:bg-cyan-900 dark:text-cyan-300',
  content_similarity: 'bg-amber-100 text-amber-700 dark:bg-amber-900 dark:text-amber-300',
  same_source_family: 'bg-slate-100 text-slate-600 dark:bg-slate-700 dark:text-slate-300',
  recent_related: 'bg-gray-100 text-gray-500 dark:bg-gray-800 dark:text-gray-400',
};

export function RelatedKnowledgeReasonBadge({ reason }: { reason: string }) {
  const label = REASON_LABELS[reason] ?? reason;
  const color = REASON_COLORS[reason] ?? 'bg-slate-100 text-slate-600 dark:bg-slate-700 dark:text-slate-300';
  return (
    <span
      data-testid="related-reason-badge"
      className={`inline-flex items-center px-1.5 py-0.5 text-xs font-medium rounded ${color}`}
    >
      {label}
    </span>
  );
}
