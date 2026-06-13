import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useAuth } from '../../auth/context';

export default function LoginPage() {
  const nav = useNavigate();
  const { login, register } = useAuth();
  const [isRegister, setRegister] = useState(false);
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [name, setName] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true); setError('');
    try {
      if (isRegister) { await register(name, email, password); } else { await login(email, password); }
      nav('/', { replace: true });
    } catch (err: any) {
      setError(err.message || 'Authentication failed');
    } finally { setLoading(false); }
  };

  return (
    <div className="min-h-screen flex items-center justify-center">
      <div className="card w-full max-w-sm">
        <h1 className="text-2xl font-bold mb-6 text-center">ClarityIT</h1>
        <h2 className="text-sm text-[var(--text-muted)] mb-4 text-center">{isRegister ? 'Create Account' : 'Sign In'}</h2>
        {error && <div className="error-msg mb-4">{error}</div>}
        <form onSubmit={submit} className="space-y-3">
          {isRegister && <input placeholder="Full Name" value={name} onChange={e => setName(e.target.value)} required />}
          <input type="email" placeholder="Email" value={email} onChange={e => setEmail(e.target.value)} required />
          <input type="password" placeholder="Password" value={password} onChange={e => setPassword(e.target.value)} required minLength={8} />
          <button type="submit" disabled={loading} className="w-full">{loading ? '...' : (isRegister ? 'Register' : 'Login')}</button>
        </form>
        <p className="text-sm text-center mt-4 text-[var(--text-muted)]">
          {isRegister ? 'Already have an account?' : "Don't have an account?"}{' '}
          <button className="text-[var(--primary)] bg-transparent p-0 text-sm" onClick={() => { setRegister(!isRegister); setError(''); }}>
            {isRegister ? 'Sign in' : 'Register'}
          </button>
        </p>
        {!isRegister && (
          <p className="text-sm text-center mt-2">
            <button className="text-[var(--text-muted)] bg-transparent p-0 text-sm hover:text-white" onClick={() => nav('/forgot-password')}>Forgot password?</button>
          </p>
        )}
      </div>
    </div>
  );
}
