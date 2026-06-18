import { ShieldAlert } from 'lucide-react';
import { StatusBadge } from '@/components/ui/status-badge';

/**
 * The A0–A5 agent autonomy ladder. A5 (full autonomy) is hardcoded-disabled
 * backend-side — it's rejected before any DB lookup and excluded by a CHECK
 * constraint. The UI keeps A5 visible but marks it unavailable to make the
 * safety boundary explicit (mirrors the backend's ESAA model).
 */
export const AUTONOMY_LEVELS = ['A0', 'A1', 'A2', 'A3', 'A4', 'A5'] as const;

export const AUTONOMY_DESCRIPTIONS: Record<string, string> = {
  A0: 'A0 — No autonomy (observe only)',
  A1: 'A1 — Suggest (propose, never act)',
  A2: 'A2 — Recommend (recommend with rationale)',
  A3: 'A3 — Supervised (act with human approval)',
  A4: 'A4 — Semi-autonomous (act, report, review)',
  A5: 'A5 — Full autonomy (disabled by policy)',
};

/** Tone for displaying an autonomy level as a badge. */
export function autonomyTone(level: string): 'success' | 'info' | 'warning' | 'danger' | 'neutral' {
  if (level === 'A0' || level === 'A1') return 'neutral';
  if (level === 'A2') return 'info';
  if (level === 'A3') return 'warning';
  if (level === 'A4') return 'danger';
  return 'danger'; // A5 — disabled
}

/** Renders an autonomy level as a status badge, with a shield icon for A4/A5. */
export function AutonomyBadge({ level }: { level: string }) {
  const tone = autonomyTone(level);
  return (
    <StatusBadge tone={tone} dot>
      {(level === 'A4' || level === 'A5') && <ShieldAlert className="mr-0.5 size-3" />}
      {level}
    </StatusBadge>
  );
}
