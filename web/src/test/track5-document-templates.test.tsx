import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor, fireEvent, act } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';

vi.mock('../api/client', () => ({
  api: {
    listTemplates: vi.fn(),
    createTemplate: vi.fn(),
    instantiateTemplate: vi.fn(),
  },
  ApiError: class extends Error { constructor(public status: number, msg: string) { super(msg); } },
}));

import TemplateGallery from '../features/artifacts/TemplateGallery';
import { api } from '../api/client';

const mockTemplates = [
  {
    id: 't-md-1',
    template_type: 'document',
    name: 'Architecture Walkthrough',
    description: 'Architecture doc template',
    content_markdown: '# Architecture',
    metadata: {},
    is_system: true,
    template_format: 'markdown',
  },
  {
    id: 't-doc-1',
    template_type: 'document',
    name: 'Implementation Plan (Structured)',
    description: 'Structured implementation plan',
    content_markdown: null,
    document_json: {
      schema_version: 1,
      title: 'Implementation Plan',
      document_type: 'implementation_plan',
      blocks: [
        { id: 'blk_001', type: 'heading', level: 1, text: 'Implementation Plan' },
        { id: 'blk_002', type: 'paragraph', text: 'Brief description.' },
        { id: 'blk_003', type: 'bullets', items: ['Item 1', 'Item 2'] },
      ],
    },
    metadata: { doc_type: 'implementation_plan' },
    is_system: true,
    schema_version: 1,
    template_format: 'document_json',
  },
  {
    id: 't-team-1',
    template_type: 'document',
    name: 'Team Custom Template',
    description: 'A team template',
    content_markdown: '# Team Doc',
    metadata: {},
    is_system: false,
    template_format: 'markdown',
  },
];

function renderGallery(props: Partial<Parameters<typeof TemplateGallery>[0]> = {}) {
  const defaultProps = {
    onClose: vi.fn(),
    onInstantiated: vi.fn(),
    ...props,
  };
  return render(
    <MemoryRouter>
      <TemplateGallery {...defaultProps as any} />
    </MemoryRouter>
  );
}

beforeEach(() => {
  vi.clearAllMocks();
  (api.listTemplates as any).mockResolvedValue(mockTemplates);
  (api.instantiateTemplate as any).mockResolvedValue({ artifact_id: 'new-art-1' });
  (api.createTemplate as any).mockResolvedValue({ id: 'new-tmpl' });
});

