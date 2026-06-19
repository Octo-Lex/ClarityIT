import { useState } from 'react';
import { api, ApiError } from '../../api/client';

interface RiskPolicyInput {
  min_approvers: number;
  requires_mfa: boolean;
  allow_self_approval: boolean;
}

interface SimResult {
  scenario_id: string;
  action_type: string;
  risk_level: string;
  allowed: boolean;
  blocked: boolean;
  requires_approval: boolean;
  requires_mfa: boolean;
  min_approvers: number;
  allow_self_approval: boolean;
  decision_explanation: string;
}

interface PolicyDiffChange {
  risk_level: string;
  field: string;
  current: any;
  draft: any;
}

interface SimResponse {
  simulation_only: boolean;
  live_policy_changed: boolean;
  results: SimResult[];
  policy_diff: {
    changed: boolean;
    changes: PolicyDiffChange[];
  };
}

const defaultPolicy: Record<string, RiskPolicyInput> = {
  low: { min_approvers: 0, requires_mfa: false, allow_self_approval: true },
  medium: { min_approvers: 1, requires_mfa: false, allow_self_approval: false },
  high: { min_approvers: 1, requires_mfa: true, allow_self_approval: false },
  critical: { min_approvers: 2, requires_mfa: true, allow_self_approval: false },
};

const defaultScenarios = [
  { scenario_id: 'low-action', action_type: 'noop.check', risk_level: 'low', description: 'Low-risk read-only action' },
  { scenario_id: 'medium-action', action_type: 'work_items.update', risk_level: 'medium', description: 'Medium-risk update' },
  { scenario_id: 'high-action', action_type: 'proxmox.shutdown', risk_level: 'high', description: 'Shutdown VM' },
  { scenario_id: 'critical-action', action_type: 'proxmox.stop', risk_level: 'critical', description: 'Force stop VM' },
];

