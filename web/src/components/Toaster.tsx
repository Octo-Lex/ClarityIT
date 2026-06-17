import { Toaster as SonnerToaster, toast } from 'sonner';
import { CheckCircle2, AlertCircle, Info, AlertTriangle } from 'lucide-react';

/**
 * Global toast host. Mount once near the app root. The `toast` helpers below
 * give the whole app a consistent notification vocabulary — replacing the
 * silent success/error handling and the one-off AdminApprovals toasts.
 *
 * Usage:
 *   import { notify } from '@/components/Toaster';
 *   notify.success('Incident created');
 *   notify.error('Save failed', err.message);
 */

export function Toaster() {
  return (
    <SonnerToaster
      position="bottom-right"
      richColors={false}
      closeButton
      toastOptions={{
        classNames: {
          toast: 'rounded-lg border border-border bg-background text-foreground shadow-md',
          description: 'text-muted-foreground',
        },
      }}
    />
  );
}

function icon(kind: 'success' | 'error' | 'warning' | 'info') {
  const cls = 'size-4';
  if (kind === 'success') return <CheckCircle2 className={`${cls} text-success`} />;
  if (kind === 'error') return <AlertCircle className={`${cls} text-destructive`} />;
  if (kind === 'warning') return <AlertTriangle className={`${cls} text-warning`} />;
  return <Info className={`${cls} text-info`} />;
}

export const notify = {
  success: (msg: string, description?: string) =>
    toast.success(msg, { description, icon: icon('success') }),
  error: (msg: string, description?: string) =>
    toast.error(msg, { description, icon: icon('error') }),
  warning: (msg: string, description?: string) =>
    toast.warning(msg, { description, icon: icon('warning') }),
  info: (msg: string, description?: string) =>
    toast.info(msg, { description, icon: icon('info') }),
  /** Standard message for a mutation error — friendly + the underlying detail. */
  mutationError: (action: string, err: unknown) => {
    const detail = err instanceof Error ? err.message : undefined;
    toast.error(`${action} failed`, { description: detail, icon: icon('error') });
  },
};
