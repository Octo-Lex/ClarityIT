import { useState, useEffect } from 'react';
import { api, ApiError, type ContextQuality } from '../../api/client';

export default function ContextQualityPanel() {
  const [data, setData] = useState<ContextQuality | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [reviewing, setReviewing] = useState<string | null>(null);

  useEffect(() => {
    let active = true;
    api.getContextQuality({ stale_days: 30, confidence_threshold: 0.60 })
      .then((d) => { if (active) { setData(d); setLoading(false); } })
      .catch((e: unknown) => {
        if (active) {
          setError(e instanceof ApiError ? e.message : 'Failed to load context quality');
          setLoading(false);
        }
      });
    return () => { active = false; };
  }, []);

  const handleConfirm = (relationId: string) => {
    setReviewing(relationId);
    api.confirmRelation(relationId, 'Reviewed during context quality cleanup')
      .then(() => {
        // Refresh data
        api.getContextQuality({ stale_days: 30, confidence_threshold: 0.60 })
          .then((d) => setData(d));
        setReviewing(null);
      })
      .catch(() => setReviewing(null));
  };

  const handleDismiss = (relationId: string) => {
    setReviewing(relationId);
    api.dismissRelation(relationId, 'Reviewed during context quality cleanup')
      .then(() => {
        api.getContextQuality({ stale_days: 30, confidence_threshold: 0.60 })
          .then((d) => setData(d));
        setReviewing(null);
      })
      .catch(() => setReviewing(null));
  };

  if (loading) return null;

  if (error) {
    return (
      <div className="rounded-xl border border-border bg-surface p-4" data-testid="quality-error-container">
        <div className="text-sm text-destructive" data-testid="quality-error">{error}</div>
      </div>
    );
  }

  if (!data) return null;

  const scoreColor = data.quality_score >= 80 ? 'text-success'
    : data.quality_score >= 50 ? 'text-warning' : 'text-warning';

  return (
    <div className="space-y-3" data-testid="quality-panel">
      <h2 className="text-lg font-semibold">Context Graph Quality</h2>

      {/* Advisory warning */}
      <div className="p-3 bg-warning/20 border border-warning/40 rounded text-sm text-warning" data-testid="quality-warning">
        ⚠ Context quality controls are advisory only. Confirming or dismissing a relation does not delete graph data.
      </div>

      {/* Score and summary */}
      <div className="rounded-xl border border-border bg-surface p-4" data-testid="quality-summary">
        <div className="flex items-center gap-4 mb-3">
          <div>
            <span className="text-sm text-muted-foreground">Quality Score: </span>
            <span className={`text-2xl font-bold ${scoreColor}`} data-testid="quality-score">{data.quality_score}</span>
            <span className="text-sm text-muted-foreground"> / 100</span>
          </div>
        </div>
        <div className="grid grid-cols-4 gap-3 text-xs">
          <div><span className="text-muted-foreground">Nodes: </span>{data.summary.total_nodes}</div>
          <div><span className="text-muted-foreground">Relations: </span>{data.summary.total_relations}</div>
          <div><span className="text-muted-foreground">Confirmed: </span>{data.summary.confirmed_relations}</div>
          <div><span className="text-muted-foreground">Dismissed: </span>{data.summary.dismissed_relations}</div>
        </div>
      </div>

      {/* Stale nodes */}
      {data.stale_nodes.length > 0 && (
        <div className="rounded-xl border border-border bg-surface p-4" data-testid="quality-stale">
          <h3 className="text-sm font-semibold mb-2">Stale Nodes ({data.stale_nodes.length})</h3>
          {data.stale_nodes.slice(0, 10).map((n) => (
            <div key={n.node_id} className="text-xs mb-1 flex justify-between">
              <span>{n.label} <span className="text-muted-foreground">({n.node_type})</span></span>
              <span className="text-warning" data-testid={`stale-${n.node_id}`}>{n.days_stale}d stale</span>
            </div>
          ))}
        </div>
      )}

      {/* Low-confidence relations */}
      {data.low_confidence_relations.length > 0 && (
        <div className="rounded-xl border border-border bg-surface p-4" data-testid="quality-low-conf">
          <h3 className="text-sm font-semibold mb-2">Low-Confidence Relations ({data.low_confidence_relations.length})</h3>
          {data.low_confidence_relations.slice(0, 10).map((r) => (
            <div key={r.relation_id} className="text-xs mb-2 flex justify-between items-center">
              <span data-testid={`lowconf-${r.relation_id}`}>
                {r.relation_type} (conf: {(r.confidence * 100).toFixed(0)}%)
              </span>
              <span className="flex gap-1">
                <button
                  onClick={() => handleConfirm(r.relation_id)}
                  disabled={reviewing === r.relation_id}
                  className="px-2 py-0.5 text-xs bg-success/15 text-success rounded disabled:opacity-50"
                  data-testid={`confirm-${r.relation_id}`}
                >
                  Confirm
                </button>
                <button
                  onClick={() => handleDismiss(r.relation_id)}
                  disabled={reviewing === r.relation_id}
                  className="px-2 py-0.5 text-xs bg-destructive/15 text-destructive rounded disabled:opacity-50"
                  data-testid={`dismiss-${r.relation_id}`}
                >
                  Dismiss
                </button>
              </span>
            </div>
          ))}
        </div>
      )}

      {/* Conflicting relations */}
      {data.conflicting_relations.length > 0 && (
        <div className="rounded-xl border border-border bg-surface p-4" data-testid="quality-conflicts">
          <h3 className="text-sm font-semibold mb-2">Conflicting Relations ({data.conflicting_relations.length})</h3>
          {data.conflicting_relations.slice(0, 10).map((r) => (
            <div key={r.relation_id} className="text-xs mb-2 flex justify-between items-center">
              <span data-testid={`conflict-${r.relation_id}`}>{r.conflict_reason}</span>
              <span className="flex gap-1">
                <button
                  onClick={() => handleConfirm(r.relation_id)}
                  disabled={reviewing === r.relation_id}
                  className="px-2 py-0.5 text-xs bg-success/15 text-success rounded disabled:opacity-50"
                  data-testid={`confirm-conflict-${r.relation_id}`}
                >
                  Confirm
                </button>
                <button
                  onClick={() => handleDismiss(r.relation_id)}
                  disabled={reviewing === r.relation_id}
                  className="px-2 py-0.5 text-xs bg-destructive/15 text-destructive rounded disabled:opacity-50"
                  data-testid={`dismiss-conflict-${r.relation_id}`}
                >
                  Dismiss
                </button>
              </span>
            </div>
          ))}
        </div>
      )}

      {/* Empty state */}
      {data.stale_nodes.length === 0 && data.low_confidence_relations.length === 0 && data.conflicting_relations.length === 0 && (
        <div className="rounded-xl border border-border bg-surface p-4" data-testid="quality-empty">
          <p className="text-sm text-muted-foreground">No quality issues detected.</p>
        </div>
      )}
    </div>
  );
}
