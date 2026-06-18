import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { RefreshCw, AlertTriangle, TrendingUp, Activity, Volume2 } from 'lucide-react';
import { api } from '@/api/client';
import { keys } from '@/api/keys';
import { useAuth } from '@/auth/context';
import { Card } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { StatusBadge } from '@/components/ui/status-badge';

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

const patternTypeIcon: Record<string, typeof RefreshCw> = {
  recurring_asset: RefreshCw,
  recurring_symptom: TrendingUp,
  cluster: AlertTriangle,
  noisy_asset: Volume2,
};

function confidenceTone(c: number): 'success' | 'warning' | 'danger' {
  if (c >= 0.7) return 'success';
  if (c >= 0.4) return 'warning';
  return 'danger';
}

export default function PatternCards() {
  const { activeTeamId } = useAuth();
  const [expanded, setExpanded] = useState(false);

  const { data: patterns, isPending, error, refetch } = useQuery({
    queryKey: keys.incidents.patterns(activeTeamId ?? '', { window_days: 7, min_occurrences: 2 }),
    queryFn: ({ signal }) => api.getIncidentPatterns({ window_days: 7, min_occurrences: 2 }, signal),
    enabled: !!activeTeamId,
  });

  const list = (patterns?.patterns ?? []) as unknown as Pattern[];

  if (isPending) return null;

  if (error) {
    return (
      <Card className="p-4" data-testid="pattern-error-container">
        <div className="flex items-center justify-between">
          <div className="text-sm text-destructive" data-testid="pattern-error">
            {error instanceof Error ? error.message : 'Failed to load patterns'}
          </div>
          <Button size="sm" variant="secondary" onClick={() => refetch()}>Retry</Button>
        </div>
      </Card>
    );
  }

  if (!list.length) {
    return (
      <Card className="p-4" data-testid="pattern-empty">
        <p className="flex items-center gap-2 text-sm text-muted-foreground">
          <Activity className="size-4" /> No incident patterns detected in the last 7 days.
        </p>
      </Card>
    );
  }

  return (
    <div className="space-y-2" data-testid="pattern-cards">
      <div className="flex items-center justify-between">
        <h2 className="font-heading text-lg font-semibold">Incident Patterns</h2>
        <button
          onClick={() => setExpanded(!expanded)}
          className="text-xs text-muted-foreground hover:text-foreground"
          data-testid="pattern-toggle"
        >
          {expanded ? 'Collapse' : 'Show all'}
        </button>
      </div>

      {list.slice(0, expanded ? list.length : 3).map((p) => {
        const Icon = patternTypeIcon[p.pattern_type] ?? AlertTriangle;
        return (
          <Card key={p.pattern_id} className="p-4" data-testid={`pattern-card-${p.pattern_type}`}>
            <div className="mb-2 flex items-start justify-between">
              <div className="flex items-center gap-2">
                <Icon className="size-4 text-muted-foreground" />
                <span className="text-sm font-semibold" data-testid={`pattern-type-${p.pattern_type}`}>
                  {patternTypeLabel[p.pattern_type] || p.pattern_type}
                </span>
              </div>
              <StatusBadge tone={confidenceTone(p.confidence)} data-testid={`pattern-confidence-${p.pattern_type}`}>
                {Math.round(p.confidence * 100)}%
              </StatusBadge>
            </div>

            <p className="mb-2 text-sm text-muted-foreground" data-testid={`pattern-description-${p.pattern_type}`}>
              {p.pattern_description}
            </p>

            <div className="flex flex-wrap gap-3 text-xs">
              <span className="text-muted-foreground" data-testid={`pattern-count-${p.pattern_type}`}>
                {p.occurrence_count} incident{p.occurrence_count !== 1 ? 's' : ''}
              </span>

              {p.affected_assets && p.affected_assets.length > 0 && (
                <span data-testid={`pattern-assets-${p.pattern_type}`}>
                  Assets: {p.affected_assets.map(a => a.name || a.asset_id).join(', ')}
                </span>
              )}

              {p.severity_mix && (p.severity_mix.critical > 0 || p.severity_mix.high > 0) && (
                <span className="text-destructive" data-testid={`pattern-severity-${p.pattern_type}`}>
                  {p.severity_mix.critical > 0 && `${p.severity_mix.critical} critical`}
                  {p.severity_mix.critical > 0 && p.severity_mix.high > 0 && ', '}
                  {p.severity_mix.high > 0 && `${p.severity_mix.high} high`}
                </span>
              )}
            </div>

            <div className="mt-2 flex items-center gap-2">
              <StatusBadge tone="warning" data-testid={`pattern-advisory-${p.pattern_type}`}>
                Pattern detected — review recommended
              </StatusBadge>
            </div>

            {expanded && p.incident_ids.length > 0 && (
              <div className="mt-2 text-xs text-muted-foreground" data-testid={`pattern-incidents-${p.pattern_type}`}>
                Incidents: {p.incident_ids.slice(0, 5).map(id => id.substring(0, 8)).join(', ')}
                {p.incident_ids.length > 5 && ` +${p.incident_ids.length - 5} more`}
              </div>
            )}
          </Card>
        );
      })}

      {!expanded && list.length > 3 && (
        <p className="text-center text-xs text-muted-foreground">
          +{list.length - 3} more pattern{list.length - 3 !== 1 ? 's' : ''}
        </p>
      )}
    </div>
  );
}
