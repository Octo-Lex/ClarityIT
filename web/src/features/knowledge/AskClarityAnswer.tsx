import React from 'react';
import type { AskClarityResponse } from '../../api/client';
import { AskClaritySourceCard } from './AskClaritySourceCard';

const CONFIDENCE_LABELS: Record<string, { label: string; color: string }> = {
  low: { label: 'Low Confidence', color: 'bg-red-100 text-red-700 dark:bg-red-900 dark:text-red-300' },
  medium: { label: 'Medium Confidence', color: 'bg-amber-100 text-amber-700 dark:bg-amber-900 dark:text-amber-300' },
  high: { label: 'High Confidence', color: 'bg-green-100 text-green-700 dark:bg-green-900 dark:text-green-300' },
};

export function AskClarityAnswer({ response }: { response: AskClarityResponse }) {
  const conf = CONFIDENCE_LABELS[response.confidence] ?? CONFIDENCE_LABELS.low;

  return (
    <div data-testid="ask-clarity-answer" className="space-y-4">
      {/* Answer text */}
      <div className="bg-slate-50 dark:bg-slate-800 rounded-lg p-4">
        <div className="flex items-center gap-2 mb-2">
          <span
            data-testid="ask-confidence-badge"
            className={`text-xs font-medium px-2 py-0.5 rounded-full ${conf.color}`}
          >
            {conf.label}
          </span>
        </div>
        <p className="text-sm text-slate-800 dark:text-slate-200 whitespace-pre-wrap">
          {response.answer}
        </p>
      </div>

      {/* Missing info */}
      {response.missing_info && response.missing_info.length > 0 && (
        <div data-testid="ask-missing-info" className="text-sm text-amber-600 dark:text-amber-400">
          {response.missing_info.map((info, i) => (
            <p key={i}>⚠️ {info}</p>
          ))}
        </div>
      )}

      {/* Source citations */}
      {response.sources && response.sources.length > 0 ? (
        <div>
          <h4 className="text-xs font-semibold text-slate-500 uppercase mb-2">
            Sources ({response.sources.length})
          </h4>
          <div className="space-y-2">
            {response.sources.map((src, i) => (
              <AskClaritySourceCard key={i} source={src} />
            ))}
          </div>
        </div>
      ) : (
        <div data-testid="ask-no-sources" className="text-center py-4 text-slate-400">
          <p className="text-sm">No source documents found for this question.</p>
        </div>
      )}
    </div>
  );
}
