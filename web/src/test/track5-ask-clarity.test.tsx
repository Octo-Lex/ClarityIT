import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor, fireEvent } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';

// Mock the API client
vi.mock('../api/client', () => ({
  api: {
    askClarity: vi.fn(),
  },
  ApiError: class extends Error {
    constructor(public status: number, msg: string) { super(msg); }
  },
}));

import { AskClarityPanel } from '../features/knowledge/AskClarityPanel';
import { api } from '../api/client';

function renderPanel() {
  return render(
    <MemoryRouter>
      <AskClarityPanel />
    </MemoryRouter>
  );
}

describe('AskClarityPanel — Ask Clarity Knowledge Q&A', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  // Test 1: Panel renders
  it('renders the ask clarity panel', () => {
    renderPanel();
    expect(screen.getByTestId('ask-clarity-panel')).toBeInTheDocument();
  });

  // Test 2: Question input updates
  it('updates question input', () => {
    renderPanel();
    const input = screen.getByTestId('ask-clarity-input') as HTMLTextAreaElement;
    fireEvent.change(input, { target: { value: 'What is our backup policy?' } });
    expect(input.value).toBe('What is our backup policy?');
  });

  // Test 3: Ask button disabled for too-short question
  it('disables ask button for short questions', () => {
    renderPanel();
    const button = screen.getByTestId('ask-clarity-button') as HTMLButtonElement;
    expect(button.disabled).toBe(true);

    fireEvent.change(screen.getByTestId('ask-clarity-input'), { target: { value: 'hi' } });
    expect(button.disabled).toBe(true);
  });

  // Test 4: Successful answer renders
  it('renders answer on success', async () => {
    vi.mocked(api.askClarity).mockResolvedValue({
      answer: 'Backups run daily at 2am.',
      sources: [],
      confidence: 'high',
      missing_info: [],
    });
    renderPanel();

    fireEvent.change(screen.getByTestId('ask-clarity-input'), { target: { value: 'What is the backup schedule?' } });
    fireEvent.click(screen.getByTestId('ask-clarity-button'));

    await waitFor(() => {
      expect(screen.getByTestId('ask-clarity-answer')).toBeInTheDocument();
    });
    expect(screen.getByText(/Backups run daily/)).toBeInTheDocument();
  });

  // Test 5: Source cards render
  it('renders source cards with answer', async () => {
    vi.mocked(api.askClarity).mockResolvedValue({
      answer: 'According to the runbook.',
      sources: [{
        source_type: 'clarity_document',
        source_id: 'doc-1',
        knowledge_item_id: 'ki-1',
        chunk_id: 'ch-1',
        title: 'Backup Runbook',
        snippet: 'Daily verification process',
      }],
      confidence: 'medium',
      missing_info: [],
    });
    renderPanel();

    fireEvent.change(screen.getByTestId('ask-clarity-input'), { target: { value: 'What is the backup process?' } });
    fireEvent.click(screen.getByTestId('ask-clarity-button'));

    await waitFor(() => {
      expect(screen.getByTestId('ask-clarity-source-card')).toBeInTheDocument();
    });
  });

  // Test 6: Missing info renders
  it('renders missing info when present', async () => {
    vi.mocked(api.askClarity).mockResolvedValue({
      answer: 'Limited info available.',
      sources: [],
      confidence: 'low',
      missing_info: ['No matching indexed knowledge was found.'],
    });
    renderPanel();

    fireEvent.change(screen.getByTestId('ask-clarity-input'), { target: { value: 'What is the backup process?' } });
    fireEvent.click(screen.getByTestId('ask-clarity-button'));

    await waitFor(() => {
      expect(screen.getByTestId('ask-missing-info')).toBeInTheDocument();
    });
  });

  // Test 7: No sources state renders
  it('renders no-sources state', async () => {
    vi.mocked(api.askClarity).mockResolvedValue({
      answer: 'Not enough knowledge to answer.',
      sources: [],
      confidence: 'low',
      missing_info: ['No matching indexed knowledge was found.'],
    });
    renderPanel();

    fireEvent.change(screen.getByTestId('ask-clarity-input'), { target: { value: 'What is the backup process?' } });
    fireEvent.click(screen.getByTestId('ask-clarity-button'));

    await waitFor(() => {
      expect(screen.getByTestId('ask-no-sources')).toBeInTheDocument();
    });
  });

  // Test 8: Loading state renders
  it('renders loading state', async () => {
    let resolveFn: (v: any) => void;
    vi.mocked(api.askClarity).mockReturnValue(
      new Promise((resolve) => { resolveFn = resolve; })
    );
    renderPanel();

    fireEvent.change(screen.getByTestId('ask-clarity-input'), { target: { value: 'What is the backup?' } });
    fireEvent.click(screen.getByTestId('ask-clarity-button'));

    await waitFor(() => {
      expect(screen.getByTestId('ask-clarity-loading')).toBeInTheDocument();
    });
    resolveFn!({ answer: '', sources: [], confidence: 'low', missing_info: [] });
  });

  // Test 9: Error state renders safely
  it('renders safe error state on failure', async () => {
    vi.mocked(api.askClarity).mockRejectedValue(new Error('Internal error'));
    renderPanel();

    fireEvent.change(screen.getByTestId('ask-clarity-input'), { target: { value: 'What is the backup?' } });
    fireEvent.click(screen.getByTestId('ask-clarity-button'));

    await waitFor(() => {
      const err = screen.getByTestId('ask-clarity-error');
      expect(err).toBeInTheDocument();
      expect(err.textContent).not.toContain('Internal error');
    });
  });

  // Test 10: Source type filter sent with request
  it('sends source type filters with request', async () => {
    vi.mocked(api.askClarity).mockResolvedValue({
      answer: 'Answer',
      sources: [],
      confidence: 'low',
      missing_info: [],
    });
    renderPanel();

    // Select a filter
    fireEvent.click(screen.getByTestId('ask-filter-incident'));

    fireEvent.change(screen.getByTestId('ask-clarity-input'), { target: { value: 'What is the backup?' } });
    fireEvent.click(screen.getByTestId('ask-clarity-button'));

    await waitFor(() => {
      expect(api.askClarity).toHaveBeenCalledWith(
        'What is the backup?',
        ['incident'],
        8
      );
    });
  });

  // Test 11: Clear/reset works
  it('clears input and response on clear', async () => {
    vi.mocked(api.askClarity).mockResolvedValue({
      answer: 'Answer here',
      sources: [],
      confidence: 'high',
      missing_info: [],
    });
    renderPanel();

    const input = screen.getByTestId('ask-clarity-input') as HTMLTextAreaElement;
    fireEvent.change(input, { target: { value: 'What is the backup?' } });
    fireEvent.click(screen.getByTestId('ask-clarity-button'));

    await waitFor(() => {
      expect(screen.getByTestId('ask-clarity-answer')).toBeInTheDocument();
    });

    fireEvent.click(screen.getByTestId('ask-clarity-clear'));

    expect(input.value).toBe('');
    expect(screen.queryByTestId('ask-clarity-answer')).not.toBeInTheDocument();
  });

  // Test 12: No chain-of-thought rendered
  it('does not render chain-of-thought', async () => {
    vi.mocked(api.askClarity).mockResolvedValue({
      answer: 'Answer',
      sources: [],
      confidence: 'high',
      missing_info: [],
    });
    renderPanel();

    fireEvent.change(screen.getByTestId('ask-clarity-input'), { target: { value: 'What is the backup?' } });
    fireEvent.click(screen.getByTestId('ask-clarity-button'));

    await waitFor(() => {
      expect(screen.getByTestId('ask-clarity-answer')).toBeInTheDocument();
    });
    expect(screen.queryByText(/chain_of_thought/i)).not.toBeInTheDocument();
    expect(screen.queryByText(/thinking/i)).not.toBeInTheDocument();
  });

  // Test 13: No raw prompt rendered
  it('does not render raw prompt', () => {
    renderPanel();
    expect(screen.queryByText(/^prompt$/i)).not.toBeInTheDocument();
    expect(screen.queryByText(/system_prompt/i)).not.toBeInTheDocument();
  });

  // Test 14: No share/public/approval/execute controls
  it('does not render share, approve, or execute controls', () => {
    renderPanel();
    expect(screen.queryByText(/share/i)).not.toBeInTheDocument();
    expect(screen.queryByText(/approve/i)).not.toBeInTheDocument();
    expect(screen.queryByText(/execute/i)).not.toBeInTheDocument();
  });

  // Test 15: No mutation controls
  it('does not render mutation controls', () => {
    renderPanel();
    expect(screen.queryByText(/edit source/i)).not.toBeInTheDocument();
    expect(screen.queryByText(/delete source/i)).not.toBeInTheDocument();
    expect(screen.queryByText(/modify/i)).not.toBeInTheDocument();
  });
});
