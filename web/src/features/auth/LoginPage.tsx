import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useAuth } from '@/auth/context';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { AuthCard } from './AuthCard';

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
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Authentication failed');
    } finally { setLoading(false); }
  };

  return (
    <AuthCard
      title="ClarityIT"
      subtitle={isRegister ? 'Create your account' : 'Sign in to your account'}
      footer={
        <>
          {isRegister ? 'Already have an account?' : "Don't have an account?"}{' '}
          <button
            type="button"
            data-testid="auth-mode-toggle"
            className="font-medium text-primary hover:underline"
            onClick={() => { setRegister(!isRegister); setError(''); }}
          >
            {isRegister ? 'Sign in' : 'Register'}
          </button>
        </>
      }
    >
      {error && (
        <div role="alert" className="mb-4 rounded-md border border-destructive/30 bg-destructive/5 p-3 text-sm text-destructive">
          {error}
        </div>
      )}
      <form onSubmit={submit} className="space-y-4">
        {isRegister && (
          <div className="space-y-1.5">
            <Label htmlFor="reg-name">Full Name</Label>
            <Input id="reg-name" data-testid="reg-name" value={name} onChange={e => setName(e.target.value)} required />
          </div>
        )}
        <div className="space-y-1.5">
          <Label htmlFor="login-email">Email</Label>
          <Input id="login-email" type="email" data-testid="login-email" placeholder="you@example.com" value={email} onChange={e => setEmail(e.target.value)} required />
        </div>
        <div className="space-y-1.5">
          <Label htmlFor="login-password">Password</Label>
          <Input id="login-password" type="password" data-testid="login-password" placeholder="••••••••" value={password} onChange={e => setPassword(e.target.value)} required minLength={8} />
        </div>
        <Button type="submit" className="w-full" disabled={loading}>
          {loading ? 'Please wait…' : (isRegister ? 'Register' : 'Sign in')}
        </Button>
      </form>
      {!isRegister && (
        <p className="mt-3 text-center text-sm">
          <button type="button" className="text-muted-foreground hover:text-foreground" onClick={() => nav('/forgot-password')}>
            Forgot password?
          </button>
        </p>
      )}
    </AuthCard>
  );
}
