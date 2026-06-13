import { useState } from 'react';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { api } from '../../api/client';

export default function ResetPasswordPage() {
  const nav = useNavigate();
  const [params] = useSearchParams();
  const token = params.get('token') || '';
  const [password, setPassword] = useState('');
  const [confirm, setConfirm] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);
  const [success, setSuccess] = useState(false);

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (password !== confirm) { setError('Passwords do not match'); return; }
    if (password.length < 8) { setError('Password must be at least 8 characters'); return; }
    if (!token) { setError('Invalid or missing reset token'); return; }
    setLoading(true); setError('');
    try {
      await api.resetPassword(token, password);
      setSuccess(true);
    } catch (err: any) {
      setError(err.message || 'Reset failed');
    } finally { setLoading(false); }
  };

  if (success) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="card w-full max-w-sm">
          <h1 className="text-2xl font-bold mb-4 text-center">Password Reset</h1>
          <p className="text-sm text-[var(--text-muted)] text-center mb-4">Your password has been reset successfully.</p>
          <button onClick={() => nav('/login')} className="w-full">Sign In</button>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen flex items-center justify-center">
      <div className="card w-full max-w-sm">
        <h1 className="text-2xl font-bold mb-4 text-center">Reset Password</h1>
        {error && <div className="error-msg mb-4">{error}</div>}
        <form onSubmit={submit} className="space-y-3">
          <input type="password" placeholder="New Password" value={password} onChange={e => setPassword(e.target.value)} required minLength={8} />
          <input type="password" placeholder="Confirm Password" value={confirm} onChange={e => setConfirm(e.target.value)} required minLength={8} />
          <button type="submit" disabled={loading} className="w-full">{loading ? 'Resetting...' : 'Reset Password'}</button>
        </form>
      </div>
    </div>
  );
}
