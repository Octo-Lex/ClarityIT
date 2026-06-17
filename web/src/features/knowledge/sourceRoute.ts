/**
 * Maps a knowledge search result to a navigable frontend route.
 * Returns null if no safe route is known (caller should use fallback).
 */

export function getSourceRoute(
  teamId: string | undefined,
  sourceType: string,
  sourceId: string
): string | null {
  const tid = teamId ?? '';
  switch (sourceType) {
    case 'clarity_document':
      return `/teams/${tid}/artifacts/documents/${sourceId}`;
    case 'artifact':
      return `/artifacts`;
    case 'meeting_summary':
      return `/artifacts`;
    case 'status_report':
      return `/artifacts`;
    case 'presentation':
      return `/artifacts`;
    case 'template':
      return `/artifacts`;
    case 'work_item':
      return `/objects/${sourceId}`;
    case 'incident':
      return `/incidents/${sourceId}`;
    case 'asset':
      return `/objects/${sourceId}`;
    case 'remediation':
      return `/incidents`;
    case 'approval':
      return `/admin/approvals`;
    case 'context_node':
      return null; // fallback
    case 'project':
      return null; // not yet implemented
    default:
      return null;
  }
}
