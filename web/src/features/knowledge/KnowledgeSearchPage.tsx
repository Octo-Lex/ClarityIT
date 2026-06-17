import React, { useState, useCallback } from 'react';
import { api, ApiError } from '../../api/client';
import type { KnowledgeSearchResult } from '../../api/client';
import { SearchResultCard } from './SearchResultCard';
import { SearchFilters } from './SearchFilters';

export function KnowledgeSearchPage() {
  const [query, setQuery] = useState('');
  const [submittedQuery, setSubmittedQuery] = useState('');
  const [sourceType, setSourceType] = useState('all');
  const [results, setResults] = useState<KnowledgeSearchResult[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [hasSearched, setHasSearched] = useState(false);

  const doSearch = useCallback(
    async (q: string, st: string) => {
      if (!q.trim()) return;
      setLoading(true);
      setError(null);
      try {
        const resp = await api.knowledgeSearch(q, st, 20, 0);
        setResults(resp.results ?? []);
        setTotal(resp.total);
        setHasSearched(true);
      } catch (e) {
        if (e instanceof ApiError) {
          setError('Search failed. Please try again.');
        } else {
          setError('An unexpected error occurred.');
        }
        setResults([]);
        setTotal(0);
        setHasSearched(true);
      } finally {
        setLoading(false);
      }
    },
    []
  );

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    setSubmittedQuery(query);
    doSearch(query, sourceType);
  };

  const handleFilterSelect = (st: string) => {
    if (st === sourceType) return;
    setSourceType(st);
    if (submittedQuery.trim()) {
      doSearch(submittedQuery, st);
    }
  };

  return (
    <div data-testid="knowledge-page" className="max-w-4xl mx-auto px-4 py-6">
      <h1 className="text-2xl font-bold text-slate-900 dark:text-slate-100 mb-6">
        Knowledge Search
      </h1>

      {/* Search input */}
      <form onSubmit={handleSubmit} className="mb-4">
        <div className="flex gap-2">
          <input
            type="text"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder="Search documents, artifacts, incidents, work items..."
            data-testid="knowledge-search-input"
            className="flex-1 px-4 py-2 border border-slate-300 dark:border-slate-600 rounded-lg bg-white dark:bg-slate-800 text-slate-900 dark:text-slate-100 focus:outline-none focus:ring-2 focus:ring-blue-500"
            autoFocus
          />
          <button
            type="submit"
            data-testid="knowledge-search-button"
            disabled={loading || !query.trim()}
            className="px-6 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
          >
            {loading ? 'Searching...' : 'Search'}
          </button>
        </div>
      </form>

      {/* Filters */}
      <div className="mb-6">
        <SearchFilters active={sourceType} onSelect={handleFilterSelect} />
      </div>

      {/* States */}
      {loading && (
        <div data-testid="knowledge-loading" className="text-center py-12">
          <div className="inline-block animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600 mb-4" />
          <p className="text-slate-500 dark:text-slate-400">Searching...</p>
        </div>
      )}

      {error && !loading && (
        <div
          data-testid="knowledge-error"
          className="text-center py-12 text-red-600 dark:text-red-400"
        >
          {error}
        </div>
      )}

      {!loading && !error && !hasSearched && (
        <div data-testid="knowledge-empty" className="text-center py-16">
          <div className="text-5xl mb-4">🔍</div>
          <p className="text-lg text-slate-500 dark:text-slate-400">
            Search across documents, artifacts, incidents, work items, and more.
          </p>
          <p className="text-sm text-slate-400 mt-2">
            Type a query above to get started.
          </p>
        </div>
      )}

      {!loading && !error && hasSearched && results.length === 0 && (
        <div data-testid="knowledge-no-results" className="text-center py-16">
          <div className="text-4xl mb-4">📭</div>
          <p className="text-lg text-slate-500 dark:text-slate-400">
            No results found for "{submittedQuery}".
          </p>
          <p className="text-sm text-slate-400 mt-2">
            Try different keywords or remove filters.
          </p>
        </div>
      )}

      {!loading && !error && results.length > 0 && (
        <>
          <div className="text-sm text-slate-500 dark:text-slate-400 mb-4">
            {total} {total === 1 ? 'result' : 'results'} for "{submittedQuery}"
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
