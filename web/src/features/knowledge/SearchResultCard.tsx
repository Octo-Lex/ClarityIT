import React, { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import type { KnowledgeSearchResult } from '../../api/client';
import { getStoredTeamId } from '../../api/client';
import { KnowledgeSourceBadge } from './KnowledgeSourceBadge';
import { KnowledgeSnippet } from './KnowledgeSnippet';
import { getSourceRoute } from './sourceRoute';
import { SaveToCollectionDialog } from './SaveToCollectionDialog';

function formatRank(rank: number): string {
  if (rank >= 0.1) return '★★★';
  if (rank >= 0.01) return '★★';
  return '★';
}

export function SearchResultCard({ result }: { result: KnowledgeSearchResult }) {
  const navigate = useNavigate();
  const [showSave, setShowSave] = useState(false);

  const handleClick = () => {
    const teamId = getStoredTeamId() ?? undefined;
    const route = getSourceRoute(teamId, result.source_type, result.source_id);
    if (route) {
      navigate(route);
    } else {
      // Safe fallback — navigate to knowledge item detail
      navigate(`/knowledge/${result.source_id}`);
    }
  };

  return (
    <div
      data-testid="search-result-card"
      onClick={handleClick}
      className="border border-slate-200 dark:border-slate-700 rounded-lg p-4 hover:border-blue-400 dark:hover:border-blue-500 cursor-pointer transition-colors"
    >
      <div className="flex items-start justify-between gap-3 mb-2">
        <div className="flex items-center gap-2">
          <KnowledgeSourceBadge sourceType={result.source_type} />
          <span className="text-xs text-slate-400" title="Relevance">
            {formatRank(result.rank)}
          </span>
        </div>
        <time className="text-xs text-slate-400">
          {new Date(result.updated_at).toLocaleDateString()}
        </time>
      </div>
      <h3 className="font-semibold text-slate-900 dark:text-slate-100 mb-1">
        {result.title}
      </h3>
      {result.summary && (
        <p className="text-sm text-slate-500 dark:text-slate-400 mb-2 line-clamp-1">
          {result.summary}
        </p>
      )}
      {result.snippet && <KnowledgeSnippet snippet={result.snippet} />}
      <div className="mt-2">
        <button
          data-testid="save-to-collection-btn"
          onClick={(e) => { e.stopPropagation(); setShowSave(true); }}
          className="text-xs text-indigo-600 hover:underline"
        >
          Save to Collection
        </button>
      </div>
      {showSave && (
        <SaveToCollectionDialog
          sourceType={result.source_type}
          sourceId={result.source_id}
          title={result.title}
          onClose={() => setShowSave(false)}
        />
      )}
    </div>
  );
}
