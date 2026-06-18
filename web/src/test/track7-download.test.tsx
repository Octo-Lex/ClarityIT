import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor, fireEvent } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';

vi.mock('../api/client', () => ({
  api: {
    getArtifact: vi.fn(),
    updateArtifact: vi.fn(),
    archiveArtifact: vi.fn(),
    downloadArtifact: vi.fn(),
    exportArtifactUrl: vi.fn((id: string, fmt: string) => `/api/teams/t/artifacts/${id}/export/${fmt}`),
  },
  ApiError: class extends Error { constructor(public status: number, msg: string) { super(msg); } },
}));

import ArtifactEditor from '../features/artifacts/ArtifactEditor';
import { api } from '../api/client';

describe('Track 7 — Download and Export', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  // Test 1: Download button renders for file artifact
  it('renders Download button for file-backed artifact', async () => {
    vi.mocked(api.getArtifact).mockResolvedValue({
      id: 'art-1', artifact_type: 'presentation', title: 'My Deck', status: 'draft',
      content_markdown: '', storage_object_id: 'so-1', file_format: 'pptx',
    });
    render(<MemoryRouter><ArtifactEditor mode="edit" artifactId="art-1" onClose={() => {}} /></MemoryRouter>);
    await waitFor(() => expect(screen.getByTestId('editor-download')).toBeInTheDocument());
  });

  // Test 2: Download button hidden for inline artifact
  it('does not render Download button for inline artifact', async () => {
    vi.mocked(api.getArtifact).mockResolvedValue({
      id: 'art-2', artifact_type: 'document', title: 'My Doc', status: 'draft',
      content_markdown: 'Hello', storage_object_id: null,
    });
    render(<MemoryRouter><ArtifactEditor mode="edit" artifactId="art-2" onClose={() => {}} /></MemoryRouter>);
    await waitFor(() => expect(screen.getByTestId('editor-actions')).toBeInTheDocument());
    expect(screen.queryByTestId('editor-download')).not.toBeInTheDocument();
  });

  // Test 3: Export Markdown renders for inline artifact
  it('renders Export Markdown for inline artifact', async () => {
    vi.mocked(api.getArtifact).mockResolvedValue({
      id: 'art-3', artifact_type: 'document', title: 'Doc', status: 'draft',
      content_markdown: '# Hello', storage_object_id: null,
    });
    render(<MemoryRouter><ArtifactEditor mode="edit" artifactId="art-3" onClose={() => {}} /></MemoryRouter>);
    await waitFor(() => expect(screen.getByTestId('editor-export-md')).toBeInTheDocument());
  });

  // Test 4: Export PDF renders for inline artifact
  it('renders Export PDF for inline artifact', async () => {
    vi.mocked(api.getArtifact).mockResolvedValue({
      id: 'art-4', artifact_type: 'document', title: 'Doc', status: 'draft',
      content_markdown: '# Hello', storage_object_id: null,
    });
    render(<MemoryRouter><ArtifactEditor mode="edit" artifactId="art-4" onClose={() => {}} /></MemoryRouter>);
    await waitFor(() => expect(screen.getByTestId('editor-export-pdf')).toBeInTheDocument());
  });

  // Test 5: Copy Markdown renders when content exists
  it('renders Copy Markdown when content exists', async () => {
    vi.mocked(api.getArtifact).mockResolvedValue({
      id: 'art-5', artifact_type: 'document', title: 'Doc', status: 'draft',
      content_markdown: '# Content', storage_object_id: null,
    });
    render(<MemoryRouter><ArtifactEditor mode="edit" artifactId="art-5" onClose={() => {}} /></MemoryRouter>);
    await waitFor(() => expect(screen.getByTestId('editor-copy-md')).toBeInTheDocument());
  });

  // Test 6: Copy Markdown hidden when no content
  it('does not render Copy Markdown when no content', async () => {
    vi.mocked(api.getArtifact).mockResolvedValue({
      id: 'art-6', artifact_type: 'presentation', title: 'Empty', status: 'draft',
      content_markdown: '', storage_object_id: 'so-1',
    });
    render(<MemoryRouter><ArtifactEditor mode="edit" artifactId="art-6" onClose={() => {}} /></MemoryRouter>);
    await waitFor(() => expect(screen.getByTestId('editor-actions')).toBeInTheDocument());
    expect(screen.queryByTestId('editor-copy-md')).not.toBeInTheDocument();
  });

  // Test 7: 15-minute expiry note renders after download
  it('renders 15-minute expiry note after download', async () => {
    vi.mocked(api.getArtifact).mockResolvedValue({
      id: 'art-7', artifact_type: 'presentation', title: 'Deck', status: 'draft',
      content_markdown: '', storage_object_id: 'so-1',
    });
    vi.mocked(api.downloadArtifact).mockResolvedValue({ download_url: 'https://minio/file', expires_in_seconds: 900 });
    render(<MemoryRouter><ArtifactEditor mode="edit" artifactId="art-7" onClose={() => {}} /></MemoryRouter>);
    await waitFor(() => expect(screen.getByTestId('editor-download')).toBeInTheDocument());
    fireEvent.click(screen.getByTestId('editor-download'));
    await waitFor(() => expect(screen.getByTestId('editor-download-note')).toBeInTheDocument());
    expect(screen.getByTestId('editor-download-note').textContent).toContain('15 minutes');
  });

  // Test 8: Error state renders safely
  it('renders error message on download failure', async () => {
    vi.mocked(api.getArtifact).mockResolvedValue({
      id: 'art-8', artifact_type: 'presentation', title: 'Deck', status: 'draft',
      content_markdown: '', storage_object_id: 'so-1',
    });
    vi.mocked(api.downloadArtifact).mockRejectedValue(new (class extends Error { constructor() { super('Download failed'); } })());
    render(<MemoryRouter><ArtifactEditor mode="edit" artifactId="art-8" onClose={() => {}} /></MemoryRouter>);
    await waitFor(() => expect(screen.getByTestId('editor-download')).toBeInTheDocument());
    fireEvent.click(screen.getByTestId('editor-download'));
    await waitFor(() => expect(screen.getByTestId('editor-download-error')).toBeInTheDocument());
  });

  // Test 9: No public/external share/email buttons
  it('does not render share/email buttons', async () => {
    vi.mocked(api.getArtifact).mockResolvedValue({
      id: 'art-9', artifact_type: 'document', title: 'Doc', status: 'draft',
      content_markdown: 'content', storage_object_id: null,
    });
    const { container } = render(<MemoryRouter><ArtifactEditor mode="edit" artifactId="art-9" onClose={() => {}} /></MemoryRouter>);
    await waitFor(() => expect(screen.getByTestId('editor-actions')).toBeInTheDocument());
    expect(container.textContent).not.toContain('Share');
    expect(container.textContent).not.toContain('Email');
    expect(container.textContent).not.toContain('Public');
  });

  // Test 10: No raw bucket/object key rendered
  it('does not render raw storage identifiers', async () => {
    vi.mocked(api.getArtifact).mockResolvedValue({
      id: 'art-10', artifact_type: 'presentation', title: 'Deck', status: 'draft',
      content_markdown: '', storage_object_id: 'so-1',
    });
    vi.mocked(api.downloadArtifact).mockResolvedValue({ download_url: 'https://minio/file', expires_in_seconds: 900 });
    const { container } = render(<MemoryRouter><ArtifactEditor mode="edit" artifactId="art-10" onClose={() => {}} /></MemoryRouter>);
    await waitFor(() => expect(screen.getByTestId('editor-download')).toBeInTheDocument());
    fireEvent.click(screen.getByTestId('editor-download'));
    await waitFor(() => expect(api.downloadArtifact).toHaveBeenCalled());
    expect(container.textContent).not.toContain('bucket');
    expect(container.textContent).not.toContain('object_key');
  });
});