export default function PolicySimulationPanel({ isPlatformOwner }: { isPlatformOwner: boolean }) {
  const [policy, setPolicy] = useState<Record<string, RiskPolicyInput>>(JSON.parse(JSON.stringify(defaultPolicy)));
  const [scenarios] = useState(defaultScenarios);
  const [results, setResults] = useState<SimResponse | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  if (!isPlatformOwner) {
    return (
      <div className="rounded-xl border border-border bg-surface p-4" data-testid="sim-panel-unauthorized">
        <p className="text-sm text-muted-foreground">Platform owner access required.</p>
      </div>
    );
  }

  const handleSimulate = () => {
    setLoading(true);
    setError('');
    api.simulateApprovalPolicy({
      draft_policy: { scope: 'team', ...policy },
      scenarios,
    })
      .then((data) => { setResults(data as unknown as SimResponse); setLoading(false); })
      .catch((e: unknown) => {
        setError(e instanceof ApiError ? e.message : 'Simulation failed');
        setLoading(false);
      });
  };

  const updatePolicy = (level: string, field: keyof RiskPolicyInput, value: any) => {
    setPolicy(prev => ({
      ...prev,
      [level]: { ...prev[level], [field]: value },
    }));
  };

  return (
    <div className="rounded-xl border border-border bg-surface p-4" data-testid="sim-panel">
      <h2 className="text-lg font-semibold mb-2">Approval Policy Simulation</h2>

      {/* Simulation-only warning */}
      <div className="mb-4 p-3 bg-warning/20 border border-warning/40 rounded text-sm text-warning" data-testid="sim-warning">
        ⚠ Simulation only — no changes to live policy
      </div>

      {/* Draft policy inputs */}
      <div className="mb-4 space-y-3" data-testid="sim-policy-inputs">
        <h3 className="text-sm font-semibold">Draft Policy</h3>
        <div className="grid grid-cols-4 gap-3">
          {['low', 'medium', 'high', 'critical'].map(level => (
            <div key={level} className="border border-border rounded p-2" data-testid={`sim-policy-${level}`}>
              <div className="text-xs font-semibold capitalize mb-2">{level}</div>
              <div className="space-y-1">
                <label className="text-xs block">
                  Min Approvers:
                  <input
                    type="number"
                    min={0}
                    max={5}
                    value={policy[level].min_approvers}
                    onChange={e => updatePolicy(level, 'min_approvers', parseInt(e.target.value) || 0)}
                    className="w-full bg-background border border-border rounded px-1 text-xs"
                    data-testid={`sim-${level}-approvers`}
                  />
                </label>
                <label className="text-xs flex items-center gap-1">
                  <input
                    type="checkbox"
                    checked={policy[level].requires_mfa}
                    onChange={e => updatePolicy(level, 'requires_mfa', e.target.checked)}
                    data-testid={`sim-${level}-mfa`}
                  />
                  MFA
                </label>
                <label className="text-xs flex items-center gap-1">
                  <input
                    type="checkbox"
                    checked={policy[level].allow_self_approval}
                    onChange={e => updatePolicy(level, 'allow_self_approval', e.target.checked)}
                    data-testid={`sim-${level}-self`}
                  />
                  Self-Approve
                </label>
              </div>
            </div>
          ))}
        </div>
      </div>

      {/* Simulate button */}
      <button
        onClick={handleSimulate}
        disabled={loading}
        className="px-4 py-2 bg-primary text-white rounded text-sm disabled:opacity-50"
        data-testid="sim-button"
      >
        {loading ? 'Simulating...' : 'Run Simulation'}
      </button>
      {/* Explicitly NO save/apply button */}

      {error && (
        <div className="mt-3 text-sm text-destructive" data-testid="sim-error">{error}</div>
      )}

      {/* Results */}
      {results && (
        <div className="mt-4 space-y-4">
          {/* Result table */}
          <div data-testid="sim-results">
            <h3 className="text-sm font-semibold mb-2">Simulation Results</h3>
            <div className="overflow-x-auto">
              <table className="w-full text-xs">
                <thead>
                  <tr className="text-left text-muted-foreground border-b border-border">
                    <th className="pb-1 pr-2">Scenario</th>
                    <th className="pb-1 pr-2">Action</th>
                    <th className="pb-1 pr-2">Risk</th>
                    <th className="pb-1 pr-2">Allowed</th>
                    <th className="pb-1 pr-2">Approval</th>
                    <th className="pb-1 pr-2">MFA</th>
                    <th className="pb-1 pr-2">Min Approvers</th>
                    <th className="pb-1 pr-2">Self-Approve</th>
                    <th className="pb-1">Explanation</th>
                  </tr>
                </thead>
                <tbody>
                  {results.results.map((r) => (
                    <tr key={r.scenario_id} className="border-b border-border" data-testid={`sim-result-${r.scenario_id}`}>
                      <td className="py-1 pr-2">{r.scenario_id}</td>
                      <td className="py-1 pr-2">{r.action_type}</td>
                      <td className="py-1 pr-2 capitalize">{r.risk_level}</td>
                      <td className="py-1 pr-2">
                        <span className={r.allowed ? 'text-success' : 'text-destructive'} data-testid={`sim-result-allowed-${r.scenario_id}`}>
                          {r.allowed ? 'Yes' : 'No'}
                        </span>
                      </td>
                      <td className="py-1 pr-2" data-testid={`sim-result-approval-${r.scenario_id}`}>
                        {r.requires_approval ? 'Yes' : 'No'}
                      </td>
                      <td className="py-1 pr-2" data-testid={`sim-result-mfa-${r.scenario_id}`}>
                        {r.requires_mfa ? '✓' : '—'}
                      </td>
                      <td className="py-1 pr-2" data-testid={`sim-result-approvers-${r.scenario_id}`}>{r.min_approvers}</td>
                      <td className="py-1 pr-2">{r.allow_self_approval ? 'Yes' : 'No'}</td>
                      <td className="py-1 text-muted-foreground">{r.decision_explanation}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>

          {/* Policy diff */}
          <div data-testid="sim-diff">
            <h3 className="text-sm font-semibold mb-2">Policy Diff (Draft vs Current)</h3>
            {results.policy_diff.changed ? (
              <div className="space-y-1">
                {results.policy_diff.changes.map((c, i) => (
                  <div key={i} className="text-xs flex gap-2" data-testid={`sim-diff-${i}`}>
                    <span className="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium bg-warning/20 text-warning capitalize">{c.risk_level}</span>
                    <span className="text-muted-foreground">{c.field}:</span>
                    <span className="text-destructive">{String(c.current)}</span>
                    <span className="text-muted-foreground">→</span>
                    <span className="text-success">{String(c.draft)}</span>
                  </div>
                ))}
              </div>
            ) : (
              <p className="text-xs text-muted-foreground">No changes from current policy.</p>
            )}
          </div>
        </div>
      )}
    </div>
  );
}
