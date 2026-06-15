import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor, fireEvent } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';

// Mock the API client
vi.mock('../api/client', () => ({
  api: {
    listArtifacts: vi.fn(),
    getArtifact: vi.fn(),
    createArtifact: vi.fn(),
    updateArtifact: vi.fn(),
    archiveArtifact: vi.fn(),
    getPresentonStatus: vi.fn(),
    generatePresentation: vi.fn(),
  },
  ApiError: class extends Error { constructor(public status: number, msg: string) { super(msg); } },
}));

import ArtifactsPage from '../features/artifacts/ArtifactsPage';
import PresentationModal from '../features/artifacts/PresentationModal';
import { api } from '../api/client';

function renderPage() {
  return render(
    <MemoryRouter>
      <ArtifactsPage />
    </MemoryRouter>
  );
}

describe('Track 2 — Team Presentations via Presenton', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  // Test 1: Generate Presentation button renders
  it('renders Generate Presentation button', async () => {
    vi.mocked(api.listArtifacts).mockResolvedValue([]);
    renderPage();
    await waitFor(() => expect(screen.getByTestId('artifacts-generate-presentation-btn')).toBeInTheDocument());
  });

  // Test 2: modal opens with required fields
  it('opens presentation modal with fields', async () => {
    vi.mocked(api.listArtifacts).mockResolvedValue([]);
    vi.mocked(api.getPresentonStatus).mockResolvedValue({ enabled: true, reachable: true, message: 'OK' });
    renderPage();
    await waitFor(() => expect(screen.getByTestId('artifacts-generate-presentation-btn')).toBeInTheDocument());
    fireEvent.click(screen.getByTestId('artifacts-generate-presentation-btn'));
    await waitFor(() => expect(screen.getByTestId('presentation-modal')).toBeInTheDocument());
    expect(screen.getByTestId('presentation-title')).toBeInTheDocument();
    expect(screen.getByTestId('presentation-content')).toBeInTheDocument();
    expect(screen.getByTestId('presentation-slides')).toBeInTheDocument();
  });

  // Test 3: status disabled state renders
  it('renders disabled banner when Presenton is disabled', async () => {
    vi.mocked(api.listArtifacts).mockResolvedValue([]);
    vi.mocked(api.getPresentonStatus).mockResolvedValue({ enabled: false, reachable: false, message: 'Presenton integration is disabled.' });
    renderPage();
    await waitFor(() => expect(screen.getByTestId('artifacts-generate-presentation-btn')).toBeInTheDocument());
    fireEvent.click(screen.getByTestId('artifacts-generate-presentation-btn'));
    await waitFor(() => expect(screen.getByTestId('presenton-disabled-banner')).toBeInTheDocument());
    expect(screen.getByTestId('presenton-disabled-banner').textContent).toContain('disabled');
  });

  // Test 4: successful generation calls API
  it('calls generatePresentation API on submit', async () => {
    vi.mocked(api.listArtifacts).mockResolvedValue([]);
    vi.mocked(api.getPresentonStatus).mockResolvedValue({ enabled: true, reachable: true, message: 'OK' });
    vi.mocked(api.generatePresentation).mockResolvedValue({ artifact_id: 'art-1', file_format: 'pptx' });
    renderPage();
    await waitFor(() => expect(screen.getByTestId('artifacts-generate-presentation-btn')).toBeInTheDocument());
    fireEvent.click(screen.getByTestId('artifacts-generate-presentation-btn'));
    await waitFor(() => expect(screen.getByTestId('presentation-generate-btn')).toBeInTheDocument());

    fireEvent.change(screen.getByTestId('presentation-title'), { target: { value: 'Test Deck' } });
    fireEvent.change(screen.getByTestId('presentation-content'), { target: { value: 'About testing' } });
    fireEvent.click(screen.getByTestId('presentation-generate-btn'));

    await waitFor(() => expect(api.generatePresentation).toHaveBeenCalledTimes(1));
  });

  // Test 5: spinner/loading state renders
  it('renders spinner during generation', async () => {
    vi.mocked(api.listArtifacts).mockResolvedValue([]);
    vi.mocked(api.getPresentonStatus).mockResolvedValue({ enabled: true, reachable: true, message: 'OK' });
    // Never resolves to keep loading state
    vi.mocked(api.generatePresentation).mockReturnValue(new Promise(() => {}));
    renderPage();
    await waitFor(() => expect(screen.getByTestId('artifacts-generate-presentation-btn')).toBeInTheDocument());
    fireEvent.click(screen.getByTestId('artifacts-generate-presentation-btn'));
    await waitFor(() => expect(screen.getByTestId('presentation-generate-btn')).toBeInTheDocument());

    fireEvent.change(screen.getByTestId('presentation-title'), { target: { value: 'Test' } });
    fireEvent.change(screen.getByTestId('presentation-content'), { target: { value: 'Content' } });
    fireEvent.click(screen.getByTestId('presentation-generate-btn'));

    await waitFor(() => expect(screen.getByTestId('presentation-spinner')).toBeInTheDocument());
    expect(screen.getByTestId('presentation-spinner').textContent).toContain('Generating');
  });

  // Test 6: generated artifact appears in list after generation
  it('refetches artifacts after successful generation', async () => {
    vi.mocked(api.listArtifacts).mockResolvedValue([]);
    vi.mocked(api.getPresentonStatus).mockResolvedValue({ enabled: true, reachable: true, message: 'OK' });
    vi.mocked(api.generatePresentation).mockResolvedValue({ artifact_id: 'art-gen', file_format: 'pptx' });
    renderPage();
    await waitFor(() => expect(screen.getByTestId('artifacts-generate-presentation-btn')).toBeInTheDocument());
    fireEvent.click(screen.getByTestId('artifacts-generate-presentation-btn'));
    await waitFor(() => expect(screen.getByTestId('presentation-generate-btn')).toBeInTheDocument());

    fireEvent.change(screen.getByTestId('presentation-title'), { target: { value: 'Generated Deck' } });
    fireEvent.change(screen.getByTestId('presentation-content'), { target: { value: 'Content' } });
    fireEvent.click(screen.getByTestId('presentation-generate-btn'));

    // After generation, listArtifacts should be called again (refetch)
    await waitFor(() => {
      expect(api.listArtifacts).toHaveBeenCalledTimes(2); // initial + refetch
    });
  });

  // Test 7: error state renders safely
  it('renders error message on generation failure', async () => {
    vi.mocked(api.listArtifacts).mockResolvedValue([]);
    vi.mocked(api.getPresentonStatus).mockResolvedValue({ enabled: true, reachable: true, message: 'OK' });
    vi.mocked(api.generatePresentation).mockRejectedValue(new Error('Generation failed'));
    renderPage();
    await waitFor(() => expect(screen.getByTestId('artifacts-generate-presentation-btn')).toBeInTheDocument());
    fireEvent.click(screen.getByTestId('artifacts-generate-presentation-btn'));
    await waitFor(() => expect(screen.getByTestId('presentation-generate-btn')).toBeInTheDocument());

    fireEvent.change(screen.getByTestId('presentation-title'), { target: { value: 'Test' } });
    fireEvent.change(screen.getByTestId('presentation-content'), { target: { value: 'Content' } });
    fireEvent.click(screen.getByTestId('presentation-generate-btn'));

    await waitFor(() => {
      expect(screen.getByTestId('presentation-modal').textContent).toContain('Failed');
    });
  });

  // Test 8: export format selector works
  it('export format selector switches between pptx and pdf', async () => {
    vi.mocked(api.listArtifacts).mockResolvedValue([]);
    vi.mocked(api.getPresentonStatus).mockResolvedValue({ enabled: true, reachable: true, message: 'OK' });
    renderPage();
    await waitFor(() => expect(screen.getByTestId('artifacts-generate-presentation-btn')).toBeInTheDocument());
    fireEvent.click(screen.getByTestId('artifacts-generate-presentation-btn'));
    await waitFor(() => expect(screen.getByTestId('presentation-format')).toBeInTheDocument());

    const select = screen.getByTestId('presentation-format') as HTMLSelectElement;
    expect(select.value).toBe('pptx');
    fireEvent.change(select, { target: { value: 'pdf' } });
    expect(select.value).toBe('pdf');
  });

  // Test 9: static template/tone selectors render
  it('renders static template and tone selectors', async () => {
    vi.mocked(api.listArtifacts).mockResolvedValue([]);
    vi.mocked(api.getPresentonStatus).mockResolvedValue({ enabled: true, reachable: true, message: 'OK' });
    renderPage();
    await waitFor(() => expect(screen.getByTestId('artifacts-generate-presentation-btn')).toBeInTheDocument());
    fireEvent.click(screen.getByTestId('artifacts-generate-presentation-btn'));
    await waitFor(() => expect(screen.getByTestId('presentation-template')).toBeInTheDocument());
    expect(screen.getByTestId('presentation-tone')).toBeInTheDocument();

    const templateSelect = screen.getByTestId('presentation-template') as HTMLSelectElement;
    expect(templateSelect.querySelectorAll('option').length).toBeGreaterThan(1);
  });

  // Test 10: no Presenton raw path rendered
  it('does not render Presenton internal paths', async () => {
    vi.mocked(api.listArtifacts).mockResolvedValue([]);
    vi.mocked(api.getPresentonStatus).mockResolvedValue({ enabled: true, reachable: true, message: 'OK' });
    vi.mocked(api.generatePresentation).mockResolvedValue({ artifact_id: 'art-1', file_format: 'pptx' });
    const { container } = renderPage();
    await waitFor(() => expect(screen.getByTestId('artifacts-generate-presentation-btn')).toBeInTheDocument());
    fireEvent.click(screen.getByTestId('artifacts-generate-presentation-btn'));
    await waitFor(() => expect(screen.getByTestId('presentation-generate-btn')).toBeInTheDocument());

    fireEvent.change(screen.getByTestId('presentation-title'), { target: { value: 'Test' } });
    fireEvent.change(screen.getByTestId('presentation-content'), { target: { value: 'Content' } });
    fireEvent.click(screen.getByTestId('presentation-generate-btn'));

    await waitFor(() => expect(api.generatePresentation).toHaveBeenCalled());

    // No internal paths should appear in the DOM
    expect(container.textContent).not.toContain('/internal/');
    expect(container.textContent).not.toContain('/tmp/');
    expect(container.textContent).not.toContain('edit_path');
  });
});
