import { useState, useEffect } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { api, type SavedKnowledgeAnswer } from '../../api/client';
import { AskClaritySourceCard } from './AskClaritySourceCard';

export function SavedKnowledgeAnswerDetailPage() {
  const { answerId } = useParams<{ answerId: string }>();
  const navigate = useNavigate();
  const [answer, setAnswer] = useState<SavedKnowledgeAnswer | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    (async () => {
      if (!answerId) return;
      try {
        const resp = await api.getSavedAnswer(answerId);
        setAnswer(resp);
      } catch {
        setError('Failed to load saved answer');
      } finally {
        setLoading(false);
      }
    })();
  }, [answerId]);

  if (loading) return <div data-testid="saved-answer-detail-loading" className="p-8 text-center text-gray-500">Loading…</div>;
  if (error) return <div data-testid="saved-answer-detail-error" className="p-8 text-center text-red-500">{error}</div>;
  if (!answer) return null;

  return (
    <div className="max-w-4xl mx-auto p-6">
      <button onClick={() => navigate('/knowledge/saved-answers')} className="text-indigo-600 mb-4 hover:underline">
        ← Back to Saved Answers
      </button>

      <div className="mb-6">
        <div className="flex items-center gap-3 mb-3">
          <ConfidenceBadge confidence={answer.confidence} />
          <span className="text-gray-400 text-sm">{new Date(answer.created_at).toLocaleString()}</span>
        </div>
        <h1 className="text-xl font-semibold mb-3">{answer.question}</h1>
        <div data-testid="saved-answer-detail-text" className="prose max-w-none text-gray-700 whitespace-pre-wrap">
          {answer.answer}
        </div>
      </div>

      {answer.sources && answer.sources.length > 0 && (
        <div className="mt-6">
          <h2 className="text-lg font-semibold mb-3">Sources</h2>
          <div className="space-y-2">
            {answer.sources.map((src, i) => (
              <AskClaritySourceCard key={i} source={src} />
            ))}
          </div>
        </div>
      )}

      <button
        data-testid="saved-answer-delete"
        onClick={async () => {
          try {
            await api.deleteSavedAnswer(answer.id);
            navigate('/knowledge/saved-answers');
          } catch {
            setError('Failed to delete answer');
          }
        }}
        className="mt-6 text-red-500 hover:underline text-sm"
      >
        Delete Answer
      </button>
    </div>
  );
}

function ConfidenceBadge({ confidence }: { confidence: string }) {
  const colors: Record<string, string> = {
    high: 'bg-green-100 text-green-700',
    medium: 'bg-yellow-100 text-yellow-700',
    low: 'bg-gray-100 text-gray-600',
  };
  return (
    <span className={`px-2 py-0.5 rounded-full text-xs font-medium ${colors[confidence] || colors.low}`}>
      {confidence}
    </span>
  );
}
