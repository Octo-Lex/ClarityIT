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
import MeetingSummaryEditor from '../features/artifacts/MeetingSummaryEditor';
import { api } from '../api/client';

describe('Track 3 — Meeting Summary and Action Items', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  // Test 1: meeting summary editor renders
  it('renders meeting summary editor', () => {
    render(
      <MemoryRouter>
        <MeetingSummaryEditor mode="create" onClose={() => {}} onSaved={() => {}} />
      </MemoryRouter>
    );
    expect(screen.getByTestId('meeting-editor')).toBeInTheDocument();
  });

  // Test 2: attendees section add/remove works
  it('attendees add/remove works', () => {
    render(
      <MemoryRouter>
        <MeetingSummaryEditor mode="create" onClose={() => {}} onSaved={() => {}} />
      </MemoryRouter>
    );
    expect(screen.getByTestId('meeting-attendees-empty')).toBeInTheDocument();
    fireEvent.click(screen.getByTestId('meeting-add-attendee'));
    expect(screen.getByTestId('meeting-attendee-name-0')).toBeInTheDocument();
    fireEvent.click(screen.getByTestId('meeting-attendee-remove-0'));
    expect(screen.getByTestId('meeting-attendees-empty')).toBeInTheDocument();
  });

  // Test 3: agenda section add/remove works
  it('agenda add/remove works', () => {
    render(
      <MemoryRouter>
        <MeetingSummaryEditor mode="create" onClose={() => {}} onSaved={() => {}} />
      </MemoryRouter>
    );
    expect(screen.getByTestId('meeting-agenda-empty')).toBeInTheDocument();
    fireEvent.click(screen.getByTestId('meeting-add-agenda'));
    expect(screen.getByTestId('meeting-agenda-title-0')).toBeInTheDocument();
    fireEvent.click(screen.getByTestId('meeting-agenda-remove-0'));
    expect(screen.getByTestId('meeting-agenda-empty')).toBeInTheDocument();
  });

  // Test 4: decisions section add/remove works
  it('decisions add/remove works', () => {
    render(
      <MemoryRouter>
        <MeetingSummaryEditor mode="create" onClose={() => {}} onSaved={() => {}} />
      </MemoryRouter>
    );
    expect(screen.getByTestId('meeting-decisions-empty')).toBeInTheDocument();
    fireEvent.click(screen.getByTestId('meeting-add-decision'));
    expect(screen.getByTestId('meeting-decision-text-0')).toBeInTheDocument();
    fireEvent.click(screen.getByTestId('meeting-decision-remove-0'));
    expect(screen.getByTestId('meeting-decisions-empty')).toBeInTheDocument();
  });

  // Test 5: action items section add/remove works
  it('action items add/remove works', () => {
    render(
      <MemoryRouter>
        <MeetingSummaryEditor mode="create" onClose={() => {}} onSaved={() => {}} />
      </MemoryRouter>
    );
    expect(screen.getByTestId('meeting-actions-empty')).toBeInTheDocument();
    fireEvent.click(screen.getByTestId('meeting-add-action'));
    expect(screen.getByTestId('meeting-action-text-0')).toBeInTheDocument();
    fireEvent.click(screen.getByTestId('meeting-action-remove-0'));
    expect(screen.getByTestId('meeting-actions-empty')).toBeInTheDocument();
  });

  // Test 6: action item assignee/due date/status render
  it('action item fields render', () => {
    render(
      <MemoryRouter>
        <MeetingSummaryEditor mode="create" onClose={() => {}} onSaved={() => {}} />
      </MemoryRouter>
    );
    fireEvent.click(screen.getByTestId('meeting-add-action'));
    expect(screen.getByTestId('meeting-action-assignee-0')).toBeInTheDocument();
    expect(screen.getByTestId('meeting-action-due-0')).toBeInTheDocument();
    expect(screen.getByTestId('meeting-action-status-0')).toBeInTheDocument();
    expect(screen.getByTestId('meeting-action-check-0')).toBeInTheDocument();
  });

  // Test 7: create meeting summary calls API
  it('calls create API on save', async () => {
    vi.mocked(api.createMeetingSummary).mockResolvedValue({ id: 'mtg-1' });
    render(
      <MemoryRouter>
        <MeetingSummaryEditor mode="create" onClose={() => {}} onSaved={() => {}} />
      </MemoryRouter>
    );
    fireEvent.change(screen.getByTestId('meeting-title'), { target: { value: 'Test Meeting' } });
    fireEvent.click(screen.getByTestId('meeting-save'));
    await waitFor(() => expect(api.createMeetingSummary).toHaveBeenCalledTimes(1));
  });

  // Test 8: edit meeting summary calls API
  it('calls update API on edit save', async () => {
    vi.mocked(api.getMeetingSummary).mockResolvedValue({
      title: 'Existing Meeting',
      attendees: [{ name: 'Alice' }],
    });
    vi.mocked(api.updateMeetingSummary).mockResolvedValue({});
    render(
      <MemoryRouter>
        <MeetingSummaryEditor mode="edit" meetingId="mtg-1" onClose={() => {}} onSaved={() => {}} />
      </MemoryRouter>
    );
    await waitFor(() => expect(screen.getByTestId('meeting-title')).toBeInTheDocument());
    fireEvent.click(screen.getByTestId('meeting-save'));
    await waitFor(() => expect(api.updateMeetingSummary).toHaveBeenCalledTimes(1));
  });

  // Test 9: empty section states render
  it('renders all empty states', () => {
    render(
      <MemoryRouter>
        <MeetingSummaryEditor mode="create" onClose={() => {}} onSaved={() => {}} />
      </MemoryRouter>
    );
    expect(screen.getByTestId('meeting-attendees-empty')).toBeInTheDocument();
    expect(screen.getByTestId('meeting-agenda-empty')).toBeInTheDocument();
    expect(screen.getByTestId('meeting-decisions-empty')).toBeInTheDocument();
    expect(screen.getByTestId('meeting-actions-empty')).toBeInTheDocument();
  });

  // Test 10: no calendar/email/create-work-item buttons rendered
  it('does not render out-of-scope buttons', () => {
    const { container } = render(
      <MemoryRouter>
        <MeetingSummaryEditor mode="create" onClose={() => {}} onSaved={() => {}} />
      </MemoryRouter>
    );
    expect(container.textContent).not.toContain('Calendar');
    expect(container.textContent).not.toContain('Send Email');
    expect(container.textContent).not.toContain('Create Work Item');
    expect(container.textContent).not.toContain('Approve');
    expect(container.textContent).not.toContain('Execute');
  });

  // Test 11: meeting summary button renders on Artifacts page
  it('renders Meeting Summary button on artifacts page', async () => {
    vi.mocked(api.listArtifacts).mockResolvedValue([]);
    render(
      <MemoryRouter>
        <ArtifactsPage />
      </MemoryRouter>
    );
    await waitFor(() => expect(screen.getByTestId('artifacts-new-meeting-btn')).toBeInTheDocument());
  });
});