describe('TemplateGallery (Track 5)', () => {
  it('1. renders document templates', async () => {
    renderGallery();
    await waitFor(() => expect(screen.getByTestId('template-list')).toBeTruthy());
    expect(screen.getByText('Implementation Plan (Structured)')).toBeTruthy();
  });

  it('2. format badge renders', async () => {
    renderGallery();
    await waitFor(() => expect(screen.getByTestId('template-list')).toBeTruthy());
    expect(screen.getByTestId('template-badge-md-t-md-1')).toBeTruthy();
    expect(screen.getByTestId('template-badge-doc-t-doc-1')).toBeTruthy();
  });

  it('3. system/team badge renders', async () => {
    renderGallery();
    await waitFor(() => expect(screen.getByTestId('template-list')).toBeTruthy());
    expect(screen.getByTestId('template-badge-system-t-doc-1')).toBeTruthy();
    expect(screen.getByTestId('template-badge-team-t-team-1')).toBeTruthy();
  });

  it('4. type filter works', async () => {
    renderGallery();
    await waitFor(() => expect(screen.getByTestId('template-list')).toBeTruthy());
    const filter = screen.getByTestId('template-filter') as HTMLSelectElement;
    fireEvent.change(filter, { target: { value: 'document' } });
    expect(api.listTemplates).toHaveBeenCalledWith('document', undefined);
  });

  it('5. format filter works', async () => {
    renderGallery();
    await waitFor(() => expect(screen.getByTestId('template-list')).toBeTruthy());
    const filter = screen.getByTestId('template-format-filter') as HTMLSelectElement;
    fireEvent.change(filter, { target: { value: 'document_json' } });
    expect(api.listTemplates).toHaveBeenCalledWith(undefined, 'document_json');
  });

  it('6. document_json preview renders blocks not raw JSON', async () => {
    renderGallery();
    await waitFor(() => expect(screen.getByTestId('template-list')).toBeTruthy());
    // Click on the document_json template card
    fireEvent.click(screen.getByTestId('template-card-t-doc-1'));
    // Should render block preview, not raw JSON
    await waitFor(() => expect(screen.getByTestId('template-preview-doc')).toBeTruthy());
    // The heading text should be rendered as text, not as JSON
    expect(screen.getByText('Implementation Plan')).toBeTruthy();
    expect(screen.getByText('Brief description.')).toBeTruthy();
    // Should NOT see raw JSON
    expect(screen.queryByTestId('template-preview-content')).toBeNull();
  });

  it('7. Use Template calls instantiate API', async () => {
    renderGallery();
    await waitFor(() => expect(screen.getByTestId('template-list')).toBeTruthy());
    fireEvent.click(screen.getByTestId('template-card-t-doc-1'));
    await waitFor(() => expect(screen.getByTestId('template-use-btn')).toBeTruthy());
    await act(async () => {
      fireEvent.click(screen.getByTestId('template-use-btn'));
    });
    expect(api.instantiateTemplate).toHaveBeenCalled();
  });

  it('8. document_json template instantiates and calls onInstantiated', async () => {
    const onInstantiated = vi.fn();
    renderGallery({ onInstantiated });
    await waitFor(() => expect(screen.getByTestId('template-list')).toBeTruthy());
    fireEvent.click(screen.getByTestId('template-card-t-doc-1'));
    await waitFor(() => expect(screen.getByTestId('template-use-btn')).toBeTruthy());
    await act(async () => {
      fireEvent.click(screen.getByTestId('template-use-btn'));
    });
    await waitFor(() => expect(onInstantiated).toHaveBeenCalledWith('new-art-1'));
  });

  it('9. custom document template form renders', async () => {
    renderGallery();
    await waitFor(() => expect(screen.getByTestId('template-list')).toBeTruthy());
    fireEvent.click(screen.getByTestId('template-create-btn'));
    expect(screen.getByTestId('template-create-form')).toBeTruthy();
    expect(screen.getByTestId('template-form-format')).toBeTruthy();
  });

  it('10. custom document template submit calls API', async () => {
    renderGallery();
    await waitFor(() => expect(screen.getByTestId('template-list')).toBeTruthy());
    fireEvent.click(screen.getByTestId('template-create-btn'));
    fireEvent.change(screen.getByTestId('template-form-name'), { target: { value: 'My Template' } });
    fireEvent.change(screen.getByTestId('template-form-format'), { target: { value: 'document_json' } });
    fireEvent.change(screen.getByTestId('template-form-doc-json'), {
      target: { value: '{"schema_version":1,"title":"T","document_type":"general_document","blocks":[{"id":"b1","type":"paragraph","text":"hi"}]}' },
    });
    await act(async () => {
      fireEvent.click(screen.getByTestId('template-form-save'));
    });
    expect(api.createTemplate).toHaveBeenCalled();
    const call = (api.createTemplate as any).mock.calls[0][0];
    expect(call.template_format).toBe('document_json');
  });

  it('11. empty/error states render safely', async () => {
    (api.listTemplates as any).mockRejectedValue(new Error('Network error'));
    renderGallery();
    await waitFor(() => expect(screen.getByTestId('template-error')).toBeTruthy());
    expect(screen.getByTestId('template-error').textContent).toContain('Failed');
  });

  it('12. no agent/export/version/share/approval/execute buttons', async () => {
    renderGallery();
    await waitFor(() => expect(screen.getByTestId('template-list')).toBeTruthy());
    expect(screen.queryByTestId('export-docx')).toBeNull();
    expect(screen.queryByTestId('export-pdf')).toBeNull();
    expect(screen.queryByTestId('version-history')).toBeNull();
    expect(screen.queryByTestId('share-button')).toBeNull();
    expect(screen.queryByTestId('approve-button')).toBeNull();
    expect(screen.queryByTestId('execute-button')).toBeNull();
    expect(screen.queryByTestId('generate-document')).toBeNull();
  });
});
