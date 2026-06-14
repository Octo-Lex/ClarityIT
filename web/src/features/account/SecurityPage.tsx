import { useState, useEffect, useCallback } from 'react';
import { api, ApiError } from '../../api/client';

// Check if WebAuthn is available in the browser
function isWebAuthnAvailable(): boolean {
  return typeof window !== 'undefined' &&
    typeof window.PublicKeyCredential !== 'undefined' &&
    typeof navigator.credentials !== 'undefined';
}

// Base64URL decode to ArrayBuffer
function base64urlToBuffer(base64url: string): ArrayBuffer {
  const padding = '='.repeat((4 - base64url.length % 4) % 4);
  const base64 = (base64url + padding).replace(/-/g, '+').replace(/_/g, '/');
  const rawData = atob(base64);
  const buffer = new Uint8Array(rawData.length);
  for (let i = 0; i < rawData.length; i++) buffer[i] = rawData.charCodeAt(i);
  return buffer.buffer;
}

// ArrayBuffer to Base64URL string
function bufferToBase64url(buffer: ArrayBuffer): string {
  const bytes = new Uint8Array(buffer);
  let binary = '';
  for (let i = 0; i < bytes.length; i++) binary += String.fromCharCode(bytes[i]);
  return btoa(binary).replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/, '');
}

