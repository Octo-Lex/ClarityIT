import { Check, Circle, Loader2, X } from 'lucide-react';
import { cn } from '@/lib/utils';

interface Props {
  status: 'saved' | 'unsaved' | 'saving' | 'error';
  lastSaved?: string | null;
}

const config = {
  saved: { color: 'text-success', Icon: Check, label: 'Saved', spin: false },
  unsaved: { color: 'text-warning', Icon: Circle, label: 'Unsaved changes', spin: false },
  saving: { color: 'text-info', Icon: Loader2, label: 'Saving…', spin: true },
  error: { color: 'text-destructive', Icon: X, label: 'Save failed', spin: false },
} as const;

export default function DocumentSaveStatus({ status, lastSaved }: Props) {
  const c = config[status];
  const Icon = c.Icon;
  return (
    <div className="flex items-center gap-2 text-xs" data-testid="save-status">
      <Icon className={cn('size-3.5', c.color, c.spin && 'animate-spin')} />
      <span className={c.color}>{c.label}</span>
      {lastSaved && status === 'saved' && (
        <span className="text-muted-foreground">at {new Date(lastSaved).toLocaleTimeString()}</span>
      )}
    </div>
  );
}
