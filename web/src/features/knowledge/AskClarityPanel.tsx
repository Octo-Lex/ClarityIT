import { useState } from 'react';
import { useMutation } from '@tanstack/react-query';
import { Sparkles, X } from 'lucide-react';
import { api } from '@/api/client';
import type { AskClarityResponse } from '@/api/client';
import { Textarea } from '@/components/ui/textarea';
import { Button } from '@/components/ui/button';
import { InlineSpinner } from '@/components/PageState';
import { cn } from '@/lib/utils';
import { AskClarityAnswer } from './AskClarityAnswer';

const SOURCE_TYPE_OPTIONS = [
  { key: 'clarity_document', label: 'Docs' },
  { key: 'artifact', label: 'Artifacts' },
  { key: 'incident', label: 'Incidents' },
  { key: 'work_item', label: 'Work Items' },
  { key: 'template', label: 'Templates' },
];

export function AskClarityPanel() {
  const [question, setQuestion] = useState('');
  const [selectedTypes, setSelectedTypes] = useState<string[]>([]);

  // One-shot Q&A mutation (not a cacheable query — the answer depends on the
  // exact question + filter combination at ask-time).
  const askMut = useMutation({
    mutationFn: () => api.askClarity(question.trim(), selectedTypes.length > 0 ? selectedTypes : undefined, 8),
  });

  const response = askMut.data ?? null;
  const loading = askMut.isPending;
  const error = askMut.isError;

  const handleAsk = () => {
    if (question.trim().length < 5) return;
    askMut.mutate();
  };

  const handleClear = () => {
    setQuestion('');
    askMut.reset();
  };

  const toggleType = (type: string) => {
    setSelectedTypes(prev => prev.includes(type) ? prev.filter(t => t !== type) : [...prev, type]);
  };

  return (
    <div data-testid="ask-clarity-panel" className="space-y-4 rounded-lg border border-border bg-card p-4">
      <div className="flex items-center gap-2">
        <Sparkles className="size-5 text-primary" />
        <h2 className="font-heading text-lg font-semibold">Ask Clarity</h2>
        <span className="text-xs text-muted-foreground">Source-grounded Q&A</span>
      </div>

      <div className="space-y-2">
        <Textarea
          data-testid="ask-clarity-input"
          value={question}
          onChange={(e) => setQuestion(e.target.value)}
          placeholder="Ask a question about your team's knowledge…"
          rows={2}
          maxLength={1000}
          className="resize-none"
        />

        <div className="flex flex-wrap gap-2">
          {SOURCE_TYPE_OPTIONS.map(opt => (
            <button
              key={opt.key}
              data-testid={`ask-filter-${opt.key}`}
              onClick={() => toggleType(opt.key)}
              className={cn(
                'rounded-full px-2 py-0.5 text-xs transition-colors',
                selectedTypes.includes(opt.key)
                  ? 'bg-primary text-primary-foreground'
                  : 'bg-muted text-muted-foreground hover:bg-accent hover:text-foreground',
              )}
            >
              {opt.label}
            </button>
          ))}
        </div>
      </div>

      <div className="flex gap-2">
        <Button size="sm" data-testid="ask-clarity-button" onClick={handleAsk} disabled={loading || question.trim().length < 5}>
          <Sparkles className="size-4" /> {loading ? 'Asking…' : 'Ask'}
        </Button>
        <Button size="sm" variant="ghost" data-testid="ask-clarity-clear" onClick={handleClear}>
          <X className="size-4" /> Clear
        </Button>
      </div>

      {loading && (
        <div data-testid="ask-clarity-loading">
          <InlineSpinner label="Searching knowledge and generating answer…" />
        </div>
      )}

      {!loading && error && (
        <div data-testid="ask-clarity-error" className="py-4 text-center text-sm text-destructive">
          Failed to get an answer. Please try again.
        </div>
      )}

      {!loading && !error && response && (
        <AskClarityAnswer response={response} question={question} />
      )}
    </div>
  );
}
