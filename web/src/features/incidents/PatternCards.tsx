import { useState, useEffect } from 'react';
import { api, ApiError } from '../../api/client';

interface AffectedAsset {
  asset_id: string;
  name?: string;
  provider?: string;
}

interface Pattern {
  pattern_id: string;
  pattern_type: string;
  pattern_description: string;
  confidence: number;
  incident_ids: string[];
  asset_ids?: string[];
  affected_assets?: AffectedAsset[];
  severity_mix?: Record<string, number>;
  first_seen: string;
  last_seen: string;
  occurrence_count: number;
  advisory_only: boolean;
}

const patternTypeLabel: Record<string, string> = {
  recurring_asset: 'Recurring Asset',
  recurring_symptom: 'Recurring Symptom',
  cluster: 'Incident Cluster',
  noisy_asset: 'Noisy Asset',
};

const patternTypeIcon: Record<string, string> = {
  recurring_asset: '🔄',
  recurring_symptom: '🔁',
  cluster: '💥',
  noisy_asset: '📢',
};

export default function PatternCards() {
  const [patterns, setPatterns] = useState<Pattern[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [expanded, setExpanded] = useState(false);

  useEffect(() => {
    let active = true;
    api.getIncidentPatterns({ window_days: 7, min_occurrences: 2 })
      .then((data) => { if (active) { setPatterns((data.patterns || []) as unknown as Pattern[]); setLoading(false); } })
      .catch((e: unknown) => {
        if (active) {
          setError(e instanceof ApiError ? e.message : 'Failed to load patterns');
          setLoading(false);
        }
      });
    return () => { active = false; };
  }, []);

  if (loading) return null;

  if (error) {
    return (
      <div className="card" data-testid="pattern-error-container">
        <div className="text-sm text-red-400" data-testid="pattern-error">{error}</div>
      </div>
    );
  }

  if (!patterns.length) {
    return (
      <div className="card" data-testid="pattern-empty">
        <p className="text-sm text-[var(--text-muted)]">No incident patterns detected in the last 7 days.</p>
      </div>
    );
  }

  const confidenceColor = (c: number) => {
    if (c >= 0.7) return 'text-green-400';
    if (c >= 0.4) return 'text-yellow-400';
    return 'text-orange-400';
  };

  return (
    <div className="space-y-2" data-testid="pattern-cards">
      <div className="flex items-center justify-between">
        <h2 className="text-lg font-semibold">Incident Patterns</h2>
        <button
          onClick={() => setExpanded(!expanded)}
          className="text-xs text-[var(--text-muted)] hover:text-[var(--text)]"
          data-testid="pattern-toggle"
        >
          {expanded ? 'Collapse' : 'Show all'}
        </button>
      </div>

      {patterns.slice(0, expanded ? patterns.length : 3).map((p) => (
        <div
          key={p.pattern_id}
          className="card p-3"
          data-testid={`pattern-card-${p.pattern_type}`}
        >
          <div className="flex items-start justify-between mb-2">
            <div className="flex items-center gap-2">
              <span className="text-lg">{patternTypeIcon[p.pattern_type] || '⚠'}</span>
              <span className="text-sm font-semibold" data-testid={`pattern-type-${p.pattern_type}`}>
                {patternTypeLabel[p.pattern_type] || p.pattern_type}
              </span>
            </div>
            <span className={`text-sm font-semibold ${confidenceColor(p.confidence)}`} data-testid={`pattern-confidence-${p.pattern_type}`}>
              {Math.round(p.confidence * 100)}%
            </span>
          </div>

          <p className="text-sm text-[var(--text-muted)] mb-2" data-testid={`pattern-description-${p.pattern_type}`}>
            {p.pattern_description}
          </p>

          <div className="flex flex-wrap gap-3 text-xs">
            <span className="text-[var(--text-muted)]" data-testid={`pattern-count-${p.pattern_type}`}>
              {p.occurrence_count} incident{p.occurrence_count !== 1 ? 's' : ''}
            </span>

            {p.affected_assets && p.affected_assets.length > 0 && (
              <span data-testid={`pattern-assets-${p.pattern_type}`}>
                Assets: {p.affected_assets.map(a => a.name || a.asset_id).join(', ')}
              </span>
            )}

            {p.severity_mix && (p.severity_mix.critical > 0 || p.severity_mix.high > 0) && (
              <span className="text-red-400" data-testid={`pattern-severity-${p.pattern_type}`}>
                {p.severity_mix.critical > 0 && `${p.severity_mix.critical} critical`}
                {p.severity_mix.critical > 0 && p.severity_mix.high > 0 && ', '}
                {p.severity_mix.high > 0 && `${p.severity_mix.high} high`}
              </span>
            )}
          </div>

          <div className="mt-2 flex items-center gap-2">
            <span className="badge badge-yellow text-xs" data-testid={`pattern-advisory-${p.pattern_type}`}>
              Pattern detected — review recommended
            </span>
          </div>

          {expanded && p.incident_ids.length > 0 && (
            <div className="mt-2 text-xs text-[var(--text-muted)]" data-testid={`pattern-incidents-${p.pattern_type}`}>
              Incidents: {p.incident_ids.slice(0, 5).map(id => id.substring(0, 8)).join(', ')}
              {p.incident_ids.length > 5 && ` +${p.incident_ids.length - 5} more`}
            </div>
          )}
        </div>
      ))}

      {!expanded && patterns.length > 3 && (
        <p className="text-xs text-[var(--text-muted)] text-center">
          +{patterns.length - 3} more pattern{patterns.length - 3 !== 1 ? 's' : ''}
        </p>
      )}
    </div>
  );
}
