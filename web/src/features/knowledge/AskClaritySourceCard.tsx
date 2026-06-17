import React, { useState } from 'react';
import { KnowledgeSourceBadge } from './KnowledgeSourceBadge';
import { KnowledgeSnippet } from './KnowledgeSnippet';
import { SaveToCollectionDialog } from './SaveToCollectionDialog';
import type { AskClaritySource } from '../../api/client';

export function AskClaritySourceCard({ source }: { source: AskClaritySource }) {
  const [showSave, setShowSave] = useState(false);

  return (
    <div
      data-testid="ask-clarity-source-card"
      className="border border-slate-200 dark:border-slate-700 rounded-lg p-3"
    >
      <div className="flex items-center gap-2 mb-1">
        <KnowledgeSourceBadge sourceType={source.source_type} />
      </div>
      <h4 className="text-sm font-medium text-slate-900 dark:text-slate-100 mb-1">
        {source.title}
      </h4>
      {source.snippet && <KnowledgeSnippet snippet={source.snippet} />}
      <button
        data-testid="ask-source-save-to-collection"
        onClick={() => setShowSave(true)}
        className="text-xs text-indigo-600 hover:underline mt-1"
      >
        Save to Collection
      </button>
      {showSave && (
        <SaveToCollectionDialog
          sourceType={source.source_type}
          sourceId={source.source_id}
          title={source.title}
          knowledgeItemId={source.knowledge_item_id}
          onClose={() => setShowSave(false)}
        />
      )}
    </div>
  );
}
