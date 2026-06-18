import { useState, useEffect } from 'react';
import { api, ApiError } from '../../api/client';

interface ScenarioResult {
  scenario_id: string;
  scenario_name: string;
  passed: boolean;
  score: number;
  correctness_score: number;
  safety_score: number;
  explainability_score: number;
  quality_score: number;
  failure_reasons: string[];
}

interface EvalData {
  run_id: string | null;
  run_status: string;
  scenario_count: number;
  passed_count: number;
  failed_count: number;
  average_score: number;
  safety_score: number;
  explainability_score: number;
  correctness_score: number;
  quality_score: number;
  evaluation_only: boolean;
  created_at?: string;
  completed_at?: string;
  scenarios: ScenarioResult[];
}

export default function AgentEvaluationPanel() {
  const [data, setData] = useState<EvalData | null>(null);
  const [loading, setLoading] = useState(true);
  const [running, setRunning] = useState(false);
  const [error, setError] = useState('');

  const fetchResults = () => {
    api.getEvaluationResults()
      .then((d) => { setData(d as unknown as EvalData); setLoading(false); })
      .catch((e: unknown) => {
        if (e instanceof ApiError) { setError(e.message); }
        setLoading(false);
      });
  };

  useEffect(() => { fetchResults(); }, []);

  const handleRun = () => {
    setRunning(true);
    setError('');
    api.runEvaluation()
      .then(() => {
        fetchResults();
        setRunning(false);
      })
      .catch((e: unknown) => {
        if (e instanceof ApiError) { setError(e.message); }
        setRunning(false);
      });
  };

  if (loading) return null;

  const hasRun = data?.run_id;

  return (
    <div className="space-y-3" data-testid="eval-panel">
      <h2 className="text-lg font-semibold">Agent Recommendation Evaluation</h2>

      {/* Evaluation-only warning */}
      <div className="p-3 bg-yellow-900/20 border border-yellow-700 rounded text-sm text-yellow-300" data-testid="eval-warning">
        ⚠ Evaluation uses controlled scenarios only and does not execute tools or change live operations.
      </div>

      {error && (
        <div className="text-sm text-red-400" data-testid="eval-error">{error}</div>
      )}

      {/* Run button */}
      <button
        onClick={handleRun}
        disabled={running}
        className="px-4 py-2 bg-blue-600 text-white rounded disabled:opacity-50 text-sm"
        data-testid="eval-run-btn"
      >
        {running ? 'Running...' : 'Run Evaluation'}
      </button>

      {/* Empty state */}
      {!hasRun && !error && (
        <div className="card p-4" data-testid="eval-empty">
          <p className="text-sm text-[var(--text-muted)]">No evaluation runs yet. Click "Run Evaluation" to start.</p>
        </div>
      )}

      {/* Run summary */}
      {hasRun && data && (
        <div className="card p-4" data-testid="eval-summary">
          <div className="grid grid-cols-4 gap-3 text-xs mb-3">
            <div><span className="text-[var(--text-muted)]">Status: </span>{data.run_status}</div>
            <div><span className="text-[var(--text-muted)]">Scenarios: </span>{data.scenario_count}</div>
            <div><span className="text-[var(--text-muted)]">Passed: </span><span className="text-green-400">{data.passed_count}</span></div>
            <div><span className="text-[var(--text-muted)]">Failed: </span><span className="text-red-400">{data.failed_count}</span></div>
          </div>
          <div className="grid grid-cols-5 gap-3 text-xs">
            <div><span className="text-[var(--text-muted)]">Avg: </span>{(data.average_score * 100).toFixed(0)}%</div>
            <div><span className="text-[var(--text-muted)]">Correctness: </span>{(data.correctness_score * 100).toFixed(0)}%</div>
            <div><span className="text-[var(--text-muted)]">Safety: </span>{(data.safety_score * 100).toFixed(0)}%</div>
            <div><span className="text-[var(--text-muted)]">Explainability: </span>{(data.explainability_score * 100).toFixed(0)}%</div>
            <div><span className="text-[var(--text-muted)]">Quality: </span>{(data.quality_score * 100).toFixed(0)}%</div>
          </div>
          {data.created_at && (
            <div className="text-xs text-[var(--text-muted)] mt-2" data-testid="eval-timestamp">
              Created: {data.created_at}
            </div>
          )}
        </div>
      )}

      {/* Scenario results table */}
      {hasRun && data && data.scenarios.length > 0 && (
        <div className="card p-4" data-testid="eval-scenarios">
          <table className="w-full text-xs">
            <thead>
              <tr className="text-[var(--text-muted)] border-b border-[var(--border)]">
                <th className="text-left py-1">Scenario</th>
                <th className="text-center py-1">Result</th>
                <th className="text-center py-1">Score</th>
                <th className="text-center py-1">Correctness</th>
                <th className="text-center py-1">Safety</th>
                <th className="text-center py-1">Explainability</th>
                <th className="text-center py-1">Quality</th>
                <th className="text-left py-1">Failure Reasons</th>
              </tr>
            </thead>
            <tbody>
              {data.scenarios.map((s) => (
                <tr key={s.scenario_id} className="border-b border-[var(--border)]" data-testid={`eval-scenario-${s.scenario_id}`}>
                  <td className="py-1">{s.scenario_name}</td>
                  <td className="text-center py-1">
                    {s.passed
                      ? <span className="text-green-400" data-testid={`eval-pass-${s.scenario_id}`}>✓ PASS</span>
                      : <span className="text-red-400" data-testid={`eval-fail-${s.scenario_id}`}>✗ FAIL</span>}
                  </td>
                  <td className="text-center py-1" data-testid={`eval-score-${s.scenario_id}`}>
                    {(s.score * 100).toFixed(0)}%
                  </td>
                  <td className="text-center py-1">{(s.correctness_score * 100).toFixed(0)}%</td>
                  <td className="text-center py-1">{(s.safety_score * 100).toFixed(0)}%</td>
                  <td className="text-center py-1">{(s.explainability_score * 100).toFixed(0)}%</td>
                  <td className="text-center py-1">{(s.quality_score * 100).toFixed(0)}%</td>
                  <td className="py-1">
                    {s.failure_reasons.length > 0 ? (
                      <ul className="text-red-400" data-testid={`eval-reasons-${s.scenario_id}`}>
                        {s.failure_reasons.map((r, i) => <li key={i}>• {r}</li>)}
                      </ul>
                    ) : (
                      <span className="text-[var(--text-muted)]">—</span>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
