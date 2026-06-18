import { useState } from 'react';
import { KnowledgeSourceBadge } from './KnowledgeSourceBadge';
import { KnowledgeSnippet } from './KnowledgeSnippet';
import { SaveToCollectionDialog } from './SaveToCollectionDialog';
import type { AskClaritySource } from '@/api/client';

export function AskClaritySourceCard({ source }: { source: AskClaritySource }) {
  const [showSave, setShowSave] = useState(false);

  return (
    <div data-testid="ask-clarity-source-card" className="rounded-lg border border-border p-3">
      <div className="mb-1 flex items-center gap-2">
        <KnowledgeSourceBadge sourceType={source.source_type} />
      </div>
      <h4 className="mb-1 text-sm font-medium">{source.title}</h4>
      {source.snippet && <KnowledgeSnippet snippet={source.snippet} />}
      <button
        data-testid="ask-source-save-to-collection"
        onClick={() => setShowSave(true)}
        className="mt-1 text-xs text-primary hover:underline"
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
