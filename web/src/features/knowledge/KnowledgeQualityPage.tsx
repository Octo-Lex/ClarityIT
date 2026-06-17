import { useState, useEffect } from 'react';
import { api, type KnowledgeQualityReport } from '../../api/client';
import { KnowledgeSourceBadge } from './KnowledgeSourceBadge';

export function KnowledgeQualityPage() {
  const [report, setReport] = useState<KnowledgeQualityReport | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const load = async () => {
    setLoading(true);
    setError(null);
    try {
      const resp = await api.getQualityReport();
      setReport(resp);
    } catch {
      setError('Failed to load quality report');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => { load(); }, []);

  if (loading) return <div data-testid="quality-loading" className="p-8 text-center text-gray-500">Loading quality report…</div>;
  if (error) return <div data-testid="quality-error" className="p-8 text-center text-red-500">{error}</div>;
  if (!report) return null;

  return (
    <div className="max-w-4xl mx-auto p-6">
      <h1 className="text-2xl font-bold mb-6">Knowledge Quality</h1>

      {/* Summary Cards */}
      <div className="grid grid-cols-4 gap-4 mb-8">
        <div data-testid="quality-total" className="p-4 border rounded-lg text-center">
          <div className="text-3xl font-bold text-indigo-600">{report.total_items}</div>
          <div className="text-sm text-gray-500 mt-1">Total Items</div>
        </div>
        <div data-testid="quality-stale" className="p-4 border rounded-lg text-center">
          <div className={`text-3xl font-bold ${report.stale_count > 0 ? 'text-amber-600' : 'text-green-600'}`}>{report.stale_count}</div>
          <div className="text-sm text-gray-500 mt-1">Stale</div>
        </div>
        <div data-testid="quality-duplicates" className="p-4 border rounded-lg text-center">
          <div className={`text-3xl font-bold ${report.duplicate_count > 0 ? 'text-orange-600' : 'text-green-600'}`}>{report.duplicate_count}</div>
          <div className="text-sm text-gray-500 mt-1">Duplicates</div>
        </div>
        <div data-testid="quality-orphans" className="p-4 border rounded-lg text-center">
          <div className={`text-3xl font-bold ${report.orphan_count > 0 ? 'text-red-600' : 'text-green-600'}`}>{report.orphan_count}</div>
          <div className="text-sm text-gray-500 mt-1">Orphans</div>
        </div>
      </div>

      {/* By Type Breakdown */}
      <div className="mb-8">
        <h2 className="text-lg font-semibold mb-3">By Source Type</h2>
        <div className="flex flex-wrap gap-2">
          {Object.entries(report.by_type).map(([type, count]) => (
            <div key={type} className="px-3 py-1 border rounded-full text-sm">
              <KnowledgeSourceBadge sourceType={type} /> <span className="font-medium">{count}</span>
            </div>
          ))}
          {Object.keys(report.by_type).length === 0 && (
            <span className="text-gray-400 text-sm">No indexed items yet.</span>
          )}
        </div>
      </div>

      {/* Stale Items */}
      {report.stale_items.length > 0 && (
        <div className="mb-8">
          <h2 className="text-lg font-semibold mb-3" data-testid="quality-stale-section">Stale Items ({report.stale_count})</h2>
          <div className="space-y-2">
            {report.stale_items.map((item) => (
              <div key={item.knowledge_item_id} data-testid="quality-stale-item" className="p-3 border rounded-lg">
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-2">
                    <KnowledgeSourceBadge sourceType={item.source_type} />
                    <span className="font-medium">{item.title}</span>
                  </div>
                  <span className="text-amber-600 text-sm font-medium">{item.days_stale} days stale</span>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Duplicates */}
      {report.duplicate_groups.length > 0 && (
        <div className="mb-8">
          <h2 className="text-lg font-semibold mb-3" data-testid="quality-dup-section">Duplicate Groups ({report.duplicate_groups.length})</h2>
          <div className="space-y-3">
            {report.duplicate_groups.map((group, i) => (
              <div key={i} data-testid="quality-dup-group" className="p-3 border rounded-lg">
                <div className="flex items-center justify-between mb-2">
                  <span className="text-sm text-gray-500">Hash: {group.content_hash}</span>
                  <span className="text-orange-600 text-sm font-medium">{group.count} copies</span>
                </div>
                <div className="space-y-1">
                  {group.items.map((item) => (
                    <div key={item.knowledge_item_id} className="text-sm flex items-center gap-2">
                      <KnowledgeSourceBadge sourceType={item.source_type} />
                      <span>{item.title}</span>
                    </div>
                  ))}
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Orphans */}
      {report.orphan_items.length > 0 && (
        <div className="mb-8">
          <h2 className="text-lg font-semibold mb-3" data-testid="quality-orphan-section">Orphaned Items ({report.orphan_count})</h2>
          <div className="space-y-2">
            {report.orphan_items.map((item) => (
              <div key={item.knowledge_item_id} data-testid="quality-orphan-item" className="p-3 border rounded-lg border-red-200">
                <div className="flex items-center gap-2">
                  <KnowledgeSourceBadge sourceType={item.source_type} />
                  <span className="font-medium">{item.title}</span>
                  <span className="text-red-500 text-xs ml-auto">Source deleted</span>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* All Clean */}
      {report.stale_count === 0 && report.duplicate_count === 0 && report.orphan_count === 0 && report.total_items > 0 && (
        <div data-testid="quality-all-clean" className="text-center py-12 text-green-600">
          ✓ All knowledge items are fresh, unique, and have valid sources.
        </div>
      )}

      {/* Refresh */}
      <button onClick={load} className="mt-4 px-4 py-2 border rounded-lg text-sm">Refresh Report</button>
    </div>
  );
}
