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
  },
  ApiError: class extends Error { constructor(public status: number, msg: string) { super(msg); } },
}));

import ArtifactsPage from '../features/artifacts/ArtifactsPage';
import TemplateGallery from '../features/artifacts/TemplateGallery';
import { api } from '../api/client';

const mockTemplates = [
  { id: 'sys-1', template_type: 'status_report', name: 'Weekly Status Report', description: 'Standard template', content_markdown: '# Weekly Status', is_system: true },
  { id: 'team-1', template_type: 'document', name: 'Team Doc', description: 'Custom', content_markdown: '# Team Doc', is_system: false },
];

describe('Track 5 — Template Library', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  // Test 1: template gallery renders
  it('renders template gallery', () => {
    vi.mocked(api.listTemplates).mockResolvedValue(mockTemplates);
    render(
      <MemoryRouter>
        <TemplateGallery onClose={() => {}} onInstantiated={() => {}} />
      </MemoryRouter>
    );
    expect(screen.getByTestId('template-gallery')).toBeInTheDocument();
  });

  // Test 2: system template card renders
  it('renders system template card with badge', async () => {
    vi.mocked(api.listTemplates).mockResolvedValue(mockTemplates);
    render(
      <MemoryRouter>
        <TemplateGallery onClose={() => {}} onInstantiated={() => {}} />
      </MemoryRouter>
    );
    await waitFor(() => expect(screen.getByTestId('template-card-sys-1')).toBeInTheDocument());
    expect(screen.getByTestId('template-badge-system-sys-1')).toBeInTheDocument();
  });

  // Test 3: team template card renders
  it('renders team template card with badge', async () => {
    vi.mocked(api.listTemplates).mockResolvedValue(mockTemplates);
    render(
      <MemoryRouter>
        <TemplateGallery onClose={() => {}} onInstantiated={() => {}} />
      </MemoryRouter>
    );
    await waitFor(() => expect(screen.getByTestId('template-card-team-1')).toBeInTheDocument());
    expect(screen.getByTestId('template-badge-team-team-1')).toBeInTheDocument();
  });

  // Test 4: template type filter works
  it('calls listTemplates with type filter', async () => {
    vi.mocked(api.listTemplates).mockResolvedValue([mockTemplates[0]]);
    render(
      <MemoryRouter>
        <TemplateGallery onClose={() => {}} onInstantiated={() => {}} />
      </MemoryRouter>
    );
    await waitFor(() => expect(screen.getByTestId('template-filter')).toBeInTheDocument());
    fireEvent.change(screen.getByTestId('template-filter'), { target: { value: 'status_report' } });
    expect(api.listTemplates).toHaveBeenCalledWith('status_report');
  });

  // Test 5: preview content renders
  it('renders preview when card clicked', async () => {
    vi.mocked(api.listTemplates).mockResolvedValue(mockTemplates);
    render(
      <MemoryRouter>
        <TemplateGallery onClose={() => {}} onInstantiated={() => {}} />
      </MemoryRouter>
    );
    await waitFor(() => expect(screen.getByTestId('template-card-sys-1')).toBeInTheDocument());
    fireEvent.click(screen.getByTestId('template-card-sys-1'));
    await waitFor(() => expect(screen.getByTestId('template-preview-content')).toBeInTheDocument());
    expect(screen.getByTestId('template-preview-content').textContent).toContain('Weekly Status');
  });

  // Test 6: Use Template calls instantiate API
  it('calls instantiateTemplate on Use Template', async () => {
    vi.mocked(api.listTemplates).mockResolvedValue(mockTemplates);
    vi.mocked(api.instantiateTemplate).mockResolvedValue({ artifact_id: 'art-1' });
    render(
      <MemoryRouter>
        <TemplateGallery onClose={() => {}} onInstantiated={() => {}} />
      </MemoryRouter>
    );
    await waitFor(() => expect(screen.getByTestId('template-card-sys-1')).toBeInTheDocument());
    fireEvent.click(screen.getByTestId('template-card-sys-1'));
    await waitFor(() => expect(screen.getByTestId('template-use-btn')).toBeInTheDocument());
    fireEvent.click(screen.getByTestId('template-use-btn'));
    await waitFor(() => expect(api.instantiateTemplate).toHaveBeenCalledTimes(1));
  });

  // Test 7: created artifact triggers refetch (onInstantiated callback)
  it('calls onInstantiated after instantiation', async () => {
    const onInstantiated = vi.fn();
    vi.mocked(api.listTemplates).mockResolvedValue(mockTemplates);
    vi.mocked(api.instantiateTemplate).mockResolvedValue({ artifact_id: 'art-1' });
    render(
      <MemoryRouter>
        <TemplateGallery onClose={() => {}} onInstantiated={onInstantiated} />
      </MemoryRouter>
    );
    await waitFor(() => expect(screen.getByTestId('template-card-sys-1')).toBeInTheDocument());
    fireEvent.click(screen.getByTestId('template-card-sys-1'));
    await waitFor(() => expect(screen.getByTestId('template-use-btn')).toBeInTheDocument());
    fireEvent.click(screen.getByTestId('template-use-btn'));
    await waitFor(() => expect(onInstantiated).toHaveBeenCalledWith('art-1'));
  });

  // Test 8: custom template creation form renders
  it('renders custom template form on button click', async () => {
    vi.mocked(api.listTemplates).mockResolvedValue(mockTemplates);
    render(
      <MemoryRouter>
        <TemplateGallery onClose={() => {}} onInstantiated={() => {}} />
      </MemoryRouter>
    );
    await waitFor(() => expect(screen.getByTestId('template-create-btn')).toBeInTheDocument());
    fireEvent.click(screen.getByTestId('template-create-btn'));
    expect(screen.getByTestId('template-create-form')).toBeInTheDocument();
    expect(screen.getByTestId('template-form-name')).toBeInTheDocument();
    expect(screen.getByTestId('template-form-content')).toBeInTheDocument();
  });

  // Test 9: empty state renders
  it('renders empty state when no templates', async () => {
    vi.mocked(api.listTemplates).mockResolvedValue([]);
    render(
      <MemoryRouter>
        <TemplateGallery onClose={() => {}} onInstantiated={() => {}} />
      </MemoryRouter>
    );
    await waitFor(() => expect(screen.getByTestId('template-empty')).toBeInTheDocument());
  });

  // Test 10: no out-of-scope buttons rendered
  it('does not render out-of-scope buttons', async () => {
    vi.mocked(api.listTemplates).mockResolvedValue(mockTemplates);
    const { container } = render(
      <MemoryRouter>
        <TemplateGallery onClose={() => {}} onInstantiated={() => {}} />
      </MemoryRouter>
    );
    expect(container.textContent).not.toContain('Presenton');
    expect(container.textContent).not.toContain('Marketplace');
    expect(container.textContent).not.toContain('Approve');
    expect(container.textContent).not.toContain('Execute');
  });

  // Test 11: templates button renders on artifacts page
  it('renders Templates button on artifacts page', async () => {
    vi.mocked(api.listArtifacts).mockResolvedValue([]);
    render(
      <MemoryRouter>
        <ArtifactsPage />
      </MemoryRouter>
    );
    await waitFor(() => expect(screen.getByTestId('artifacts-templates-btn')).toBeInTheDocument());
  });
});
