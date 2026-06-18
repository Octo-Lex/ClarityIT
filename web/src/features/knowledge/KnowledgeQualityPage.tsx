import { useQuery } from '@tanstack/react-query';
import { RefreshCw, CheckCircle2, AlertTriangle } from 'lucide-react';
import { api } from '@/api/client';
import { keys } from '@/api/keys';
import { useAuth } from '@/auth/context';
import { Card } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { StatusBadge } from '@/components/ui/status-badge';
import { TableSkeleton, ErrorState } from '@/components/PageState';
import { KnowledgeSourceBadge } from './KnowledgeSourceBadge';

export function KnowledgeQualityPage() {
  const { activeTeamId } = useAuth();
  const { data: report, isPending, isError, refetch } = useQuery({
    queryKey: keys.knowledge.quality(activeTeamId ?? ''),
    queryFn: () => api.getQualityReport(),
  });

  if (isPending) return <div data-testid="quality-loading" className="p-4"><TableSkeleton rows={6} cols={2} /></div>;
  if (isError) return <div data-testid="quality-error" className="p-4"><ErrorState message="Failed to load quality report" onRetry={() => refetch()} /></div>;
  if (!report) return null;

  return (
    <div className="mx-auto max-w-4xl space-y-8">
      <div className="flex items-center justify-between">
        <h1 className="font-heading text-2xl font-semibold tracking-tight">Knowledge Quality</h1>
        <Button variant="secondary" size="sm" onClick={() => refetch()}>
          <RefreshCw className="size-4" /> Refresh
        </Button>
      </div>

      {/* Summary cards */}
      <div className="grid grid-cols-2 gap-4 sm:grid-cols-4">
        <Card data-testid="quality-total" className="p-4 text-center">
          <div className="text-3xl font-bold text-primary">{report.total_items}</div>
          <div className="mt-1 text-sm text-muted-foreground">Total Items</div>
        </Card>
        <Card data-testid="quality-stale" className="p-4 text-center">
          <div className={`text-3xl font-bold ${report.stale_count > 0 ? 'text-warning' : 'text-success'}`}>{report.stale_count}</div>
          <div className="mt-1 text-sm text-muted-foreground">Stale</div>
        </Card>
        <Card data-testid="quality-duplicates" className="p-4 text-center">
          <div className={`text-3xl font-bold ${report.duplicate_count > 0 ? 'text-warning' : 'text-success'}`}>{report.duplicate_count}</div>
          <div className="mt-1 text-sm text-muted-foreground">Duplicates</div>
        </Card>
        <Card data-testid="quality-orphans" className="p-4 text-center">
          <div className={`text-3xl font-bold ${report.orphan_count > 0 ? 'text-destructive' : 'text-success'}`}>{report.orphan_count}</div>
          <div className="mt-1 text-sm text-muted-foreground">Orphans</div>
        </Card>
      </div>

      {/* By type */}
      <div>
        <h2 className="mb-3 font-heading text-lg font-semibold">By Source Type</h2>
        <div className="flex flex-wrap gap-2">
          {Object.entries(report.by_type).map(([type, count]) => (
            <div key={type} className="flex items-center gap-1.5 rounded-full border border-border px-3 py-1 text-sm">
              <KnowledgeSourceBadge sourceType={type} /> <span className="font-medium">{count}</span>
            </div>
          ))}
          {Object.keys(report.by_type).length === 0 && (
            <span className="text-sm text-muted-foreground">No indexed items yet.</span>
          )}
        </div>
      </div>

      {/* Stale items */}
      {report.stale_items.length > 0 && (
        <div>
          <h2 className="mb-3 font-heading text-lg font-semibold" data-testid="quality-stale-section">Stale Items ({report.stale_count})</h2>
          <div className="space-y-2">
            {report.stale_items.map((item) => (
              <Card key={item.knowledge_item_id} data-testid="quality-stale-item" className="flex items-center justify-between p-3">
                <div className="flex items-center gap-2">
                  <KnowledgeSourceBadge sourceType={item.source_type} />
                  <span className="font-medium">{item.title}</span>
                </div>
                <StatusBadge tone="warning">{item.days_stale} days stale</StatusBadge>
              </Card>
            ))}
          </div>
        </div>
      )}

      {/* Duplicates */}
      {report.duplicate_groups.length > 0 && (
        <div>
          <h2 className="mb-3 font-heading text-lg font-semibold" data-testid="quality-dup-section">Duplicate Groups ({report.duplicate_groups.length})</h2>
          <div className="space-y-3">
            {report.duplicate_groups.map((group, i) => (
              <Card key={i} data-testid="quality-dup-group" className="p-3">
                <div className="mb-2 flex items-center justify-between">
                  <span className="font-mono text-xs text-muted-foreground">Hash: {group.content_hash}</span>
                  <StatusBadge tone="warning">{group.count} copies</StatusBadge>
                </div>
                <div className="space-y-1">
                  {group.items.map((item) => (
                    <div key={item.knowledge_item_id} className="flex items-center gap-2 text-sm">
                      <KnowledgeSourceBadge sourceType={item.source_type} />
                      <span>{item.title}</span>
                    </div>
                  ))}
                </div>
              </Card>
            ))}
          </div>
        </div>
      )}

      {/* Orphans */}
      {report.orphan_items.length > 0 && (
        <div>
          <h2 className="mb-3 font-heading text-lg font-semibold" data-testid="quality-orphan-section">Orphaned Items ({report.orphan_count})</h2>
          <div className="space-y-2">
            {report.orphan_items.map((item) => (
              <Card key={item.knowledge_item_id} data-testid="quality-orphan-item" className="flex items-center gap-2 border-destructive/30 p-3">
                <KnowledgeSourceBadge sourceType={item.source_type} />
                <span className="font-medium">{item.title}</span>
                <span className="ml-auto flex items-center gap-1 text-xs text-destructive">
                  <AlertTriangle className="size-3" /> Source deleted
                </span>
              </Card>
            ))}
          </div>
        </div>
      )}

      {/* All clean */}
      {report.stale_count === 0 && report.duplicate_count === 0 && report.orphan_count === 0 && report.total_items > 0 && (
        <div data-testid="quality-all-clean" className="flex flex-col items-center py-12 text-center text-success">
          <CheckCircle2 className="mb-2 size-8" />
          All knowledge items are fresh, unique, and have valid sources.
        </div>
      )}
    </div>
  );
}
