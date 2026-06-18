import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor, fireEvent } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';

vi.mock('../api/client', () => ({
  api: {
    listArtifacts: vi.fn(),
    getArtifact: vi.fn(),
    createArtifact: vi.fn(),
    updateArtifact: vi.fn(),
    archiveArtifact: vi.fn(),
    getPresentonStatus: vi.fn(),
    generatePresentation: vi.fn(),
    listMeetingSummaries: vi.fn(),
    getMeetingSummary: vi.fn(),
    createMeetingSummary: vi.fn(),
    updateMeetingSummary: vi.fn(),
    generateStatusReport: vi.fn(),
    listTemplates: vi.fn(),
    createTemplate: vi.fn(),
    instantiateTemplate: vi.fn(),
    getRecentArtifacts: vi.fn().mockResolvedValue([]),
    searchArtifacts: vi.fn().mockResolvedValue([]),
    getStorageSummary: vi.fn().mockResolvedValue(null),
  },
  ApiError: class extends Error { constructor(public status: number, msg: string) { super(msg); } },
}));

import ArtifactsPage from '../features/artifacts/ArtifactsPage';
import StatusReportModal from '../features/artifacts/StatusReportModal';
import { api } from '../api/client';

describe('Track 4 — Project Status Report Generator', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  // Test 1: generate status report button renders
  it('renders Status Report button on artifacts page', async () => {
    vi.mocked(api.listArtifacts).mockResolvedValue([]);
    render(
      <MemoryRouter>
        <ArtifactsPage />
      </MemoryRouter>
    );
    await waitFor(() => expect(screen.getByTestId('artifacts-status-report-btn')).toBeInTheDocument());
  });

  // Test 2: modal renders project/date/section inputs
  it('renders status report modal with inputs', () => {
    render(
      <MemoryRouter>
        <StatusReportModal onClose={() => {}} onGenerated={() => {}} />
      </MemoryRouter>
    );
    expect(screen.getByTestId('status-report-modal')).toBeInTheDocument();
    expect(screen.getByTestId('report-title')).toBeInTheDocument();
    expect(screen.getByTestId('report-period-start')).toBeInTheDocument();
    expect(screen.getByTestId('report-period-end')).toBeInTheDocument();
    expect(screen.getByTestId('report-sections')).toBeInTheDocument();
  });

  // Test 3: section checkbox toggles
  it('toggles section checkboxes', () => {
    render(
      <MemoryRouter>
        <StatusReportModal onClose={() => {}} onGenerated={() => {}} />
      </MemoryRouter>
    );
    const checkbox = screen.getByTestId('report-section-risks') as HTMLInputElement;
    expect(checkbox.checked).toBe(false);
    fireEvent.click(checkbox);
    expect(checkbox.checked).toBe(true);
    fireEvent.click(checkbox);
    expect(checkbox.checked).toBe(false);
  });

  // Test 4: generate calls API
  it('calls generateStatusReport API', async () => {
    vi.mocked(api.generateStatusReport).mockResolvedValue({
      artifact_id: 'art-1',
      content_markdown: '# Report',
    });
    render(
      <MemoryRouter>
        <StatusReportModal onClose={() => {}} onGenerated={() => {}} />
      </MemoryRouter>
    );
    fireEvent.change(screen.getByTestId('report-title'), { target: { value: 'Test Report' } });
    fireEvent.change(screen.getByTestId('report-period-start'), { target: { value: '2026-01-01' } });
    fireEvent.change(screen.getByTestId('report-period-end'), { target: { value: '2026-06-15' } });
    fireEvent.click(screen.getByTestId('report-generate-btn'));
    await waitFor(() => expect(api.generateStatusReport).toHaveBeenCalledTimes(1));
  });

  // Test 5: generated report preview appears
  it('shows generated report preview', async () => {
    vi.mocked(api.generateStatusReport).mockResolvedValue({
      artifact_id: 'art-1',
      content_markdown: '# Weekly Status\n\n## Summary\n- All good',
    });
    render(
      <MemoryRouter>
        <StatusReportModal onClose={() => {}} onGenerated={() => {}} />
      </MemoryRouter>
    );
    fireEvent.change(screen.getByTestId('report-title'), { target: { value: 'Weekly Status' } });
    fireEvent.change(screen.getByTestId('report-period-start'), { target: { value: '2026-01-01' } });
    fireEvent.change(screen.getByTestId('report-period-end'), { target: { value: '2026-06-15' } });
    fireEvent.click(screen.getByTestId('report-generate-btn'));
    await waitFor(() => expect(screen.getByTestId('report-preview')).toBeInTheDocument());
    expect(screen.getByTestId('report-preview').textContent).toContain('Weekly Status');
  });

  // Test 6: download as Markdown button renders
  it('renders download markdown button after generation', async () => {
    vi.mocked(api.generateStatusReport).mockResolvedValue({
      artifact_id: 'art-1',
      content_markdown: '# Report',
    });
    render(
      <MemoryRouter>
        <StatusReportModal onClose={() => {}} onGenerated={() => {}} />
      </MemoryRouter>
    );
    fireEvent.change(screen.getByTestId('report-title'), { target: { value: 'Test' } });
    fireEvent.change(screen.getByTestId('report-period-start'), { target: { value: '2026-01-01' } });
    fireEvent.change(screen.getByTestId('report-period-end'), { target: { value: '2026-06-15' } });
    fireEvent.click(screen.getByTestId('report-generate-btn'));
    await waitFor(() => expect(screen.getByTestId('report-download-md')).toBeInTheDocument());
  });

  // Test 7: error state renders safely
  it('renders error on generation failure', async () => {
    vi.mocked(api.generateStatusReport).mockRejectedValue(new Error('Generation failed'));
    render(
      <MemoryRouter>
        <StatusReportModal onClose={() => {}} onGenerated={() => {}} />
      </MemoryRouter>
    );
    fireEvent.change(screen.getByTestId('report-title'), { target: { value: 'Test' } });
    fireEvent.change(screen.getByTestId('report-period-start'), { target: { value: '2026-01-01' } });
    fireEvent.change(screen.getByTestId('report-period-end'), { target: { value: '2026-06-15' } });
    fireEvent.click(screen.getByTestId('report-generate-btn'));
    await waitFor(() => {
      expect(screen.getByTestId('status-report-modal').textContent).toContain('Failed');
    });
  });

  // Test 8: no Presenton/PPTX/operational buttons rendered
  it('does not render out-of-scope buttons', () => {
    const { container } = render(
      <MemoryRouter>
        <StatusReportModal onClose={() => {}} onGenerated={() => {}} />
      </MemoryRouter>
    );
    expect(container.textContent).not.toContain('Presenton');
    expect(container.textContent).not.toContain('PPTX');
    expect(container.textContent).not.toContain('Approve');
    expect(container.textContent).not.toContain('Execute');
    expect(container.textContent).not.toContain('Tool Gateway');
  });
});
