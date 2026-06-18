import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import type { RelatedKnowledgeItem } from '@/api/client';
import { getStoredTeamId } from '@/api/client';
import { KnowledgeSourceBadge } from './KnowledgeSourceBadge';
import { RelatedKnowledgeReasonBadge } from './RelatedKnowledgeReasonBadge';
import { getSourceRoute } from './sourceRoute';
import { SaveToCollectionDialog } from './SaveToCollectionDialog';

export function RelatedKnowledgeCard({ item }: { item: RelatedKnowledgeItem }) {
  const navigate = useNavigate();
  const [showSave, setShowSave] = useState(false);

  const handleClick = () => {
    const teamId = getStoredTeamId() ?? undefined;
    const route = getSourceRoute(teamId, item.source_type, item.source_id);
    if (route) navigate(route);
    else navigate(`/knowledge/${item.source_id}`);
  };

  return (
    <div
      data-testid="related-knowledge-card"
      onClick={handleClick}
      className="cursor-pointer rounded-lg border border-border p-3 transition-colors hover:border-primary/50"
    >
      <div className="mb-1 flex items-center gap-2">
        <KnowledgeSourceBadge sourceType={item.source_type} />
        <RelatedKnowledgeReasonBadge reason={item.reason} />
      </div>
      <h4 className="mb-1 line-clamp-1 text-sm font-medium">{item.title}</h4>
      {item.summary && <p className="line-clamp-2 text-xs text-muted-foreground">{item.summary}</p>}
      <time className="mt-1 block text-xs text-muted-foreground">
        {new Date(item.updated_at).toLocaleDateString()}
      </time>
      <button
        data-testid="related-save-to-collection"
        onClick={(e) => { e.stopPropagation(); setShowSave(true); }}
        className="mt-1 text-xs text-primary hover:underline"
      >
        Save to Collection
      </button>
      {showSave && (
        <SaveToCollectionDialog
          sourceType={item.source_type}
          sourceId={item.source_id}
          title={item.title}
          onClose={() => setShowSave(false)}
        />
      )}
    </div>
  );
}
