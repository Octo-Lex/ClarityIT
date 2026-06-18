import { ReactNode } from 'react';
import { ShieldCheck } from 'lucide-react';
import { Card } from '@/components/ui/card';

/**
 * Shared layout for the standalone auth screens (login, register, forgot/reset
 * password, bootstrap) — a centered card with the brand mark. Keeps these
 * full-viewport screens visually consistent.
 */
export function AuthCard({
  title, subtitle, children, footer, maxWidth = 'max-w-sm',
}: {
  title: string;
  subtitle?: string;
  children: ReactNode;
  footer?: ReactNode;
  maxWidth?: 'max-w-sm' | 'max-w-md';
}) {
  return (
    <div className="flex min-h-screen items-center justify-center bg-background p-4">
      <Card className={`w-full ${maxWidth} p-6`}>
        <div className="mb-6 flex flex-col items-center gap-2 text-center">
          <div className="flex size-10 items-center justify-center rounded-xl bg-primary text-lg font-bold text-primary-foreground">
            <ShieldCheck className="size-5" />
          </div>
          <h1 className="font-heading text-xl font-semibold tracking-tight">{title}</h1>
          {subtitle && <p className="text-sm text-muted-foreground">{subtitle}</p>}
        </div>
        {children}
        {footer && <div className="mt-4 text-center text-sm text-muted-foreground">{footer}</div>}
      </Card>
    </div>
  );
}
