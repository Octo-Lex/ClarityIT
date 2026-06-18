import { useState } from 'react';
import { useMutation } from '@tanstack/react-query';
import { api, type ObjectDetail } from '@/api/client';
import { useAuth } from '@/auth/context';
import { Card } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Textarea } from '@/components/ui/textarea';
import { Label } from '@/components/ui/label';
import { Select, SelectTrigger, SelectValue, SelectContent, SelectItem } from '@/components/ui/select';
import { notify } from '@/components/Toaster';

interface Props {
  obj: ObjectDetail;
  onUpdated: () => void;
  onCancel: () => void;
}

const STATUSES = ['open', 'in_progress', 'blocked', 'resolved', 'closed'];
const PRIORITIES = ['none', 'low', 'medium', 'high', 'critical'];

export default function ObjectEditForm({ obj, onUpdated, onCancel }: Props) {
  const { hasPermission } = useAuth();
  const [title, setTitle] = useState(obj.title);
  const [summary, setSummary] = useState(obj.summary || '');
  const [status, setStatus] = useState(obj.status);
  const [priority, setPriority] = useState(obj.priority);
  const [conflict, setConflict] = useState(false);

  // Server-side optimistic concurrency: expected_version + 409 auto-recovery.
  // A 409 means someone else updated this object; we surface the conflict and
  // refresh so the user sees the latest version (preserving the original UX).
  const mutation = useMutation({
    mutationFn: () =>
      api.updateObject(obj.id, { title, summary, status, priority, expected_version: obj.version }),
    onSuccess: () => {
      notify.success('Saved');
      onUpdated();
    },
    onError: (err) => {
      if (err instanceof Error && 'status' in err && err.status === 409) {
        setConflict(true);
        notify.warning('Version conflict', 'Someone else updated this. Refreshing…');
        // Auto-refresh after a beat, matching the original behavior.
        setTimeout(onUpdated, 1500);
      } else {
        notify.mutationError('Update', err);
      }
    },
  });

  if (!hasPermission('objects.update')) {
    return <p className="text-sm text-muted-foreground">No permission to edit.</p>;
  }

  return (
    <Card className="space-y-4 p-5">
      {conflict && (
        <div data-testid="edit-conflict" className="rounded-md border border-warning/40 bg-warning/10 p-3 text-sm text-warning">
          Version conflict — this object was modified by someone else. Refreshing…
        </div>
      )}
      <div className="space-y-1.5">
        <Label htmlFor="obj-title">Title</Label>
        <Input id="obj-title" data-testid="obj-title" value={title} onChange={e => setTitle(e.target.value)} required />
      </div>
      <div className="space-y-1.5">
        <Label htmlFor="obj-summary">Summary</Label>
        <Textarea id="obj-summary" data-testid="obj-summary" value={summary} onChange={e => setSummary(e.target.value)} rows={3} />
      </div>
      <div className="grid grid-cols-2 gap-3">
        <div className="space-y-1.5">
          <Label>Status</Label>
          <Select value={status} onValueChange={(v) => setStatus(v ?? '')}>
            <SelectTrigger data-testid="obj-status"><SelectValue /></SelectTrigger>
            <SelectContent>
              {STATUSES.map(s => <SelectItem key={s} value={s}>{s.replace('_', ' ')}</SelectItem>)}
            </SelectContent>
          </Select>
        </div>
        <div className="space-y-1.5">
          <Label>Priority</Label>
          <Select value={priority} onValueChange={(v) => setPriority(v ?? '')}>
            <SelectTrigger data-testid="obj-priority"><SelectValue /></SelectTrigger>
            <SelectContent>
              {PRIORITIES.map(p => <SelectItem key={p} value={p} className="capitalize">{p}</SelectItem>)}
            </SelectContent>
          </Select>
        </div>
      </div>
      <div className="flex gap-2">
        <Button type="button" data-testid="obj-save" disabled={mutation.isPending} onClick={() => mutation.mutate()}>
          {mutation.isPending ? 'Saving…' : 'Save'}
        </Button>
        <Button type="button" variant="secondary" onClick={onCancel}>Cancel</Button>
      </div>
    </Card>
  );
}
