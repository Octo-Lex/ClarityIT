import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { api } from '../../api/client';

export default function ForgotPasswordPage() {
  const nav = useNavigate();
  const [email, setEmail] = useState('');
  const [sent, setSent] = useState(false);
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true); setError('');
    try {
      await api.forgotPassword(email);
      // Always show success — do not reveal whether email exists
      setSent(true);
    } catch {
      // Still show success to prevent email enumeration
      setSent(true);
    } finally { setLoading(false); }
  };

  if (sent) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="card w-full max-w-sm">
          <h1 className="text-2xl font-bold mb-4 text-center">Check Your Email</h1>
          <p className="text-sm text-[var(--text-muted)] text-center mb-4">
            If an account with that email exists, we've sent password reset instructions.
          </p>
          <button onClick={() => nav('/login')} className="w-full">Back to Login</button>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen flex items-center justify-center">
      <div className="card w-full max-w-sm">
        <h1 className="text-2xl font-bold mb-4 text-center">Forgot Password</h1>
        <p className="text-sm text-[var(--text-muted)] mb-4 text-center">Enter your email to receive a reset link.</p>
        {error && <div className="error-msg mb-4">{error}</div>}
        <form onSubmit={submit} className="space-y-3">
          <input type="email" placeholder="Email" value={email} onChange={e => setEmail(e.target.value)} required />
          <button type="submit" disabled={loading} className="w-full">{loading ? 'Sending...' : 'Send Reset Link'}</button>
        </form>
        <p className="text-sm text-center mt-4">
          <button className="text-[var(--primary)] bg-transparent p-0 text-sm" onClick={() => nav('/login')}>Back to Login</button>
        </p>
      </div>
    </div>
  );
}
