import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { Search as SearchIcon, Inbox } from 'lucide-react';
import { api } from '@/api/client';
import { keys } from '@/api/keys';
import { useAuth } from '@/auth/context';
import { Input } from '@/components/ui/input';
import { Button } from '@/components/ui/button';
import { InlineSpinner } from '@/components/PageState';
import { SearchResultCard } from './SearchResultCard';
import { SearchFilters } from './SearchFilters';
import { AskClarityPanel } from './AskClarityPanel';

export function KnowledgeSearchPage() {
  const { activeTeamId } = useAuth();
  const [query, setQuery] = useState('');
  const [submittedQuery, setSubmittedQuery] = useState('');
  const [sourceType, setSourceType] = useState('all');

  // Search-on-submit: the query is enabled only after the user submits.
  const { data, isFetching, error } = useQuery({
    queryKey: keys.knowledge.search(activeTeamId ?? '', submittedQuery, sourceType),
    queryFn: () => api.knowledgeSearch(submittedQuery, sourceType, 20, 0),
    enabled: submittedQuery.trim().length > 0,
  });

  const hasSearched = submittedQuery.trim().length > 0;
  const results = data?.results ?? [];
  const total = data?.total ?? 0;

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!query.trim()) return;
    setSubmittedQuery(query);
  };

  const handleFilterSelect = (st: string) => {
    if (st === sourceType) return;
    setSourceType(st);
  };

  return (
    <div data-testid="knowledge-page" className="mx-auto max-w-4xl space-y-6">
      <h1 className="font-heading text-2xl font-semibold tracking-tight">Knowledge Search</h1>

      {/* v1.5 Track 5: Ask Clarity Q&A Panel */}
      <AskClarityPanel />

      {/* Search input */}
      <form onSubmit={handleSubmit}>
        <div className="flex gap-2">
          <Input
            type="text"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder="Search documents, artifacts, incidents, work items…"
            data-testid="knowledge-search-input"
            autoFocus
          />
          <Button type="submit" data-testid="knowledge-search-button" disabled={isFetching || !query.trim()}>
            <SearchIcon className="size-4" /> {isFetching ? 'Searching…' : 'Search'}
          </Button>
        </div>
      </form>

      {/* Filters */}
      <SearchFilters active={sourceType} onSelect={handleFilterSelect} />

      {/* States */}
      {isFetching && (
        <div data-testid="knowledge-loading">
          <InlineSpinner label="Searching…" />
        </div>
      )}

      {error && !isFetching && (
        <div data-testid="knowledge-error" className="py-8 text-center text-sm text-destructive">
          Search failed. Please try again.
        </div>
      )}

      {!isFetching && !error && !hasSearched && (
        <div data-testid="knowledge-empty" className="flex flex-col items-center py-12 text-center">
          <div className="mb-3 flex size-10 items-center justify-center rounded-full bg-muted text-muted-foreground">
            <SearchIcon className="size-5" />
          </div>
          <p className="text-sm text-muted-foreground">
            Search across documents, artifacts, incidents, work items, and more.
          </p>
          <p className="mt-1 text-xs text-muted-foreground">Type a query above to get started.</p>
        </div>
      )}

      {!isFetching && !error && hasSearched && results.length === 0 && (
        <div data-testid="knowledge-no-results" className="flex flex-col items-center py-12 text-center">
          <div className="mb-3 flex size-10 items-center justify-center rounded-full bg-muted text-muted-foreground">
            <Inbox className="size-5" />
          </div>
          <p className="text-sm text-muted-foreground">No results found for “{submittedQuery}”.</p>
          <p className="mt-1 text-xs text-muted-foreground">Try different keywords or remove filters.</p>
        </div>
      )}

      {!isFetching && !error && results.length > 0 && (
        <>
          <div className="text-sm text-muted-foreground">
            {total} {total === 1 ? 'result' : 'results'} for “{submittedQuery}”
          </div>
          <div className="space-y-3">
            {results.map((r, i) => (
              <SearchResultCard key={`${r.source_type}-${r.source_id}-${i}`} result={r} />
            ))}
          </div>
        </>
      )}
    </div>
  );
}
