import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { ChevronDown, ChevronRight } from 'lucide-react';
import { api } from '@/api/client';
import { keys } from '@/api/keys';
import { useAuth } from '@/auth/context';
import { InlineSpinner } from '@/components/PageState';
import { RelatedKnowledgeCard } from './RelatedKnowledgeCard';

export function RelatedKnowledgePanel({
  sourceType,
  sourceId,
  title = 'Related Knowledge',
}: {
  sourceType: string;
  sourceId: string;
  title?: string;
}) {
  const { activeTeamId } = useAuth();
  const [collapsed, setCollapsed] = useState(false);

  // Non-blocking: a failure here must not break the host page (e.g. an incident
  // detail). React Query surfaces it as `error` but the panel degrades silently.
  const { data, isPending, isError } = useQuery({
    queryKey: keys.knowledge.related(activeTeamId ?? '', sourceType, sourceId),
    queryFn: () => api.getRelatedKnowledge(sourceType, sourceId, 8),
    enabled: !!sourceType && !!sourceId,
  });

  const items = data?.related ?? [];

  return (
    <div data-testid="related-knowledge-panel" className="overflow-hidden rounded-lg border border-border">
      <button
        data-testid="related-panel-toggle"
        onClick={() => setCollapsed(!collapsed)}
        className="flex w-full items-center justify-between bg-muted px-4 py-3 transition-colors hover:bg-accent"
      >
        <h3 className="text-sm font-semibold">{title}</h3>
        {collapsed ? <ChevronRight className="size-4 text-muted-foreground" /> : <ChevronDown className="size-4 text-muted-foreground" />}
      </button>

      {!collapsed && (
        <div className="p-4">
          {isPending && (
            <div data-testid="related-loading">
              <InlineSpinner label="Loading related items…" />
            </div>
          )}

          {!isPending && isError && (
            <div data-testid="related-error" className="py-4 text-center text-sm text-destructive">
              Failed to load related items.
            </div>
          )}

          {!isPending && !isError && items.length === 0 && (
            <div data-testid="related-empty" className="py-4 text-center text-sm text-muted-foreground">
              No related items found.
            </div>
          )}

          {!isPending && !isError && items.length > 0 && (
            <div data-testid="related-items" className="space-y-2">
              {items.map((item, i) => (
                <RelatedKnowledgeCard key={`${item.source_type}-${item.source_id}-${i}`} item={item} />
              ))}
            </div>
          )}
        </div>
      )}
    </div>
  );
}
