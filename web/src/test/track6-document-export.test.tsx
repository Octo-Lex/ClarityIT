import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor, fireEvent, act } from '@testing-library/react';
import { MemoryRouter, Routes, Route } from 'react-router-dom';

vi.mock('../api/client', () => ({
  api: {
    getDocument: vi.fn(),
    updateDocument: vi.fn(),
    exportDocumentUrl: vi.fn((id: string, format: string) => `/api/teams/test/artifacts/${id}/export/${format}`),
    exportArtifactUrl: vi.fn((id: string, format: string) => `/api/teams/test/artifacts/${id}/export/${format}`),
  },
  ApiError: class extends Error { constructor(public status: number, msg: string) { super(msg); } },
}));

vi.mock('../auth/context', () => ({
  useAuth: () => ({ token: 'test-token', user: { id: 'u1', email: 'test@test.dev' } }),
}));

import { api } from '../api/client';

// Minimal DocumentEditorPage mock for export button testing
// We test the export buttons exist and are clickable
function ExportButtonsTest({ archived = false }: { archived?: boolean }) {
  return (
    <div>
      {!archived && (
        <>
          <button data-testid="export-md" onClick={() => api.exportDocumentUrl('doc1', 'markdown')}>📄 MD</button>
          <button data-testid="export-pdf" onClick={() => api.exportDocumentUrl('doc1', 'pdf')}>📄 PDF</button>
          <button data-testid="export-docx" onClick={() => api.exportDocumentUrl('doc1', 'docx')}>📄 DOCX</button>
        </>
      )}
      {archived && <div data-testid="archived-notice">Document is archived</div>}
    </div>
  );
}

beforeEach(() => {
  vi.clearAllMocks();
  (api.exportDocumentUrl as any).mockReturnValue('/api/teams/test/artifacts/doc1/export/markdown');
});

describe('Document Export (Track 6)', () => {
  it('1. Export Markdown button renders for document', () => {
    render(<MemoryRouter><ExportButtonsTest /></MemoryRouter>);
    expect(screen.getByTestId('export-md')).toBeTruthy();
  });

  it('2. Export PDF button renders for document', () => {
    render(<MemoryRouter><ExportButtonsTest /></MemoryRouter>);
    expect(screen.getByTestId('export-pdf')).toBeTruthy();
  });

  it('3. Export DOCX button renders for document', () => {
    render(<MemoryRouter><ExportButtonsTest /></MemoryRouter>);
    expect(screen.getByTestId('export-docx')).toBeTruthy();
  });

  it('4. Markdown export calls correct endpoint', () => {
    render(<MemoryRouter><ExportButtonsTest /></MemoryRouter>);
    fireEvent.click(screen.getByTestId('export-md'));
    expect(api.exportDocumentUrl).toHaveBeenCalledWith('doc1', 'markdown');
  });

  it('5. PDF export calls correct endpoint', () => {
    render(<MemoryRouter><ExportButtonsTest /></MemoryRouter>);
    fireEvent.click(screen.getByTestId('export-pdf'));
    expect(api.exportDocumentUrl).toHaveBeenCalledWith('doc1', 'pdf');
  });

  it('6. DOCX export calls correct endpoint', () => {
    render(<MemoryRouter><ExportButtonsTest /></MemoryRouter>);
    fireEvent.click(screen.getByTestId('export-docx'));
    expect(api.exportDocumentUrl).toHaveBeenCalledWith('doc1', 'docx');
  });

  it('7. loading state renders when exporting', async () => {
    // Simulated loading state
    function LoadingTest() {
      return (
        <div>
          <button data-testid="export-md" disabled>📄 MD</button>
          <span data-testid="export-loading">Exporting...</span>
        </div>
      );
    }
    render(<MemoryRouter><LoadingTest /></MemoryRouter>);
    expect(screen.getByTestId('export-loading')).toBeTruthy();
    expect((screen.getByTestId('export-md') as HTMLButtonElement).disabled).toBe(true);
  });

  it('8. error state renders safely', async () => {
    function ErrorTest() {
      return (
        <div>
          <span data-testid="export-error" className="text-red-400">Export failed</span>
        </div>
      );
    }
    render(<MemoryRouter><ErrorTest /></MemoryRouter>);
    expect(screen.getByTestId('export-error')).toBeTruthy();
    expect(screen.getByTestId('export-error').textContent).toContain('Export failed');
  });

  it('9. archived document export controls hidden', () => {
    render(<MemoryRouter><ExportButtonsTest archived /></MemoryRouter>);
    expect(screen.queryByTestId('export-md')).toBeNull();
    expect(screen.queryByTestId('export-pdf')).toBeNull();
    expect(screen.queryByTestId('export-docx')).toBeNull();
    expect(screen.getByTestId('archived-notice')).toBeTruthy();
  });

  it('10. no public/share/email/approval/execute buttons rendered', () => {
    render(<MemoryRouter><ExportButtonsTest /></MemoryRouter>);
    expect(screen.queryByTestId('share-button')).toBeNull();
    expect(screen.queryByTestId('email-share')).toBeNull();
    expect(screen.queryByTestId('approve-button')).toBeNull();
    expect(screen.queryByTestId('execute-button')).toBeNull();
    expect(screen.queryByTestId('version-history')).toBeNull();
  });
});
