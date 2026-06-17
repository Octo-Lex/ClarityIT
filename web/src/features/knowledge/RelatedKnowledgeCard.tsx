import React, { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import type { RelatedKnowledgeItem } from '../../api/client';
import { getStoredTeamId } from '../../api/client';
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
    if (route) {
      navigate(route);
    } else {
      navigate(`/knowledge/${item.source_id}`);
    }
  };

  return (
    <div
      data-testid="related-knowledge-card"
      onClick={handleClick}
      className="border border-slate-200 dark:border-slate-700 rounded-lg p-3 hover:border-blue-400 dark:hover:border-blue-500 cursor-pointer transition-colors"
    >
      <div className="flex items-center gap-2 mb-1">
        <KnowledgeSourceBadge sourceType={item.source_type} />
        <RelatedKnowledgeReasonBadge reason={item.reason} />
      </div>
      <h4 className="text-sm font-medium text-slate-900 dark:text-slate-100 line-clamp-1 mb-1">
        {item.title}
      </h4>
      {item.summary && (
        <p className="text-xs text-slate-500 dark:text-slate-400 line-clamp-2">
          {item.summary}
        </p>
      )}
      <time className="text-xs text-slate-400 mt-1 block">
        {new Date(item.updated_at).toLocaleDateString()}
      </time>
      <button
        data-testid="related-save-to-collection"
        onClick={(e) => { e.stopPropagation(); setShowSave(true); }}
        className="text-xs text-indigo-600 hover:underline mt-1"
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
