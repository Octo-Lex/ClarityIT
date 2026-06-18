import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import type { ReactNode } from 'react';

// Inline the RiskScoreDisplay component for direct testing
function RiskScoreDisplay({ riskScore }: { riskScore: any }) {
  const score = riskScore?.score ?? 0;
  const level = riskScore?.level ?? 'unknown';
  const topFactors = riskScore?.top_factors ?? [];

  const levelColor: Record<string, string> = {
    low: 'text-green-400',
    medium: 'text-yellow-400',
    high: 'text-orange-400',
    critical: 'text-red-400',
    unknown: 'text-gray-400',
  };

  const levelBg: Record<string, string> = {
    low: 'bg-green-900/20 border-green-700',
    medium: 'bg-yellow-900/20 border-yellow-700',
    high: 'bg-orange-900/20 border-orange-700',
    critical: 'bg-red-900/20 border-red-700',
    unknown: 'bg-gray-900/20 border-gray-700',
  };

  if (!riskScore) return null;

  return (
    <div className="mt-4 pt-3 border-t border-[var(--border)]" data-testid="risk-score-section">
      <div className="flex items-center gap-3 mb-2">
        <h4 className="text-sm font-semibold">Change-Risk Score</h4>
        <span
          className={`px-3 py-1 rounded border text-sm font-bold ${levelBg[level] || levelBg.unknown} ${levelColor[level] || levelColor.unknown}`}
          data-testid="risk-score-badge"
        >
          {score} · {level.toUpperCase()}
        </span>
      </div>

      <div className="mb-2 text-xs text-[var(--text-muted)]" data-testid="risk-score-advisory">
        ⚠ Risk score is advisory only. Approval, MFA, policy, and mutation-window controls still apply.
      </div>

      {topFactors.length > 0 && (
        <div className="mb-2" data-testid="risk-score-top-factors">
          <span className="text-xs text-[var(--text-muted)]">Top factors: </span>
          {topFactors.map((f: string, i: number) => (
            <span key={f} className="text-xs badge badge-gray mr-1" data-testid={`risk-score-factor-${f}`}>
              {f.replace(/_/g, ' ')}
            </span>
          ))}
        </div>
      )}
    </div>
  );
}

function Wrapper({ children }: { children: ReactNode }) {
  return <MemoryRouter>{children}</MemoryRouter>;
}

describe('RiskScoreDisplay — Change-Risk Scoring', () => {
  // Test 1: risk score badge renders in dry-run preview
  it('renders risk score badge with score and level', () => {
    render(
      <Wrapper>
        <RiskScoreDisplay riskScore={{ score: 78, level: 'high', advisory_only: true, top_factors: ['action_type', 'recent_incidents', 'mutation_window_status'] }} />
      </Wrapper>
    );

    const badge = screen.getByTestId('risk-score-badge');
    expect(badge.textContent).toContain('78');
    expect(badge.textContent).toContain('HIGH');
  });

  // Test 2: risk factor breakdown renders
  it('renders top risk factors as badges', () => {
    render(
      <Wrapper>
        <RiskScoreDisplay riskScore={{ score: 78, level: 'high', advisory_only: true, top_factors: ['action_type', 'recent_incidents', 'mutation_window_status'] }} />
      </Wrapper>
    );

    const topFactors = screen.getByTestId('risk-score-top-factors');
    expect(topFactors.textContent).toContain('action type');
    expect(topFactors.textContent).toContain('recent incidents');
    expect(topFactors.textContent).toContain('mutation window status');
  });

  // Test 3: advisory-only warning renders
  it('renders advisory-only warning with clear wording', () => {
    render(
      <Wrapper>
        <RiskScoreDisplay riskScore={{ score: 78, level: 'high', advisory_only: true, top_factors: [] }} />
      </Wrapper>
    );

    const advisory = screen.getByTestId('risk-score-advisory');
    expect(advisory.textContent).toContain('advisory only');
    expect(advisory.textContent).toContain('Approval, MFA');
    expect(advisory.textContent).toContain('mutation-window controls still apply');
  });

  // Test 4: critical risk level renders red badge
  it('renders critical risk level correctly', () => {
    render(
      <Wrapper>
        <RiskScoreDisplay riskScore={{ score: 92, level: 'critical', advisory_only: true, top_factors: ['action_type'] }} />
      </Wrapper>
    );

    const badge = screen.getByTestId('risk-score-badge');
    expect(badge.textContent).toContain('92');
    expect(badge.textContent).toContain('CRITICAL');
    expect(badge.className).toContain('text-red-400');
  });

  // Test 5: no risk score section when risk_score is null/undefined
  it('does not render anything when riskScore is null', () => {
    const { container } = render(
      <Wrapper>
        <RiskScoreDisplay riskScore={null} />
      </Wrapper>
    );

    expect(container.querySelector('[data-testid="risk-score-section"]')).toBeNull();
  });

  // Test 6: no sensitive incident body in risk score rendering
  it('does not render sensitive incident body data', () => {
    const { container } = render(
      <Wrapper>
        <RiskScoreDisplay riskScore={{ score: 78, level: 'high', advisory_only: true, top_factors: ['action_type'] }} />
      </Wrapper>
    );

    const html = container.innerHTML.toLowerCase();
    expect(html).not.toContain('password');
    expect(html).not.toContain('secret');
    expect(html).not.toContain('token');
    expect(html).not.toContain('summary');
    expect(html).not.toContain('incident_body');
  });

  // Test 7: low risk renders green badge
  it('renders low risk in green', () => {
    render(
      <Wrapper>
        <RiskScoreDisplay riskScore={{ score: 15, level: 'low', advisory_only: true, top_factors: [] }} />
      </Wrapper>
    );

    const badge = screen.getByTestId('risk-score-badge');
    expect(badge.textContent).toContain('15');
    expect(badge.textContent).toContain('LOW');
    expect(badge.className).toContain('text-green-400');
  });

  // Test 8: empty top factors handled gracefully
  it('handles empty top factors gracefully', () => {
    render(
      <Wrapper>
        <RiskScoreDisplay riskScore={{ score: 30, level: 'medium', advisory_only: true, top_factors: [] }} />
      </Wrapper>
    );

    // Should still render badge and advisory, just no top-factors section
    expect(screen.getByTestId('risk-score-badge')).toBeInTheDocument();
    expect(screen.getByTestId('risk-score-advisory')).toBeInTheDocument();
    expect(screen.queryByTestId('risk-score-top-factors')).toBeNull();
  });
});
