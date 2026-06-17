import React from 'react';
import { SOURCE_LABELS, SOURCE_ICONS } from './KnowledgeSourceBadge';

const ALL_FILTERS = [
  { key: 'all', label: 'All' },
  { key: 'clarity_document', label: 'Documents' },
  { key: 'artifact', label: 'Artifacts' },
  { key: 'meeting_summary', label: 'Meetings' },
  { key: 'status_report', label: 'Status Reports' },
  { key: 'presentation', label: 'Presentations' },
  { key: 'template', label: 'Templates' },
  { key: 'work_item', label: 'Work Items' },
  { key: 'incident', label: 'Incidents' },
  { key: 'asset', label: 'Assets' },
  { key: 'remediation', label: 'Remediations' },
  { key: 'approval', label: 'Approvals' },
  { key: 'context_node', label: 'Context' },
  { key: 'project', label: 'Projects', disabled: true },
];

export function SearchFilters({
  active,
  onSelect,
}: {
  active: string;
  onSelect: (sourceType: string) => void;
}) {
  return (
    <div
      data-testid="search-filters"
      className="flex flex-wrap gap-2"
    >
      {ALL_FILTERS.map((f) => {
        const isActive = active === f.key;
        const icon = SOURCE_ICONS[f.key] ?? '📦';
        return (
          <button
            key={f.key}
            data-testid={`filter-${f.key}`}
            disabled={f.disabled}
            onClick={() => onSelect(f.key)}
            className={`px-3 py-1 text-sm rounded-full transition-colors ${
              f.disabled
                ? 'opacity-40 cursor-not-allowed bg-slate-100 text-slate-400'
                : isActive
                ? 'bg-blue-600 text-white'
                : 'bg-slate-100 text-slate-700 hover:bg-slate-200 dark:bg-slate-800 dark:text-slate-300 dark:hover:bg-slate-700'
            }`}
          >
            <span className="mr-1">{icon}</span>
            {f.label}
          </button>
        );
      })}
    </div>
  );
}
