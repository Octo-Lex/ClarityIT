import { useState } from 'react';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { Save, AlertTriangle } from 'lucide-react';
import { api } from '@/api/client';
import { keys } from '@/api/keys';
import type { AskClarityResponse } from '@/api/client';
import { useAuth } from '@/auth/context';
import { Button } from '@/components/ui/button';
import { StatusBadge } from '@/components/ui/status-badge';
import { notify } from '@/components/Toaster';
import { AskClaritySourceCard } from './AskClaritySourceCard';

const CONFIDENCE_TONE: Record<string, 'danger' | 'warning' | 'success'> = {
  low: 'danger',
  medium: 'warning',
  high: 'success',
};
const CONFIDENCE_LABEL: Record<string, string> = {
  low: 'Low Confidence',
  medium: 'Medium Confidence',
  high: 'High Confidence',
};

export function AskClarityAnswer({ response, question }: { response: AskClarityResponse; question: string }) {
  const { activeTeamId } = useAuth();
  const queryClient = useQueryClient();
  const [saved, setSaved] = useState(false);

  const saveMut = useMutation({
    mutationFn: () => api.saveAnswer({
      question,
      answer: response.answer,
      confidence: response.confidence,
      sources: response.sources,
    }),
    onSuccess: () => {
      setSaved(true);
      queryClient.invalidateQueries({ queryKey: keys.knowledge.savedAnswers.list(activeTeamId ?? '') });
      notify.success('Answer saved');
    },
    onError: () => {
      // Safe degradation — the original swallowed errors; keep that behavior
      // (saving an answer is non-critical) but still surface a toast.
      notify.warning('Could not save answer', 'Please try again.');
    },
  });

  const conf = response.confidence;

  return (
    <div data-testid="ask-clarity-answer" className="space-y-4">
      <div className="rounded-lg bg-muted p-4">
        <div className="mb-2 flex items-center justify-between gap-2">
          <StatusBadge data-testid="ask-confidence-badge" tone={CONFIDENCE_TONE[conf] ?? 'danger'}>
            {CONFIDENCE_LABEL[conf] ?? 'Low Confidence'}
          </StatusBadge>
          <Button
            size="sm" variant="secondary"
            data-testid="ask-save-answer"
            onClick={() => saveMut.mutate()}
            disabled={saveMut.isPending || saved}
          >
            <Save className="size-4" /> {saved ? 'Saved' : saveMut.isPending ? 'Saving…' : 'Save Answer'}
          </Button>
        </div>
        <p className="whitespace-pre-wrap text-sm">{response.answer}</p>
      </div>

      {response.missing_info && response.missing_info.length > 0 && (
        <div data-testid="ask-missing-info" className="space-y-1 text-sm text-warning">
          {response.missing_info.map((info, i) => (
            <p key={i} className="flex items-center gap-1.5"><AlertTriangle className="size-4 shrink-0" /> {info}</p>
          ))}
        </div>
      )}

      {response.sources && response.sources.length > 0 ? (
        <div>
          <h4 className="mb-2 text-xs font-semibold uppercase text-muted-foreground">
            Sources ({response.sources.length})
          </h4>
          <div className="space-y-2">
            {response.sources.map((src, i) => (
              <AskClaritySourceCard key={i} source={src} />
            ))}
          </div>
        </div>
      ) : (
        <div data-testid="ask-no-sources" className="py-4 text-center text-sm text-muted-foreground">
          No source documents found for this question.
        </div>
      )}
    </div>
  );
}
