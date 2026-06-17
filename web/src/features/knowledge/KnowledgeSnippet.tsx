import React from 'react';

/**
 * Renders a ts_headline snippet. PostgreSQL ts_headline wraps matched terms
 * in <start>dword<end> tags by default. We render via dangerouslySetInnerHTML
 * after converting them to <mark> tags.
 */
export function KnowledgeSnippet({ snippet }: { snippet: string }) {
  const html = snippet
    .replace(/<start>/g, '<mark class="bg-yellow-200 dark:bg-yellow-700 rounded px-0.5">')
    .replace(/<end>/g, '</mark>');

  return (
    <p
      data-testid="knowledge-snippet"
      className="text-sm text-slate-600 dark:text-slate-400 line-clamp-3"
      dangerouslySetInnerHTML={{ __html: html }}
    />
  );
}
