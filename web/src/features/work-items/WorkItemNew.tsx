import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { z } from 'zod';
import { api } from '@/api/client';
import { keys } from '@/api/keys';
import { useAuth } from '@/auth/context';
import { Card } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Textarea } from '@/components/ui/textarea';
import { Label } from '@/components/ui/label';
import { Select, SelectTrigger, SelectValue, SelectContent, SelectItem } from '@/components/ui/select';
import { notify } from '@/components/Toaster';

// Mirror backend validation: title required, type/priority enumerated.
const schema = z.object({
  title: z.string().min(1, 'Title is required').max(200, 'Title is too long'),
  summary: z.string().max(2000, 'Summary is too long').optional().default(''),
  work_item_type: z.enum(['task', 'bug', 'ticket', 'change']),
  priority: z.enum(['none', 'low', 'medium', 'high', 'critical']),
});

const TYPES = [
  { value: 'task', label: 'Task' },
  { value: 'bug', label: 'Bug' },
  { value: 'ticket', label: 'Ticket' },
  { value: 'change', label: 'Change' },
];
const PRIORITIES = [
  { value: 'none', label: 'No Priority' },
  { value: 'low', label: 'Low' },
  { value: 'medium', label: 'Medium' },
  { value: 'high', label: 'High' },
  { value: 'critical', label: 'Critical' },
];

export default function WorkItemNew() {
  const nav = useNavigate();
  const { activeTeamId } = useAuth();
  const queryClient = useQueryClient();
  const [title, setTitle] = useState('');
  const [summary, setSummary] = useState('');
  const [workType, setWorkType] = useState('task');
  const [priority, setPriority] = useState('none');
  const [errors, setErrors] = useState<Record<string, string>>({});

  const mutation = useMutation({
    mutationFn: (input: z.infer<typeof schema>) =>
      api.createWorkItem({ ...input, status: 'open' }),
    onSuccess: (res) => {
      notify.success('Work item created');
      // Invalidate the queue + board so the new item shows immediately.
      queryClient.invalidateQueries({ queryKey: keys.workItems.list(activeTeamId ?? '') });
      queryClient.invalidateQueries({ queryKey: keys.workItems.board(activeTeamId ?? '') });
      nav(`/objects/${res.id}`, { replace: true });
    },
    onError: (err) => notify.mutationError('Create', err),
  });

  const submit = (e: React.FormEvent) => {
    e.preventDefault();
    const parsed = schema.safeParse({ title, summary, work_item_type: workType, priority });
    if (!parsed.success) {
      const fieldErrors: Record<string, string> = {};
      for (const issue of parsed.error.issues) {
        fieldErrors[issue.path[0] as string] = issue.message;
      }
      setErrors(fieldErrors);
      return;
    }
    setErrors({});
    mutation.mutate(parsed.data);
  };

  return (
    <div className="mx-auto max-w-lg space-y-4">
      <h1 className="font-heading text-2xl font-semibold tracking-tight">New Work Item</h1>
      <form onSubmit={submit}>
        <Card className="space-y-4 p-5">
          <div className="space-y-1.5">
            <Label htmlFor="wi-title">Title *</Label>
            <Input
              id="wi-title" data-testid="wi-title"
              placeholder="What needs to be done?"
              value={title} onChange={e => setTitle(e.target.value)}
              aria-invalid={!!errors.title}
            />
            {errors.title && <p className="text-xs text-destructive">{errors.title}</p>}
          </div>
          <div className="space-y-1.5">
            <Label htmlFor="wi-summary">Summary</Label>
            <Textarea
              id="wi-summary" data-testid="wi-summary"
              value={summary} onChange={e => setSummary(e.target.value)} rows={3}
            />
            {errors.summary && <p className="text-xs text-destructive">{errors.summary}</p>}
          </div>
          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1.5">
              <Label>Type</Label>
              <Select value={workType} onValueChange={(v) => setWorkType(v ?? 'task')}>
                <SelectTrigger data-testid="wi-type"><SelectValue /></SelectTrigger>
                <SelectContent>
                  {TYPES.map(t => <SelectItem key={t.value} value={t.value}>{t.label}</SelectItem>)}
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-1.5">
              <Label>Priority</Label>
              <Select value={priority} onValueChange={(v) => setPriority(v ?? 'none')}>
                <SelectTrigger data-testid="wi-priority"><SelectValue /></SelectTrigger>
                <SelectContent>
                  {PRIORITIES.map(p => <SelectItem key={p.value} value={p.value}>{p.label}</SelectItem>)}
                </SelectContent>
              </Select>
            </div>
          </div>
          <div className="flex gap-2">
            <Button type="submit" data-testid="wi-create" disabled={mutation.isPending}>
              {mutation.isPending ? 'Creating…' : 'Create'}
            </Button>
            <Button type="button" variant="secondary" onClick={() => nav(-1)}>Cancel</Button>
          </div>
        </Card>
      </form>
    </div>
  );
}
