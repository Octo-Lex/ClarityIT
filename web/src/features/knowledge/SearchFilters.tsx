import { SOURCE_LABELS } from './KnowledgeSourceBadge';
import { cn } from '@/lib/utils';

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
    <div data-testid="search-filters" className="flex flex-wrap gap-2">
      {ALL_FILTERS.map((f) => {
        const isActive = active === f.key;
        return (
          <button
            key={f.key}
            data-testid={`filter-${f.key}`}
            disabled={f.disabled}
            onClick={() => onSelect(f.key)}
            className={cn(
              'rounded-full px-3 py-1 text-sm transition-colors',
              f.disabled && 'cursor-not-allowed bg-muted text-muted-foreground opacity-40',
              !f.disabled && isActive && 'bg-primary text-primary-foreground',
              !f.disabled && !isActive && 'bg-muted text-muted-foreground hover:bg-accent hover:text-foreground',
            )}
          >
            {f.label}
          </button>
        );
      })}
    </div>
  );
}
