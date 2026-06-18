import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { Rocket } from 'lucide-react';
import { api, setAccessToken } from '@/api/client';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { AuthCard } from '../auth/AuthCard';

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
    } catch (err) {
      // 409 = already bootstrapped → send to login (preserved behavior).
      if (err instanceof Error && 'status' in err && err.status === 409) {
        nav('/login', { replace: true });
        return;
      }
      setError(err instanceof Error ? err.message : 'Bootstrap failed');
    } finally { setLoading(false); }
  };

  return (
    <AuthCard
      title="Bootstrap ClarityIT"
      subtitle="Create the first platform owner and initial team."
      maxWidth="max-w-md"
    >
      {error && (
        <div role="alert" className="mb-4 rounded-md border border-destructive/30 bg-destructive/5 p-3 text-sm text-destructive">
          {error}
        </div>
      )}
      <form onSubmit={submit} className="space-y-4">
        <div className="space-y-1.5">
          <Label htmlFor="boot-name">Your Name</Label>
          <Input id="boot-name" data-testid="boot-name" value={name} onChange={e => setName(e.target.value)} required />
        </div>
        <div className="space-y-1.5">
          <Label htmlFor="boot-email">Email</Label>
          <Input id="boot-email" type="email" data-testid="boot-email" value={email} onChange={e => setEmail(e.target.value)} required />
        </div>
        <div className="space-y-1.5">
          <Label htmlFor="boot-password">Password</Label>
          <Input id="boot-password" type="password" data-testid="boot-password" value={password} onChange={e => setPassword(e.target.value)} required minLength={8} />
        </div>
        <div className="space-y-1.5">
          <Label htmlFor="boot-team">Team Name</Label>
          <Input id="boot-team" data-testid="boot-team" value={team} onChange={e => setTeam(e.target.value)} required />
        </div>
        <Button type="submit" className="w-full" disabled={loading} data-testid="boot-submit">
          <Rocket className="size-4" /> {loading ? 'Creating…' : 'Bootstrap Platform'}
        </Button>
      </form>
    </AuthCard>
  );
}
