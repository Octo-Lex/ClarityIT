import { useState, useEffect } from 'react';
import { api, ApiError } from '../../api/client';

interface Props {
  mode: 'create' | 'edit';
  artifactId?: string;
  onClose: () => void;
}

const TYPES = [
  { value: 'document', label: 'Document' },
  { value: 'report', label: 'Report' },
  { value: 'presentation', label: 'Presentation' },
  { value: 'meeting_summary', label: 'Meeting Summary' },
  { value: 'status_report', label: 'Status Report' },
  { value: 'decision_memo', label: 'Decision Memo' },
  { value: 'training_deck', label: 'Training Deck' },
];

const STATUSES = [
  { value: 'draft', label: 'Draft' },
  { value: 'published', label: 'Published' },
  { value: 'archived', label: 'Archived' },
];

export default function ArtifactEditor({ mode, artifactId, onClose }: Props) {
  const [artifactType, setArtifactType] = useState('document');
  const [title, setTitle] = useState('');
  const [description, setDescription] = useState('');
  const [contentMarkdown, setContentMarkdown] = useState('');
  const [status, setStatus] = useState('draft');
  const [loading, setLoading] = useState(mode === 'edit');
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');

  useEffect(() => {
    if (mode === 'edit' && artifactId) {
      api.getArtifact(artifactId)
        .then((art: any) => {
          setArtifactType(art.artifact_type);
          setTitle(art.title);
          setDescription(art.description || '');
          setContentMarkdown(art.content_markdown || '');
          setStatus(art.status);
          setLoading(false);
        })
        .catch(() => { setError('Failed to load artifact'); setLoading(false); });
    }
  }, [mode, artifactId]);

  const handleSave = () => {
    if (!title.trim()) { setError('Title is required'); return; }
    setSaving(true);
    setError('');

    const data: any = {
      title,
      description,
      content_markdown: contentMarkdown,
      status,
    };

    if (mode === 'create') {
      data.artifact_type = artifactType;
      api.createArtifact(data)
        .then(() => { setSaving(false); onClose(); })
        .catch((e: unknown) => {
          if (e instanceof ApiError) setError(e.message);
          else setError('Failed to create artifact');
          setSaving(false);
        });
    } else if (artifactId) {
      api.updateArtifact(artifactId, data)
        .then(() => { setSaving(false); onClose(); })
        .catch((e: unknown) => {
          if (e instanceof ApiError) setError(e.message);
          else setError('Failed to update artifact');
          setSaving(false);
        });
    }
  };

  const handleArchive = () => {
    if (!artifactId) return;
    setSaving(true);
    api.archiveArtifact(artifactId)
      .then(() => { setSaving(false); onClose(); })
      .catch((e: unknown) => {
        if (e instanceof ApiError) setError(e.message);
        else setError('Failed to archive artifact');
        setSaving(false);
      });
  };

  if (loading) return null;

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4" data-testid="artifact-editor">
      <div className="bg-[var(--bg-card)] border border-[var(--border)] rounded-lg p-6 w-full max-w-3xl max-h-[85vh] overflow-y-auto">
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-lg font-semibold">
            {mode === 'create' ? 'New Artifact' : 'Edit Artifact'}
          </h2>
          <button onClick={onClose} className="text-[var(--text-muted)] hover:text-white">✕</button>
        </div>

        {error && <div className="text-red-400 text-sm mb-3">{error}</div>}

        <div className="space-y-3">
          {mode === 'create' && (
            <div>
              <label className="text-xs text-[var(--text-muted)]">Type</label>
              <select
                value={artifactType}
                onChange={(e) => setArtifactType(e.target.value)}
                className="w-full bg-[var(--bg-input)] border border-[var(--border)] rounded px-2 py-1.5 text-sm"
                data-testid="editor-type"
              >
                {TYPES.map(t => <option key={t.value} value={t.value}>{t.label}</option>)}
              </select>
            </div>
          )}

          <div>
            <label className="text-xs text-[var(--text-muted)]">Title</label>
            <input
              type="text"
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              className="w-full bg-[var(--bg-input)] border border-[var(--border)] rounded px-3 py-1.5 text-sm"
              data-testid="editor-title"
            />
          </div>

          <div>
            <label className="text-xs text-[var(--text-muted)]">Description</label>
            <input
              type="text"
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              className="w-full bg-[var(--bg-input)] border border-[var(--border)] rounded px-3 py-1.5 text-sm"
              data-testid="editor-description"
            />
          </div>

          <div>
            <label className="text-xs text-[var(--text-muted)]">Status</label>
            <select
              value={status}
              onChange={(e) => setStatus(e.target.value)}
              className="w-full bg-[var(--bg-input)] border border-[var(--border)] rounded px-2 py-1.5 text-sm"
              data-testid="editor-status"
            >
              {STATUSES.map(s => <option key={s.value} value={s.value}>{s.label}</option>)}
            </select>
          </div>

          <div>
            <label className="text-xs text-[var(--text-muted)]">Content (Markdown)</label>
            <textarea
              value={contentMarkdown}
              onChange={(e) => setContentMarkdown(e.target.value)}
              rows={12}
              className="w-full bg-[var(--bg-input)] border border-[var(--border)] rounded px-3 py-2 text-sm font-mono"
              placeholder="Write markdown content..."
              data-testid="editor-content"
            />
          </div>

          <div className="flex gap-2 justify-end">
            {mode === 'edit' && (
              <button
                onClick={handleArchive}
                disabled={saving}
                className="px-3 py-1.5 bg-red-900/40 text-red-300 rounded text-sm disabled:opacity-50"
                data-testid="editor-archive"
              >
                Archive
              </button>
            )}
            <button
              onClick={handleSave}
              disabled={saving}
              className="px-4 py-1.5 bg-blue-600 text-white rounded text-sm disabled:opacity-50"
              data-testid="editor-save"
            >
              {saving ? 'Saving...' : 'Save'}
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}
