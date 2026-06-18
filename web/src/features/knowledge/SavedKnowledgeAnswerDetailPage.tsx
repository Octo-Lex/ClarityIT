import { useParams, useNavigate } from 'react-router-dom';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { ArrowLeft, Trash2 } from 'lucide-react';
import { api } from '@/api/client';
import { keys } from '@/api/keys';
import { useAuth } from '@/auth/context';
import { Button } from '@/components/ui/button';
import { StatusBadge } from '@/components/ui/status-badge';
import { notify } from '@/components/Toaster';
import { InlineSpinner, ErrorState } from '@/components/PageState';
import { AskClaritySourceCard } from './AskClaritySourceCard';

const CONFIDENCE_TONE: Record<string, 'success' | 'warning' | 'neutral'> = {
  high: 'success', medium: 'warning', low: 'neutral',
};

export function SavedKnowledgeAnswerDetailPage() {
  const { answerId } = useParams<{ answerId: string }>();
  const navigate = useNavigate();
  const { activeTeamId } = useAuth();
  const queryClient = useQueryClient();

  const answerQ = useQuery({
    queryKey: keys.knowledge.savedAnswers.detail(activeTeamId ?? '', answerId ?? ''),
    queryFn: () => api.getSavedAnswer(answerId!),
    enabled: !!answerId,
  });

  const deleteMut = useMutation({
    mutationFn: () => api.deleteSavedAnswer(answerId!),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: keys.knowledge.savedAnswers.list(activeTeamId ?? '') });
      notify.success('Answer deleted');
      navigate('/knowledge/saved-answers');
    },
    onError: () => notify.error('Failed to delete answer'),
  });

  if (answerQ.isPending) return <div data-testid="saved-answer-detail-loading"><InlineSpinner /></div>;
  if (answerQ.isError) return <div data-testid="saved-answer-detail-error" className="p-4"><ErrorState message="Failed to load saved answer" onRetry={() => answerQ.refetch()} /></div>;
  const answer = answerQ.data;
  if (!answer) return null;

  return (
    <div className="mx-auto max-w-4xl space-y-6">
      <button onClick={() => navigate('/knowledge/saved-answers')} className="inline-flex items-center gap-1 text-sm text-muted-foreground hover:text-foreground">
        <ArrowLeft className="size-4" /> Back to Saved Answers
      </button>

      <div>
        <div className="mb-3 flex items-center gap-3">
          <StatusBadge tone={CONFIDENCE_TONE[answer.confidence] ?? 'neutral'}>{answer.confidence}</StatusBadge>
          <span className="text-sm text-muted-foreground">{new Date(answer.created_at).toLocaleString()}</span>
        </div>
        <h1 className="mb-3 font-heading text-xl font-semibold">{answer.question}</h1>
        <div data-testid="saved-answer-detail-text" className="whitespace-pre-wrap text-sm">
          {answer.answer}
        </div>
      </div>

      {answer.sources && answer.sources.length > 0 && (
        <div>
          <h2 className="mb-3 font-heading text-lg font-semibold">Sources</h2>
          <div className="space-y-2">
            {answer.sources.map((src, i) => (
              <AskClaritySourceCard key={i} source={src} />
            ))}
          </div>
        </div>
      )}

      <Button
        variant="ghost"
        data-testid="saved-answer-delete"
        onClick={() => { if (confirm('Delete this saved answer?')) deleteMut.mutate(); }}
        className="text-destructive hover:text-destructive"
      >
        <Trash2 className="size-4" /> Delete Answer
      </Button>
    </div>
  );
}
