/**
 * Renders a ts_headline snippet. PostgreSQL ts_headline wraps matched terms
 * in <start>dword<end> tags by default. We render via dangerouslySetInnerHTML
 * after converting them to <mark> tags styled with the warning token.
 */
export function KnowledgeSnippet({ snippet }: { snippet: string }) {
  const html = snippet
    .replace(/<start>/g, '<mark class="rounded bg-warning/30 px-0.5">')
    .replace(/<end>/g, '</mark>');

  return (
    <p
      data-testid="knowledge-snippet"
      className="line-clamp-3 text-sm text-muted-foreground"
      dangerouslySetInnerHTML={{ __html: html }}
    />
  );
}
