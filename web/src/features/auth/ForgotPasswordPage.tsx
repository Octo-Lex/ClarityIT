import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { Mail, CheckCircle2 } from 'lucide-react';
import { api } from '@/api/client';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { AuthCard } from './AuthCard';

export default function ForgotPasswordPage() {
  const nav = useNavigate();
  const [email, setEmail] = useState('');
  const [sent, setSent] = useState(false);
  const [loading, setLoading] = useState(false);

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    try {
      await api.forgotPassword(email);
    } catch {
      // Intentionally swallowed: show success regardless to prevent email
      // enumeration (security contract — a missing account must look identical
      // to a valid one). The backend also always returns 200.
    } finally {
      setLoading(false);
      setSent(true);
    }
  };

  if (sent) {
    return (
      <AuthCard title="Check Your Email">
        <div className="flex flex-col items-center gap-3 text-center">
          <CheckCircle2 className="size-10 text-success" />
          <p className="text-sm text-muted-foreground">
            If an account with that email exists, we've sent password reset instructions.
          </p>
          <Button className="mt-2 w-full" onClick={() => nav('/login')}>Back to Login</Button>
        </div>
      </AuthCard>
    );
  }

  return (
    <AuthCard
      title="Forgot Password"
      subtitle="Enter your email to receive a reset link."
      footer={
        <button type="button" className="font-medium text-primary hover:underline" onClick={() => nav('/login')}>
          Back to Login
        </button>
      }
    >
      <form onSubmit={submit} className="space-y-4">
        <div className="space-y-1.5">
          <Label htmlFor="forgot-email">Email</Label>
          <Input id="forgot-email" type="email" data-testid="forgot-email" placeholder="you@example.com" value={email} onChange={e => setEmail(e.target.value)} required />
        </div>
        <Button type="submit" className="w-full" disabled={loading} data-testid="forgot-submit">
          <Mail className="size-4" /> {loading ? 'Sending…' : 'Send Reset Link'}
        </Button>
      </form>
    </AuthCard>
  );
}
