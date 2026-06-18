import { StatusBadge } from '@/components/ui/status-badge';

type Tone = 'success' | 'warning' | 'danger' | 'info' | 'neutral';

const REASON_LABELS: Record<string, string> = {
  explicit_link: 'Linked',
  context_edge: 'Connected',
  shared_reference: 'Shared Ref',
  content_similarity: 'Similar',
  same_source_family: 'Same Type',
  recent_related: 'Recent',
};

const REASON_TONES: Record<string, Tone> = {
  explicit_link: 'info',
  context_edge: 'info',
  shared_reference: 'info',
  content_similarity: 'warning',
  same_source_family: 'neutral',
  recent_related: 'neutral',
};

export function RelatedKnowledgeReasonBadge({ reason }: { reason: string }) {
  const label = REASON_LABELS[reason] ?? reason;
  const tone = REASON_TONES[reason] ?? 'neutral';
  return (
    <StatusBadge data-testid="related-reason-badge" tone={tone}>
      {label}
    </StatusBadge>
  );
}
