import { useState, useEffect } from 'react';
import { api, ApiError } from '../../api/client';

interface EvidenceItem {
  type?: string;
  description?: string;
  source?: string;
  [key: string]: any;
}

interface EvidencePack {
  available: boolean;
  recommendation_summary?: string;
  supporting_evidence?: EvidenceItem[];
  conflicting_evidence?: EvidenceItem[];
  confidence_score?: number;
  confidence_level?: string;
  risk_notes?: string;
  missing_info?: EvidenceItem[];
  is_stale?: boolean;
  message?: string;
}

export default function EvidencePanel({ recommendationId }: { recommendationId: string }) {
  const [evidence, setEvidence] = useState<EvidencePack | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  useEffect(() => {
    let active = true;
    api.getEvidence(recommendationId)
      .then((data) => { if (active) { setEvidence(data as unknown as EvidencePack); setLoading(false); } })
      .catch((e: unknown) => {
        if (active) {
          setError(e instanceof ApiError ? e.message : 'Failed to load evidence');
          setLoading(false);
        }
      });
    return () => { active = false; };
  }, [recommendationId]);

  if (loading) return null;

  if (error) {
    return (
      <div className="mt-3 p-3 bg-surface border border-border rounded-lg" data-testid="evidence-panel">
        <div className="text-xs text-destructive" data-testid="evidence-error">{error}</div>
      </div>
    );
  }

  if (!evidence || !evidence.available) {
    return (
      <div className="mt-3 p-3 bg-surface border border-border rounded-lg" data-testid="evidence-panel">
        <h4 className="text-sm font-semibold mb-1">Recommendation Evidence</h4>
        <div className="text-xs text-muted-foreground" data-testid="evidence-unavailable">
          {evidence?.message || 'Evidence unavailable for this recommendation'}
        </div>
      </div>
    );
  }

  const confidenceColor = evidence.confidence_level === 'high'
    ? 'text-success'
    : evidence.confidence_level === 'medium'
    ? 'text-warning'
    : 'text-warning';

  return (
    <div className="mt-3 p-3 bg-surface border border-border rounded-lg" data-testid="evidence-panel">
      <h4 className="text-sm font-semibold mb-2">Recommendation Evidence</h4>

      {evidence.is_stale && (
        <div className="mb-2 p-2 bg-warning/20 border border-warning/40 rounded text-xs text-warning" data-testid="evidence-stale-warning">
          ⚠ Evidence may be stale — refresh recommended
        </div>
      )}

      {/* Summary */}
      <div className="mb-2" data-testid="evidence-summary">
        <div className="text-xs text-muted-foreground">Summary</div>
        <div className="text-sm">{evidence.recommendation_summary}</div>
      </div>

      {/* Confidence */}
      <div className="mb-2 flex gap-4">
        <div>
          <span className="text-xs text-muted-foreground">Confidence: </span>
          <span className={`text-sm font-semibold ${confidenceColor}`} data-testid="evidence-confidence">
            {evidence.confidence_level} ({((evidence.confidence_score || 0) * 100).toFixed(0)}%)
          </span>
        </div>
      </div>

      {/* Supporting Evidence */}
      {evidence.supporting_evidence && evidence.supporting_evidence.length > 0 && (
        <div className="mb-2" data-testid="evidence-supporting">
          <div className="text-xs font-medium text-success mb-1">✓ Supporting Evidence</div>
          {evidence.supporting_evidence.map((item, i) => (
            <div key={i} className="text-xs ml-3 mb-1">
              {item.description && <span>{item.description}</span>}
              {item.source && <span className="text-muted-foreground ml-1">({item.source})</span>}
            </div>
          ))}
        </div>
      )}

      {/* Conflicting Evidence */}
      {evidence.conflicting_evidence && evidence.conflicting_evidence.length > 0 && (
        <div className="mb-2" data-testid="evidence-conflicting">
          <div className="text-xs font-medium text-warning mb-1">⚠ Conflicting Evidence</div>
          {evidence.conflicting_evidence.map((item, i) => (
            <div key={i} className="text-xs ml-3 mb-1">
              {item.description && <span>{item.description}</span>}
              {item.source && <span className="text-muted-foreground ml-1">({item.source})</span>}
            </div>
          ))}
        </div>
      )}

      {/* Risk Notes */}
      {evidence.risk_notes && (
        <div className="mb-2" data-testid="evidence-risk-notes">
          <div className="text-xs text-muted-foreground">Risk Notes</div>
          <div className="text-xs text-warning">{evidence.risk_notes}</div>
        </div>
      )}

      {/* Missing Information */}
      {evidence.missing_info && evidence.missing_info.length > 0 && (
        <div className="mb-2" data-testid="evidence-missing">
          <div className="text-xs font-medium text-muted-foreground mb-1">? Missing Information</div>
          {evidence.missing_info.map((item, i) => (
            <div key={i} className="text-xs ml-3 mb-1 text-muted-foreground">
              {item.description || JSON.stringify(item)}
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
