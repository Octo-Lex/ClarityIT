import React, { useState, useEffect, useCallback } from 'react';
import { api, ApiError } from '../../api/client';
import type { RelatedKnowledgeItem } from '../../api/client';
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
  const [items, setItems] = useState<RelatedKnowledgeItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(false);
  const [collapsed, setCollapsed] = useState(false);

  const fetchRelated = useCallback(async () => {
    setLoading(true);
    setError(false);
    try {
      const resp = await api.getRelatedKnowledge(sourceType, sourceId, 8);
      setItems(resp.related ?? []);
    } catch (e) {
      // Panel does not block main page on failure
      setError(true);
      setItems([]);
    } finally {
      setLoading(false);
    }
  }, [sourceType, sourceId]);

  useEffect(() => {
    if (sourceType && sourceId) {
      fetchRelated();
    }
  }, [sourceType, sourceId]);

  return (
    <div
      data-testid="related-knowledge-panel"
      className="border border-slate-200 dark:border-slate-700 rounded-lg overflow-hidden"
    >
      {/* Header with collapse toggle */}
      <button
        data-testid="related-panel-toggle"
        onClick={() => setCollapsed(!collapsed)}
        className="w-full flex items-center justify-between px-4 py-3 bg-slate-50 dark:bg-slate-800 hover:bg-slate-100 dark:hover:bg-slate-700 transition-colors"
      >
        <h3 className="text-sm font-semibold text-slate-700 dark:text-slate-300">
          {title}
        </h3>
        <span className="text-xs text-slate-400">
          {collapsed ? '▶' : '▼'}
        </span>
      </button>

      {!collapsed && (
        <div className="p-4">
          {/* Loading state */}
          {loading && (
            <div data-testid="related-loading" className="text-center py-6">
              <div className="inline-block animate-spin rounded-full h-6 w-6 border-b-2 border-blue-600 mb-2" />
              <p className="text-xs text-slate-500">Loading related items...</p>
            </div>
          )}

          {/* Error state */}
          {!loading && error && (
            <div data-testid="related-error" className="text-center py-4 text-red-500 dark:text-red-400">
              <p className="text-sm">Failed to load related items.</p>
            </div>
          )}

          {/* Empty state */}
          {!loading && !error && items.length === 0 && (
            <div data-testid="related-empty" className="text-center py-4">
              <p className="text-sm text-slate-400">No related items found.</p>
            </div>
          )}

          {/* Results */}
          {!loading && !error && items.length > 0 && (
            <div data-testid="related-items" className="space-y-2">
              {items.map((item, i) => (
                <RelatedKnowledgeCard
                  key={`${item.source_type}-${item.source_id}-${i}`}
                  item={item}
                />
              ))}
            </div>
          )}
        </div>
      )}
    </div>
  );
}
