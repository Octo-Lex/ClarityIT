import { useState, useEffect } from 'react';
import { api, ApiError } from '../../api/client';

interface OutcomeData {
  id?: string;
  available?: boolean;
  expected_result?: string | null;
  actual_result?: string | null;
  operator_feedback?: string | null;
  outcome_status?: string;
  follow_up_recommendation?: string | null;
  created_by?: string;
  created_at?: string;
  updated_at?: string;
}

type OutcomeSource = 'asset-action' | 'remediation';

export default function OutcomePanel({
  sourceType,
  sourceId,
  sourceStatus,
}: {
  sourceType: OutcomeSource;
  sourceId: string;
  sourceStatus: string;
}) {
  const [outcome, setOutcome] = useState<OutcomeData | null>(null);
  const [loading, setLoading] = useState(true);
  const [editing, setEditing] = useState(false);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');
  const [form, setForm] = useState({
    expected_result: '',
    actual_result: '',
    operator_feedback: '',
    outcome_status: 'successful',
    follow_up_recommendation: '',
  });

  const isTerminal =
    sourceType === 'asset-action'
      ? ['succeeded', 'failed', 'cancelled'].includes(sourceStatus)
      : ['completed', 'executed', 'succeeded', 'failed', 'cancelled'].includes(sourceStatus);

  const getOutcome = () =>
    sourceType === 'asset-action'
      ? api.getAssetActionOutcome(sourceId)
      : api.getRemediationOutcome(sourceId);

  const saveOutcome = (data: any) =>
    sourceType === 'asset-action'
      ? api.saveAssetActionOutcome(sourceId, data)
      : api.saveRemediationOutcome(sourceId, data);

  useEffect(() => {
    let active = true;
    getOutcome()
      .then((data: OutcomeData) => {
        if (active) {
          setOutcome(data);
          if (data.outcome_status) {
            setForm({
              expected_result: data.expected_result || '',
              actual_result: data.actual_result || '',
              operator_feedback: data.operator_feedback || '',
              outcome_status: data.outcome_status || 'successful',
              follow_up_recommendation: data.follow_up_recommendation || '',
            });
          }
          setLoading(false);
        }
      })
      .catch(() => { if (active) setLoading(false); });
    return () => { active = false; };
  }, [sourceId]);

  const handleSave = (e: React.FormEvent) => {
    e.preventDefault();
    setSaving(true);
    setError('');
    saveOutcome(form)
      .then((data: OutcomeData) => {
        setOutcome(data);
        setEditing(false);
        setSaving(false);
      })
      .catch((e: unknown) => {
        setError(e instanceof ApiError ? e.message : 'Failed to save outcome');
        setSaving(false);
      });
  };

  if (loading) return null;

  // Not terminal — no form
  if (!isTerminal) {
    return (
      <div className="mt-3 p-3 bg-[var(--card)] border border-[var(--border)] rounded-lg" data-testid="outcome-panel">
        <h4 className="text-sm font-semibold mb-1">Post-Action Outcome</h4>
        <p className="text-xs text-[var(--text-muted)]" data-testid="outcome-not-terminal">
          Outcome recording will be available once this action reaches a terminal status.
        </p>
      </div>
    );
  }

  const statusColors: Record<string, string> = {
    successful: 'text-green-400',
    partially_successful: 'text-yellow-400',
    failed: 'text-red-400',
    inconclusive: 'text-gray-400',
  };

  return (
    <div className="mt-3 p-3 bg-[var(--card)] border border-[var(--border)] rounded-lg" data-testid="outcome-panel">
      <h4 className="text-sm font-semibold mb-2">Post-Action Outcome</h4>

      {/* Warning */}
      <div className="mb-3 p-2 bg-yellow-900/20 border border-yellow-700 rounded text-xs text-yellow-300" data-testid="outcome-warning">
        ⚠ Recording an outcome will not trigger any automatic retry or follow-up action.
      </div>

      {error && (
        <div className="mb-2 text-xs text-red-400" data-testid="outcome-error">{error}</div>
      )}

      {/* Display mode */}
      {outcome?.available && !editing ? (
        <div className="space-y-2 text-xs" data-testid="outcome-display">
          <div data-testid="outcome-status-display">
            <span className="text-[var(--text-muted)]">Status: </span>
            <span className={`font-semibold ${statusColors[outcome.outcome_status] || ''}`}>
              {outcome.outcome_status}
            </span>
          </div>
          {outcome.expected_result && (
            <div><span className="text-[var(--text-muted)]">Expected: </span>{outcome.expected_result}</div>
          )}
          {outcome.actual_result && (
            <div><span className="text-[var(--text-muted)]">Actual: </span>{outcome.actual_result}</div>
          )}
          {outcome.operator_feedback && (
            <div><span className="text-[var(--text-muted)]">Feedback: </span>{outcome.operator_feedback}</div>
          )}
          {outcome.follow_up_recommendation && (
            <div><span className="text-[var(--text-muted)]">Follow-up: </span>{outcome.follow_up_recommendation}</div>
          )}
          {outcome.created_at && (
            <div className="text-[var(--text-muted)]">
              Recorded: {new Date(outcome.created_at).toLocaleDateString()}
            </div>
          )}
          <button
            onClick={() => setEditing(true)}
            className="mt-1 px-2 py-1 text-xs border border-[var(--border)] rounded hover:bg-[var(--border)]"
            data-testid="outcome-edit-btn"
          >
            Edit Outcome
          </button>
        </div>
      ) : (
        /* Form mode — shown when editing or when no outcome exists */
        (editing || !outcome?.available) && (
          <form onSubmit={handleSave} className="space-y-2" data-testid="outcome-form">
            <div>
              <label className="text-xs block text-[var(--text-muted)]">Outcome Status *</label>
              <select
                value={form.outcome_status}
                onChange={e => setForm({ ...form, outcome_status: e.target.value })}
                className="w-full bg-[var(--bg)] border border-[var(--border)] rounded px-2 py-1 text-xs"
                data-testid="outcome-form-status"
              >
                <option value="successful">Successful</option>
                <option value="partially_successful">Partially Successful</option>
                <option value="failed">Failed</option>
                <option value="inconclusive">Inconclusive</option>
              </select>
            </div>
            <div>
              <label className="text-xs block text-[var(--text-muted)]">Expected Result</label>
              <textarea
                value={form.expected_result}
                onChange={e => setForm({ ...form, expected_result: e.target.value })}
                maxLength={2000}
                className="w-full bg-[var(--bg)] border border-[var(--border)] rounded px-2 py-1 text-xs"
                rows={2}
                data-testid="outcome-form-expected"
              />
            </div>
            <div>
              <label className="text-xs block text-[var(--text-muted)]">Actual Result</label>
              <textarea
                value={form.actual_result}
                onChange={e => setForm({ ...form, actual_result: e.target.value })}
                maxLength={4000}
                className="w-full bg-[var(--bg)] border border-[var(--border)] rounded px-2 py-1 text-xs"
                rows={3}
                data-testid="outcome-form-actual"
              />
            </div>
            <div>
              <label className="text-xs block text-[var(--text-muted)]">Operator Feedback</label>
              <textarea
                value={form.operator_feedback}
                onChange={e => setForm({ ...form, operator_feedback: e.target.value })}
                maxLength={4000}
                className="w-full bg-[var(--bg)] border border-[var(--border)] rounded px-2 py-1 text-xs"
                rows={3}
                data-testid="outcome-form-feedback"
              />
            </div>
            <div>
              <label className="text-xs block text-[var(--text-muted)]">Follow-up Recommendation</label>
              <textarea
                value={form.follow_up_recommendation}
                onChange={e => setForm({ ...form, follow_up_recommendation: e.target.value })}
                maxLength={2000}
                className="w-full bg-[var(--bg)] border border-[var(--border)] rounded px-2 py-1 text-xs"
                rows={2}
                data-testid="outcome-form-followup"
              />
            </div>
            <div className="flex gap-2">
              <button
                type="submit"
                disabled={saving}
                className="px-3 py-1 bg-[var(--primary)] text-white rounded text-xs disabled:opacity-50"
                data-testid="outcome-form-submit"
              >
                {saving ? 'Saving...' : 'Save Outcome'}
              </button>
              {outcome?.available && (
                <button
                  type="button"
                  onClick={() => setEditing(false)}
                  className="px-3 py-1 text-xs border border-[var(--border)] rounded"
                >
                  Cancel
                </button>
              )}
            </div>
            {/* Explicitly NO retry or execute-follow-up buttons */}
          </form>
        )
      )}

      {/* Empty state when outcome not available but not editing */}
      {!outcome?.available && editing === false && (
        <div className="text-xs text-[var(--text-muted)]" data-testid="outcome-empty">
          No outcome recorded yet.
          <button
            onClick={() => setEditing(true)}
            className="ml-2 underline"
            data-testid="outcome-record-btn"
          >
            Record Outcome
          </button>
        </div>
      )}
    </div>
  );
}
