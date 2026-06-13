import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { api, setAccessToken } from '../../api/client';

export default function BootstrapPage() {
  const nav = useNavigate();
  const [name, setName] = useState('');
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [team, setTeam] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true); setError('');
    try {
      const r = await api.bootstrap({ name, email, password, team_name: team });
      setAccessToken(r.access_token);
      nav('/', { replace: true });
    } catch (err: any) {
      if (err?.status === 409) { nav('/login', { replace: true }); return; }
      setError(err.message || 'Bootstrap failed');
    } finally { setLoading(false); }
  };

  return (
    <div className="min-h-screen flex items-center justify-center">
      <div className="card w-full max-w-md">
        <h1 className="text-2xl font-bold mb-6 text-center">🚀 Bootstrap ClarityIT</h1>
        <p className="text-sm text-[var(--text-muted)] mb-6 text-center">Create the first platform owner and initial team.</p>
        {error && <div className="error-msg mb-4">{error}</div>}
        <form onSubmit={submit} className="space-y-4">
          <input placeholder="Your Name" value={name} onChange={e => setName(e.target.value)} required />
          <input type="email" placeholder="Email" value={email} onChange={e => setEmail(e.target.value)} required />
          <input type="password" placeholder="Password" value={password} onChange={e => setPassword(e.target.value)} required minLength={8} />
          <input placeholder="Team Name" value={team} onChange={e => setTeam(e.target.value)} required />
          <button type="submit" disabled={loading} className="w-full">{loading ? 'Creating...' : 'Bootstrap Platform'}</button>
        </form>
      </div>
    </div>
  );
}
