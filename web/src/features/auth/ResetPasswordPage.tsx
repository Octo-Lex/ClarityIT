import { useState } from 'react';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { CheckCircle2 } from 'lucide-react';
import { api } from '@/api/client';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { AuthCard } from './AuthCard';

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
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Reset failed');
    } finally { setLoading(false); }
  };

  if (success) {
    return (
      <AuthCard title="Password Reset">
        <div className="flex flex-col items-center gap-3 text-center">
          <CheckCircle2 className="size-10 text-success" />
          <p className="text-sm text-muted-foreground">Your password has been reset successfully.</p>
          <Button className="mt-2 w-full" onClick={() => nav('/login')}>Sign In</Button>
        </div>
      </AuthCard>
    );
  }

  return (
    <AuthCard title="Reset Password">
      {error && (
        <div role="alert" className="mb-4 rounded-md border border-destructive/30 bg-destructive/5 p-3 text-sm text-destructive">
          {error}
        </div>
      )}
      <form onSubmit={submit} className="space-y-4">
        <div className="space-y-1.5">
          <Label htmlFor="reset-password">New Password</Label>
          <Input id="reset-password" type="password" data-testid="reset-password" value={password} onChange={e => setPassword(e.target.value)} required minLength={8} />
        </div>
        <div className="space-y-1.5">
          <Label htmlFor="reset-confirm">Confirm Password</Label>
          <Input id="reset-confirm" type="password" data-testid="reset-confirm" value={confirm} onChange={e => setConfirm(e.target.value)} required minLength={8} />
        </div>
        <Button type="submit" className="w-full" disabled={loading} data-testid="reset-submit">
          {loading ? 'Resetting…' : 'Reset Password'}
        </Button>
      </form>
    </AuthCard>
  );
}
