import { useState, useEffect, useCallback } from 'react';
import { api, ApiError } from '../../api/client';

export default function SecurityPage() {
  const [status, setStatus] = useState<{ enabled: boolean; verified_factors: number; pending_factors: number } | null>(null);
  const [factors, setFactors] = useState<{ id: string; type: string; verified: boolean; created_at: string }[]>([]);
  const [loading, setLoading] = useState(true);
  const [enrolling, setEnrolling] = useState(false);
  const [enrollData, setEnrollData] = useState<{ factor_id: string; secret: string; otpauth_uri: string } | null>(null);
  const [verifyCode, setVerifyCode] = useState('');
  const [recoveryCodes, setRecoveryCodes] = useState<string[] | null>(null);
  const [error, setError] = useState('');

  const load = useCallback(async () => {
    try {
      const [s, f] = await Promise.all([api.mfaStatus(), api.mfaListFactors()]);
      setStatus(s); setFactors(f || []);
    } catch (e) { /* ignore */ }
    setLoading(false);
  }, []);

  useEffect(() => { load(); }, [load]);

  const handleEnroll = async () => {
    setEnrolling(true); setError('');
    try {
      const data = await api.mfaEnroll();
      setEnrollData(data);
    } catch (e) {
      setError(e instanceof ApiError ? e.message : 'Enrollment failed');
    }
    setEnrolling(false);
  };

  const handleVerifyEnrollment = async () => {
    if (!enrollData || !verifyCode) return;
    setError('');
    try {
      await api.mfaVerifyEnrollment(enrollData.factor_id, verifyCode);
      setEnrollData(null); setVerifyCode('');
      // Generate recovery codes
      const rc = await api.mfaRegenerateRecovery();
      setRecoveryCodes(rc.recovery_codes);
      await load();
    } catch (e) {
      setError(e instanceof ApiError ? e.message : 'Verification failed');
    }
  };

  const handleDisable = async (factorId: string) => {
    setError('');
    try {
      await api.mfaDisableFactor(factorId);
      await load();
    } catch (e) {
      setError(e instanceof ApiError ? e.message : 'Disable failed (recent MFA may be required)');
    }
  };

  if (loading) return <div className="p-6 text-[var(--text-muted)]">Loading...</div>;

  return (
    <div className="max-w-2xl space-y-6">
      <h1 className="text-2xl font-bold">Security</h1>
      {error && <div className="p-3 bg-red-900/30 border border-red-700 rounded text-sm text-red-300">{error}</div>}

      {/* MFA Status */}
      <section className="p-4 bg-[var(--card)] border border-[var(--border)] rounded-lg">
        <h2 className="text-lg font-semibold mb-2">Multi-Factor Authentication</h2>
        {status?.enabled ? (
          <p className="text-sm text-[var(--success)]">✓ MFA is enabled ({status.verified_factors} active factor(s))</p>
        ) : (
          <p className="text-sm text-[var(--text-muted)]">MFA is not enabled. Enroll a TOTP factor to secure your account.</p>
        )}

        {/* Active factors */}
        {factors.length > 0 && (
          <div className="mt-4 space-y-2">
            {factors.map(f => (
              <div key={f.id} className="flex items-center justify-between p-2 bg-[var(--bg)] rounded text-sm">
                <div>
                  <span className="font-medium">{f.type}</span>
                  {f.verified ? (
                    <span className="ml-2 text-[var(--success)]">✓ Verified</span>
                  ) : (
                    <span className="ml-2 text-yellow-500">⚠ Pending</span>
                  )}
                </div>
                <button
                  onClick={() => handleDisable(f.id)}
                  data-testid={`disable-factor-${f.id}`}
                  className="text-xs px-2 py-1 bg-red-900/30 border border-red-700 rounded hover:bg-red-900/50"
                >
                  Disable
                </button>
              </div>
            ))}
          </div>
        )}

        {/* Enrollment flow */}
        {!enrollData && !recoveryCodes && (
          <button
            onClick={handleEnroll}
            disabled={enrolling}
            data-testid="enroll-mfa-btn"
            className="mt-4 px-4 py-2 bg-[var(--primary)] text-white rounded text-sm hover:opacity-90"
          >
            {enrolling ? 'Enrolling...' : 'Enroll TOTP'}
          </button>
        )}

        {enrollData && (
          <div className="mt-4 space-y-3" data-testid="mfa-enrollment">
            <p className="text-sm font-medium">Scan this secret in your authenticator app:</p>
            <div className="p-3 bg-[var(--bg)] rounded font-mono text-sm break-all" data-testid="totp-secret">
              {enrollData.secret}
            </div>
            <p className="text-xs text-[var(--text-muted)]">Or use this URI: {enrollData.otpauth_uri}</p>
            <div className="flex gap-2">
              <input
                type="text"
                placeholder="6-digit code"
                value={verifyCode}
                onChange={e => setVerifyCode(e.target.value)}
                data-testid="verify-code-input"
                className="px-3 py-1.5 bg-[var(--bg)] border border-[var(--border)] rounded text-sm"
                maxLength={6}
              />
              <button
                onClick={handleVerifyEnrollment}
                data-testid="verify-enrollment-btn"
                className="px-4 py-1.5 bg-[var(--success)] text-white rounded text-sm"
              >
                Verify
              </button>
            </div>
          </div>
        )}
      </section>

      {/* Recovery Codes — shown once */}
      {recoveryCodes && (
        <section className="p-4 bg-yellow-900/20 border border-yellow-700 rounded-lg" data-testid="recovery-codes">
          <h2 className="text-lg font-semibold text-yellow-300">⚠ Save Your Recovery Codes</h2>
          <p className="text-sm mt-1 text-yellow-200">
            These codes will only be shown once. Store them in a secure password manager.
          </p>
          <div className="mt-3 grid grid-cols-2 gap-2">
            {recoveryCodes.map((code, i) => (
              <div key={i} className="p-2 bg-[var(--bg)] rounded font-mono text-sm">{code}</div>
            ))}
          </div>
          <button
            onClick={() => setRecoveryCodes(null)}
            data-testid="ack-recovery-codes"
            className="mt-4 px-4 py-2 bg-[var(--primary)] text-white rounded text-sm"
          >
            I've saved my codes
          </button>
        </section>
      )}
    </div>
  );
}
