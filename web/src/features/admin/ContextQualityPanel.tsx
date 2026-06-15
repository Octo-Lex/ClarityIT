import { useState, useEffect } from 'react';
import { api, ApiError } from '../../api/client';

interface StaleNode {
  node_id: string;
  node_type: string;
  label: string;
  days_stale: number;
  reason: string;
}

interface LowConfRel {
  relation_id: string;
  relation_type: string;
  confidence: number;
  reason: string;
}

interface ConflictRel {
  relation_id: string;
  relation_type: string;
  conflict_reason: string;
}

interface QualityData {
  quality_score: number;
  advisory_only: boolean;
  summary: {
    total_nodes: number;
    total_relations: number;
    stale_nodes: number;
    low_confidence_relations: number;
    conflicting_relations: number;
    confirmed_relations: number;
    dismissed_relations: number;
  };
  stale_nodes: StaleNode[];
  low_confidence_relations: LowConfRel[];
  conflicting_relations: ConflictRel[];
}

export default function ContextQualityPanel() {
  const [data, setData] = useState<QualityData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [reviewing, setReviewing] = useState<string | null>(null);

  useEffect(() => {
    let active = true;
    api.getContextQuality({ stale_days: 30, confidence_threshold: 0.60 })
      .then((d: QualityData) => { if (active) { setData(d); setLoading(false); } })
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
          .then((d: QualityData) => setData(d));
        setReviewing(null);
      })
      .catch(() => setReviewing(null));
  };

  const handleDismiss = (relationId: string) => {
    setReviewing(relationId);
    api.dismissRelation(relationId, 'Reviewed during context quality cleanup')
      .then(() => {
        api.getContextQuality({ stale_days: 30, confidence_threshold: 0.60 })
          .then((d: QualityData) => setData(d));
        setReviewing(null);
      })
      .catch(() => setReviewing(null));
  };

  if (loading) return null;

  if (error) {
    return (
      <div className="card p-4" data-testid="quality-error-container">
        <div className="text-sm text-red-400" data-testid="quality-error">{error}</div>
      </div>
    );
  }

  if (!data) return null;

  const scoreColor = data.quality_score >= 80 ? 'text-green-400'
    : data.quality_score >= 50 ? 'text-yellow-400' : 'text-orange-400';

  return (
    <div className="space-y-3" data-testid="quality-panel">
      <h2 className="text-lg font-semibold">Context Graph Quality</h2>

      {/* Advisory warning */}
      <div className="p-3 bg-yellow-900/20 border border-yellow-700 rounded text-sm text-yellow-300" data-testid="quality-warning">
        ⚠ Context quality controls are advisory only. Confirming or dismissing a relation does not delete graph data.
      </div>

      {/* Score and summary */}
      <div className="card p-4" data-testid="quality-summary">
        <div className="flex items-center gap-4 mb-3">
          <div>
            <span className="text-sm text-[var(--text-muted)]">Quality Score: </span>
            <span className={`text-2xl font-bold ${scoreColor}`} data-testid="quality-score">{data.quality_score}</span>
            <span className="text-sm text-[var(--text-muted)]"> / 100</span>
          </div>
        </div>
        <div className="grid grid-cols-4 gap-3 text-xs">
          <div><span className="text-[var(--text-muted)]">Nodes: </span>{data.summary.total_nodes}</div>
          <div><span className="text-[var(--text-muted)]">Relations: </span>{data.summary.total_relations}</div>
          <div><span className="text-[var(--text-muted)]">Confirmed: </span>{data.summary.confirmed_relations}</div>
          <div><span className="text-[var(--text-muted)]">Dismissed: </span>{data.summary.dismissed_relations}</div>
        </div>
      </div>

      {/* Stale nodes */}
      {data.stale_nodes.length > 0 && (
        <div className="card p-4" data-testid="quality-stale">
          <h3 className="text-sm font-semibold mb-2">Stale Nodes ({data.stale_nodes.length})</h3>
          {data.stale_nodes.slice(0, 10).map((n) => (
            <div key={n.node_id} className="text-xs mb-1 flex justify-between">
              <span>{n.label} <span className="text-[var(--text-muted)]">({n.node_type})</span></span>
              <span className="text-orange-400" data-testid={`stale-${n.node_id}`}>{n.days_stale}d stale</span>
            </div>
          ))}
        </div>
      )}

      {/* Low-confidence relations */}
      {data.low_confidence_relations.length > 0 && (
        <div className="card p-4" data-testid="quality-low-conf">
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
                  className="px-2 py-0.5 text-xs bg-green-900/40 text-green-300 rounded disabled:opacity-50"
                  data-testid={`confirm-${r.relation_id}`}
                >
                  Confirm
                </button>
                <button
                  onClick={() => handleDismiss(r.relation_id)}
                  disabled={reviewing === r.relation_id}
                  className="px-2 py-0.5 text-xs bg-red-900/40 text-red-300 rounded disabled:opacity-50"
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
        <div className="card p-4" data-testid="quality-conflicts">
          <h3 className="text-sm font-semibold mb-2">Conflicting Relations ({data.conflicting_relations.length})</h3>
          {data.conflicting_relations.slice(0, 10).map((r) => (
            <div key={r.relation_id} className="text-xs mb-2 flex justify-between items-center">
              <span data-testid={`conflict-${r.relation_id}`}>{r.conflict_reason}</span>
              <span className="flex gap-1">
                <button
                  onClick={() => handleConfirm(r.relation_id)}
                  disabled={reviewing === r.relation_id}
                  className="px-2 py-0.5 text-xs bg-green-900/40 text-green-300 rounded disabled:opacity-50"
                  data-testid={`confirm-conflict-${r.relation_id}`}
                >
                  Confirm
                </button>
                <button
                  onClick={() => handleDismiss(r.relation_id)}
                  disabled={reviewing === r.relation_id}
                  className="px-2 py-0.5 text-xs bg-red-900/40 text-red-300 rounded disabled:opacity-50"
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
        <div className="card p-4" data-testid="quality-empty">
          <p className="text-sm text-[var(--text-muted)]">No quality issues detected.</p>
        </div>
      )}
    </div>
  );
}
