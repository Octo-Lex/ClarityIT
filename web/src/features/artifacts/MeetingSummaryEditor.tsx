import { useState, useEffect } from 'react';
import { api, ApiError } from '../../api/client';

interface Props {
  mode: 'create' | 'edit';
  meetingId?: string;
  onClose: () => void;
  onSaved: () => void;
}

interface ActionItem {
  text: string;
  assignee?: string;
  due_date?: string;
  status?: string;
}

interface Attendee {
  name: string;
  role?: string;
}

interface AgendaItem {
  title: string;
  notes?: string;
}

interface Decision {
  text: string;
  decided_by?: string;
}

const ACTION_STATUSES = ['open', 'in_progress', 'done', 'blocked'];

export default function MeetingSummaryEditor({ mode, meetingId, onClose, onSaved }: Props) {
  const [title, setTitle] = useState('');
  const [description, setDescription] = useState('');
  const [contentMarkdown, setContentMarkdown] = useState('');
  const [meetingDate, setMeetingDate] = useState('');
  const [duration, setDuration] = useState<number | ''>('');
  const [attendees, setAttendees] = useState<Attendee[]>([]);
  const [agendaItems, setAgendaItems] = useState<AgendaItem[]>([]);
  const [decisions, setDecisions] = useState<Decision[]>([]);
  const [actionItems, setActionItems] = useState<ActionItem[]>([]);
  const [loading, setLoading] = useState(mode === 'edit');
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');

  useEffect(() => {
    if (mode === 'edit' && meetingId) {
      api.getMeetingSummary(meetingId)
        .then((m: any) => {
          setTitle(m.title || '');
          setDescription(m.description || '');
          setContentMarkdown(m.content_markdown || '');
          setMeetingDate(m.meeting_date || '');
          setDuration(m.duration_minutes ?? '');
          setAttendees(m.attendees || []);
          setAgendaItems(m.agenda_items || []);
          setDecisions(m.decisions || []);
          setActionItems(m.action_items || []);
          setLoading(false);
        })
        .catch(() => { setError('Failed to load meeting summary'); setLoading(false); });
    }
  }, [mode, meetingId]);

  const handleSave = () => {
    if (!title.trim()) { setError('Title is required'); return; }
    setSaving(true);
    setError('');

    const data: any = {
      title,
      description,
      content_markdown: contentMarkdown,
      meeting_date: meetingDate || undefined,
      duration_minutes: duration === '' ? undefined : duration,
      attendees,
      agenda_items: agendaItems,
      decisions,
      action_items: actionItems,
    };

    const apiCall = mode === 'create'
      ? api.createMeetingSummary(data)
      : api.updateMeetingSummary(meetingId!, data);

    apiCall
      .then(() => { setSaving(false); onSaved(); onClose(); })
      .catch((e: unknown) => {
        if (e instanceof ApiError) setError(e.message);
        else setError('Failed to save meeting summary');
        setSaving(false);
      });
  };

  if (loading) return null;

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4" data-testid="meeting-editor">
      <div className="bg-[var(--bg-card)] border border-[var(--border)] rounded-lg p-6 w-full max-w-4xl max-h-[90vh] overflow-y-auto">
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-lg font-semibold">
            {mode === 'create' ? 'New Meeting Summary' : 'Edit Meeting Summary'}
          </h2>
          <button onClick={onClose} className="text-[var(--text-muted)] hover:text-white">✕</button>
        </div>

        {error && <div className="text-red-400 text-sm mb-3">{error}</div>}

        <div className="space-y-4">
          {/* Basic fields */}
          <div>
            <label className="text-xs text-[var(--text-muted)]">Title</label>
            <input type="text" value={title} onChange={(e) => setTitle(e.target.value)}
              className="w-full bg-[var(--bg-input)] border border-[var(--border)] rounded px-3 py-1.5 text-sm"
              data-testid="meeting-title" />
          </div>

          <div>
            <label className="text-xs text-[var(--text-muted)]">Description</label>
            <input type="text" value={description} onChange={(e) => setDescription(e.target.value)}
              className="w-full bg-[var(--bg-input)] border border-[var(--border)] rounded px-3 py-1.5 text-sm"
              data-testid="meeting-description" />
          </div>

          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="text-xs text-[var(--text-muted)]">Meeting Date</label>
              <input type="date" value={meetingDate} onChange={(e) => setMeetingDate(e.target.value)}
                className="w-full bg-[var(--bg-input)] border border-[var(--border)] rounded px-3 py-1.5 text-sm"
                data-testid="meeting-date" />
            </div>
            <div>
              <label className="text-xs text-[var(--text-muted)]">Duration (minutes)</label>
              <input type="number" min={0} max={1440} value={duration}
                onChange={(e) => setDuration(e.target.value === '' ? '' : parseInt(e.target.value))}
                className="w-full bg-[var(--bg-input)] border border-[var(--border)] rounded px-3 py-1.5 text-sm"
                data-testid="meeting-duration" />
            </div>
          </div>

          {/* Attendees */}
          <div data-testid="meeting-attendees-section">
            <div className="flex items-center justify-between mb-1">
              <label className="text-xs text-[var(--text-muted)]">Attendees</label>
              <button onClick={() => setAttendees([...attendees, { name: '' }])}
                className="text-xs text-blue-400 hover:text-blue-300" data-testid="meeting-add-attendee">+ Add</button>
            </div>
            {attendees.length === 0 && <p className="text-xs text-[var(--text-muted)] italic" data-testid="meeting-attendees-empty">No attendees yet</p>}
            {attendees.map((a, i) => (
              <div key={i} className="flex gap-2 mb-1">
                <input type="text" placeholder="Name" value={a.name}
                  onChange={(e) => { const c = [...attendees]; c[i].name = e.target.value; setAttendees(c); }}
                  className="flex-1 bg-[var(--bg-input)] border border-[var(--border)] rounded px-2 py-1 text-sm" data-testid={`meeting-attendee-name-${i}`} />
                <input type="text" placeholder="Role" value={a.role || ''}
                  onChange={(e) => { const c = [...attendees]; c[i].role = e.target.value; setAttendees(c); }}
                  className="w-32 bg-[var(--bg-input)] border border-[var(--border)] rounded px-2 py-1 text-sm" data-testid={`meeting-attendee-role-${i}`} />
                <button onClick={() => setAttendees(attendees.filter((_, idx) => idx !== i))}
                  className="text-red-400 px-2" data-testid={`meeting-attendee-remove-${i}`}>✕</button>
              </div>
            ))}
          </div>

          {/* Agenda Items */}
          <div data-testid="meeting-agenda-section">
            <div className="flex items-center justify-between mb-1">
              <label className="text-xs text-[var(--text-muted)]">Agenda Items</label>
              <button onClick={() => setAgendaItems([...agendaItems, { title: '' }])}
                className="text-xs text-blue-400 hover:text-blue-300" data-testid="meeting-add-agenda">+ Add</button>
            </div>
            {agendaItems.length === 0 && <p className="text-xs text-[var(--text-muted)] italic" data-testid="meeting-agenda-empty">No agenda items yet</p>}
            {agendaItems.map((a, i) => (
              <div key={i} className="flex gap-2 mb-1">
                <input type="text" placeholder="Title" value={a.title}
                  onChange={(e) => { const c = [...agendaItems]; c[i].title = e.target.value; setAgendaItems(c); }}
                  className="flex-1 bg-[var(--bg-input)] border border-[var(--border)] rounded px-2 py-1 text-sm" data-testid={`meeting-agenda-title-${i}`} />
                <input type="text" placeholder="Notes" value={a.notes || ''}
                  onChange={(e) => { const c = [...agendaItems]; c[i].notes = e.target.value; setAgendaItems(c); }}
                  className="flex-1 bg-[var(--bg-input)] border border-[var(--border)] rounded px-2 py-1 text-sm" data-testid={`meeting-agenda-notes-${i}`} />
                <button onClick={() => setAgendaItems(agendaItems.filter((_, idx) => idx !== i))}
                  className="text-red-400 px-2" data-testid={`meeting-agenda-remove-${i}`}>✕</button>
              </div>
            ))}
          </div>

          {/* Decisions */}
          <div data-testid="meeting-decisions-section">
            <div className="flex items-center justify-between mb-1">
              <label className="text-xs text-[var(--text-muted)]">Decisions</label>
              <button onClick={() => setDecisions([...decisions, { text: '' }])}
                className="text-xs text-blue-400 hover:text-blue-300" data-testid="meeting-add-decision">+ Add</button>
            </div>
            {decisions.length === 0 && <p className="text-xs text-[var(--text-muted)] italic" data-testid="meeting-decisions-empty">No decisions yet</p>}
            {decisions.map((d, i) => (
              <div key={i} className="flex gap-2 mb-1">
                <input type="text" placeholder="Decision" value={d.text}
                  onChange={(e) => { const c = [...decisions]; c[i].text = e.target.value; setDecisions(c); }}
                  className="flex-1 bg-[var(--bg-input)] border border-[var(--border)] rounded px-2 py-1 text-sm" data-testid={`meeting-decision-text-${i}`} />
                <input type="text" placeholder="By" value={d.decided_by || ''}
                  onChange={(e) => { const c = [...decisions]; c[i].decided_by = e.target.value; setDecisions(c); }}
                  className="w-32 bg-[var(--bg-input)] border border-[var(--border)] rounded px-2 py-1 text-sm" data-testid={`meeting-decision-by-${i}`} />
                <button onClick={() => setDecisions(decisions.filter((_, idx) => idx !== i))}
                  className="text-red-400 px-2" data-testid={`meeting-decision-remove-${i}`}>✕</button>
              </div>
            ))}
          </div>

          {/* Action Items */}
          <div data-testid="meeting-actions-section">
            <div className="flex items-center justify-between mb-1">
              <label className="text-xs text-[var(--text-muted)]">Action Items</label>
              <button onClick={() => setActionItems([...actionItems, { text: '', status: 'open' }])}
                className="text-xs text-blue-400 hover:text-blue-300" data-testid="meeting-add-action">+ Add</button>
            </div>
            {actionItems.length === 0 && <p className="text-xs text-[var(--text-muted)] italic" data-testid="meeting-actions-empty">No action items yet</p>}
            {actionItems.map((a, i) => (
              <div key={i} className="flex gap-2 mb-1 items-center">
                <input type="checkbox" checked={a.status === 'done'}
                  onChange={(e) => { const c = [...actionItems]; c[i].status = e.target.checked ? 'done' : 'open'; setActionItems(c); }}
                  className="mr-1" data-testid={`meeting-action-check-${i}`} />
                <input type="text" placeholder="Action" value={a.text}
                  onChange={(e) => { const c = [...actionItems]; c[i].text = e.target.value; setActionItems(c); }}
                  className="flex-1 bg-[var(--bg-input)] border border-[var(--border)] rounded px-2 py-1 text-sm" data-testid={`meeting-action-text-${i}`} />
                <input type="text" placeholder="Assignee" value={a.assignee || ''}
                  onChange={(e) => { const c = [...actionItems]; c[i].assignee = e.target.value; setActionItems(c); }}
                  className="w-28 bg-[var(--bg-input)] border border-[var(--border)] rounded px-2 py-1 text-sm" data-testid={`meeting-action-assignee-${i}`} />
                <input type="date" value={a.due_date || ''}
                  onChange={(e) => { const c = [...actionItems]; c[i].due_date = e.target.value; setActionItems(c); }}
                  className="bg-[var(--bg-input)] border border-[var(--border)] rounded px-2 py-1 text-sm" data-testid={`meeting-action-due-${i}`} />
                <select value={a.status || 'open'}
                  onChange={(e) => { const c = [...actionItems]; c[i].status = e.target.value; setActionItems(c); }}
                  className="bg-[var(--bg-input)] border border-[var(--border)] rounded px-2 py-1 text-xs" data-testid={`meeting-action-status-${i}`}>
                  {ACTION_STATUSES.map(s => <option key={s} value={s}>{s}</option>)}
                </select>
                <button onClick={() => setActionItems(actionItems.filter((_, idx) => idx !== i))}
                  className="text-red-400 px-2" data-testid={`meeting-action-remove-${i}`}>✕</button>
              </div>
            ))}
          </div>

          {/* Narrative */}
          <div>
            <label className="text-xs text-[var(--text-muted)]">Notes (Markdown)</label>
            <textarea value={contentMarkdown} onChange={(e) => setContentMarkdown(e.target.value)} rows={4}
              className="w-full bg-[var(--bg-input)] border border-[var(--border)] rounded px-3 py-2 text-sm"
              data-testid="meeting-notes" />
          </div>

          {/* Actions */}
          <div className="flex gap-2 justify-end">
            <button onClick={handleSave} disabled={saving}
              className="px-4 py-1.5 bg-blue-600 text-white rounded text-sm disabled:opacity-50"
              data-testid="meeting-save">Save</button>
          </div>
        </div>
      </div>
    </div>
  );
}
