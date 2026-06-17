import React, { useState, useCallback } from 'react';
import { api, ApiError } from '../../api/client';
import type { AskClarityResponse } from '../../api/client';
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
  const [response, setResponse] = useState<AskClarityResponse | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(false);

  const handleAsk = useCallback(async () => {
    if (question.trim().length < 5) return;
    setLoading(true);
    setError(false);
    setResponse(null);
    try {
      const resp = await api.askClarity(question.trim(), selectedTypes.length > 0 ? selectedTypes : undefined, 8);
      setResponse(resp);
    } catch (e) {
      setError(true);
    } finally {
      setLoading(false);
    }
  }, [question, selectedTypes]);

  const handleClear = () => {
    setQuestion('');
    setResponse(null);
    setError(false);
  };

  const toggleType = (type: string) => {
    setSelectedTypes(prev =>
      prev.includes(type) ? prev.filter(t => t !== type) : [...prev, type]
    );
  };

  return (
    <div
      data-testid="ask-clarity-panel"
      className="border border-slate-200 dark:border-slate-700 rounded-lg p-4 space-y-4"
    >
      <div className="flex items-center gap-2">
        <h2 className="text-lg font-semibold text-slate-900 dark:text-slate-100">
          Ask Clarity
        </h2>
        <span className="text-xs text-slate-400">Source-grounded Q&A</span>
      </div>

      {/* Question input */}
      <div className="space-y-2">
        <textarea
          data-testid="ask-clarity-input"
          value={question}
          onChange={(e) => setQuestion(e.target.value)}
          placeholder="Ask a question about your team's knowledge..."
          rows={2}
          maxLength={1000}
          className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-lg bg-white dark:bg-slate-800 text-slate-900 dark:text-slate-100 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 resize-none"
        />

        {/* Source type filters */}
        <div className="flex flex-wrap gap-2">
          {SOURCE_TYPE_OPTIONS.map(opt => (
            <button
              key={opt.key}
              data-testid={`ask-filter-${opt.key}`}
              onClick={() => toggleType(opt.key)}
              className={`px-2 py-0.5 text-xs rounded-full transition-colors ${
                selectedTypes.includes(opt.key)
                  ? 'bg-blue-600 text-white'
                  : 'bg-slate-100 text-slate-600 hover:bg-slate-200 dark:bg-slate-800 dark:text-slate-400'
              }`}
            >
              {opt.label}
            </button>
          ))}
        </div>
      </div>

      {/* Action buttons */}
      <div className="flex gap-2">
        <button
          data-testid="ask-clarity-button"
          onClick={handleAsk}
          disabled={loading || question.trim().length < 5}
          className="px-4 py-1.5 bg-blue-600 text-white text-sm rounded-lg hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
        >
          {loading ? 'Asking...' : 'Ask'}
        </button>
        <button
          data-testid="ask-clarity-clear"
          onClick={handleClear}
          className="px-4 py-1.5 text-sm text-slate-600 hover:text-slate-900 dark:text-slate-400 dark:hover:text-slate-200 rounded-lg transition-colors"
        >
          Clear
        </button>
      </div>

      {/* Loading state */}
      {loading && (
        <div data-testid="ask-clarity-loading" className="text-center py-6">
          <div className="inline-block animate-spin rounded-full h-6 w-6 border-b-2 border-blue-600 mb-2" />
          <p className="text-xs text-slate-500">Searching knowledge and generating answer...</p>
        </div>
      )}

      {/* Error state */}
      {!loading && error && (
        <div data-testid="ask-clarity-error" className="text-center py-4 text-red-500">
          <p className="text-sm">Failed to get an answer. Please try again.</p>
        </div>
      )}

      {/* Response */}
      {!loading && !error && response && (
        <AskClarityAnswer response={response} question={question} />
      )}
    </div>
  );
}
