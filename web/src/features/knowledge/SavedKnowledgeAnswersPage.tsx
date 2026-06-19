import { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { api, type SavedKnowledgeAnswer } from '../../api/client';

export function SavedKnowledgeAnswersPage() {
  const navigate = useNavigate();
  const [answers, setAnswers] = useState<SavedKnowledgeAnswer[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    (async () => {
      try {
        const resp = await api.listSavedAnswers();
        setAnswers(resp.answers);
      } catch {
        setError('Failed to load saved answers');
      } finally {
        setLoading(false);
      }
    })();
  }, []);

  if (loading) return <div data-testid="saved-answers-loading" className="p-8 text-center text-muted-foreground">Loading…</div>;
  if (error) return <div data-testid="saved-answers-error" className="p-8 text-center text-destructive">{error}</div>;

  return (
    <div className="max-w-4xl mx-auto p-6">
      <h1 className="text-2xl font-bold mb-6">Saved Answers</h1>

      {answers.length === 0 ? (
        <div data-testid="saved-answers-empty" className="text-center py-16 text-muted-foreground">
          No saved answers yet. Ask a question and save useful answers here.
        </div>
      ) : (
        <div className="space-y-3">
          {answers.map((a) => (
            <div
              key={a.id}
              data-testid="saved-answer-card"
              className="p-4 border rounded-lg hover:border-indigo-300 cursor-pointer"
              onClick={() => navigate(`/knowledge/saved-answers/${a.id}`)}
            >
              <div className="flex items-center gap-2 mb-2">
                <ConfidenceBadge confidence={a.confidence} />
                <span className="text-muted-foreground text-xs">{new Date(a.created_at).toLocaleDateString()}</span>
              </div>
              <p className="font-medium text-foreground">{a.question}</p>
              <p className="text-muted-foreground text-sm mt-1 line-clamp-2">{a.answer}</p>
              <p className="text-muted-foreground text-xs mt-1">{a.sources?.length || 0} source{(a.sources?.length || 0) !== 1 ? 's' : ''}</p>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

function ConfidenceBadge({ confidence }: { confidence: string }) {
  const colors: Record<string, string> = {
    high: 'bg-success/15 text-success',
    medium: 'bg-warning/20 text-warning',
    low: 'bg-muted text-muted-foreground',
  };
  return (
    <span className={`px-2 py-0.5 rounded-full text-xs font-medium ${colors[confidence] || colors.low}`}>
      {confidence}
    </span>
  );
}
