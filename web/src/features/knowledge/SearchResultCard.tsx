import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { Star } from 'lucide-react';
import type { KnowledgeSearchResult } from '@/api/client';
import { getStoredTeamId } from '@/api/client';
import { KnowledgeSourceBadge } from './KnowledgeSourceBadge';
import { KnowledgeSnippet } from './KnowledgeSnippet';
import { getSourceRoute } from './sourceRoute';
import { SaveToCollectionDialog } from './SaveToCollectionDialog';

function rankTone(rank: number): 'warning' | 'neutral' {
  return rank >= 0.01 ? 'warning' : 'neutral';
}

export function SearchResultCard({ result }: { result: KnowledgeSearchResult }) {
  const navigate = useNavigate();
  const [showSave, setShowSave] = useState(false);

  const handleClick = () => {
    const teamId = getStoredTeamId() ?? undefined;
    const route = getSourceRoute(teamId, result.source_type, result.source_id);
    if (route) navigate(route);
    else navigate(`/knowledge/${result.source_id}`);
  };

  return (
    <div
      data-testid="search-result-card"
      onClick={handleClick}
      className="cursor-pointer rounded-lg border border-border p-4 transition-colors hover:border-primary/50"
    >
      <div className="mb-2 flex items-start justify-between gap-3">
        <div className="flex items-center gap-2">
          <KnowledgeSourceBadge sourceType={result.source_type} />
          <Star className={`size-3.5 ${rankTone(result.rank) === 'warning' ? 'fill-warning text-warning' : 'fill-muted-foreground text-muted-foreground'}`} />
        </div>
        <time className="text-xs text-muted-foreground">
          {new Date(result.updated_at).toLocaleDateString()}
        </time>
      </div>
      <h3 className="mb-1 font-heading font-semibold">{result.title}</h3>
      {result.summary && (
        <p className="mb-2 line-clamp-1 text-sm text-muted-foreground">{result.summary}</p>
      )}
      {result.snippet && <KnowledgeSnippet snippet={result.snippet} />}
      <div className="mt-2">
        <button
          data-testid="save-to-collection-btn"
          onClick={(e) => { e.stopPropagation(); setShowSave(true); }}
          className="text-xs text-primary hover:underline"
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