export default function SecurityPage() {
  const [status, setStatus] = useState<{ enabled: boolean; verified_factors: number; pending_factors: number } | null>(null);
  const [factors, setFactors] = useState<{ id: string; type: string; verified: boolean; created_at: string }[]>([]);
  const [loading, setLoading] = useState(true);
  const [enrolling, setEnrolling] = useState(false);
  const [enrollData, setEnrollData] = useState<{ factor_id: string; secret: string; otpauth_uri: string } | null>(null);
  const [verifyCode, setVerifyCode] = useState('');
  const [recoveryCodes, setRecoveryCodes] = useState<string[] | null>(null);
  const [error, setError] = useState('');

  // WebAuthn state
  const [waCredentials, setWaCredentials] = useState<{ id: string; label: string; status: string; created_at: string; last_used_at?: string }[]>([]);
  const [waLabel, setWaLabel] = useState('');
  const [waBusy, setWaBusy] = useState(false);
  const [waMessage, setWaMessage] = useState('');

  const load = useCallback(async () => {
    try {
      const [s, f] = await Promise.all([api.mfaStatus(), api.mfaListFactors()]);
      setStatus(s); setFactors(f || []);
      // Load WebAuthn credentials (ignore errors if not enabled)
      try {
        const waCreds = await api.webauthnListCredentials();
        setWaCredentials(waCreds || []);
      } catch { /* WebAuthn may not be enabled */ }
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

  // ─── WebAuthn handlers ───
  const handleWaRegister = async () => {
    setWaBusy(true); setWaMessage(''); setError('');
    try {
      if (!isWebAuthnAvailable()) {
        setError('WebAuthn is not supported in this browser or context. HTTPS or localhost is required.');
        setWaBusy(false);
        return;
      }
      const label = waLabel || 'Security Key';
      const startResp = await api.webauthnRegisterStart(label);
      const opts = startResp.options;

      const publicKey: PublicKeyCredentialCreationOptions = {
        challenge: base64urlToBuffer(opts.challenge),
        rp: opts.rp,
        user: {
          ...opts.user,
          id: base64urlToBuffer(opts.user.id as unknown as string),
        },
        pubKeyCredParams: opts.pubKeyCredParams,
        timeout: opts.timeout,
        excludeCredentials: opts.excludeCredentials || [],
        authenticatorSelection: opts.authenticatorSelection,
        attestation: opts.attestation || 'none',
      };

      const credential = await navigator.credentials.create({ publicKey }) as PublicKeyCredential;
      if (!credential) throw new Error('Credential creation failed');

      const response = credential.response as AuthenticatorAttestationResponse;
      const finishBody = {
        id: credential.id,
        rawId: bufferToBase64url(credential.rawId),
        type: credential.type,
        response: {
          attestationObject: bufferToBase64url(response.attestationObject),
          clientDataJSON: bufferToBase64url(response.clientDataJSON),
        },
      };

      await api.webauthnRegisterFinish(label, finishBody);
      setWaLabel('');
      setWaMessage('Security key registered successfully.');
      await load();
    } catch (e: any) {
      setError(e instanceof ApiError ? e.message : (e.message || 'WebAuthn registration failed'));
    }
    setWaBusy(false);
  };

  const handleWaAuthenticate = async () => {
    setWaBusy(true); setWaMessage(''); setError('');
    try {
      if (!isWebAuthnAvailable()) {
        setError('WebAuthn is not supported in this browser or context.');
        setWaBusy(false);
        return;
      }
      const startResp = await api.webauthnAuthStart();
      const opts = startResp.options;

      const publicKey: PublicKeyCredentialRequestOptions = {
        challenge: base64urlToBuffer(opts.challenge),
        rpId: opts.rpId,
        timeout: opts.timeout,
        allowCredentials: (opts.allowCredentials || []).map((c: any) => ({
          type: c.type,
          id: base64urlToBuffer(c.id),
          transports: c.transports || [],
        })),
        userVerification: opts.userVerification || 'preferred',
      };

      const assertion = await navigator.credentials.get({ publicKey }) as PublicKeyCredential;
      if (!assertion) throw new Error('Authentication cancelled');

      const response = assertion.response as AuthenticatorAssertionResponse;
      const finishBody = {
        id: assertion.id,
        rawId: bufferToBase64url(assertion.rawId),
        type: assertion.type,
        response: {
          authenticatorData: bufferToBase64url(response.authenticatorData),
          clientDataJSON: bufferToBase64url(response.clientDataJSON),
          signature: bufferToBase64url(response.signature),
          userHandle: response.userHandle ? bufferToBase64url(response.userHandle) : null,
        },
      };

      await api.webauthnAuthFinish(finishBody);
      setWaMessage('WebAuthn verification successful. MFA is now active for 5 minutes.');
    } catch (e: any) {
      setError(e instanceof ApiError ? e.message : (e.message || 'WebAuthn authentication failed'));
    }
    setWaBusy(false);
  };

  const handleWaDisable = async (credentialId: string) => {
    setError('');
    try {
      await api.webauthnDisableCredential(credentialId);
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

      {/* WebAuthn / Security Keys */}
      <section className="p-4 bg-[var(--card)] border border-[var(--border)] rounded-lg" data-testid="webauthn-section">
        <h2 className="text-lg font-semibold mb-2">Security Keys (WebAuthn)</h2>
        {!isWebAuthnAvailable() ? (
          <p className="text-sm text-yellow-500" data-testid="webauthn-unavailable">
            ⚠ WebAuthn is not available in this browser or context. A secure context (HTTPS or localhost) is required.
          </p>
        ) : (
          <>
            {waCredentials.length > 0 && (
              <div className="mt-3 space-y-2">
                {waCredentials.map(c => (
                  <div key={c.id} className="flex items-center justify-between p-2 bg-[var(--bg)] rounded text-sm">
                    <div data-testid={`wa-cred-${c.id}`}>
                      <span className="font-medium">{c.label}</span>
                      <span className={`ml-2 text-xs ${c.status === 'active' ? 'text-[var(--success)]' : 'text-[var(--text-muted)]'}`}>
                        {c.status}
                      </span>
                      {c.last_used_at && (
                        <span className="ml-2 text-xs text-[var(--text-muted)]">
                          Last used: {new Date(c.last_used_at).toLocaleDateString()}
                        </span>
                      )}
                    </div>
                    {c.status === 'active' && (
                      <button
                        onClick={() => handleWaDisable(c.id)}
                        data-testid={`wa-disable-${c.id}`}
                        className="text-xs px-2 py-1 bg-red-900/30 border border-red-700 rounded hover:bg-red-900/50"
                      >
                        Disable
                      </button>
                    )}
                  </div>
                ))}
              </div>
            )}

            <div className="mt-3 space-y-2">
              <div className="flex gap-2">
                <input
                  type="text"
                  placeholder="Label (e.g. YubiKey, Touch ID)"
                  value={waLabel}
                  onChange={e => setWaLabel(e.target.value)}
                  data-testid="wa-label-input"
                  className="flex-1 px-3 py-1.5 bg-[var(--bg)] border border-[var(--border)] rounded text-sm"
                  maxLength={100}
                />
                <button
                  onClick={handleWaRegister}
                  disabled={waBusy}
                  data-testid="wa-register-btn"
                  className="px-4 py-1.5 bg-[var(--primary)] text-white rounded text-sm hover:opacity-90"
                >
                  {waBusy ? '...' : 'Add Security Key'}
                </button>
              </div>
              <button
                onClick={handleWaAuthenticate}
                disabled={waBusy || waCredentials.length === 0}
                data-testid="wa-auth-btn"
                className="px-4 py-1.5 bg-[var(--success)] text-white rounded text-sm hover:opacity-90 disabled:opacity-50"
              >
                Verify with Security Key
              </button>
            </div>
            {waMessage && (
              <p className="text-sm text-[var(--success)] mt-2" data-testid="wa-message">{waMessage}</p>
            )}
          </>
        )}
      </section>
    </div>
  );
}
