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
  const [storageObjectId, setStorageObjectId] = useState<string | null>(null);
  const [loading, setLoading] = useState(mode === 'edit');
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');
  const [downloadUrl, setDownloadUrl] = useState('');
  const [downloadError, setDownloadError] = useState('');

  useEffect(() => {
    if (mode === 'edit' && artifactId) {
      api.getArtifact(artifactId)
        .then((art: any) => {
          setArtifactType(art.artifact_type);
          setTitle(art.title);
          setDescription(art.description || '');
          setContentMarkdown(art.content_markdown || '');
          setStatus(art.status);
          setStorageObjectId(art.storage_object_id || null);
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

  const handleDownload = () => {
    if (!artifactId) return;
    setDownloadError('');
    api.downloadArtifact(artifactId)
      .then((resp: any) => {
        setDownloadUrl(resp.download_url || '');
      })
      .catch((e: unknown) => {
        if (e instanceof ApiError) setDownloadError(e.message);
        else setDownloadError('Failed to generate download link');
      });
  };

  const handleCopyMarkdown = () => {
    if (contentMarkdown) {
      navigator.clipboard?.writeText(contentMarkdown);
    }
  };

  if (loading) return null;

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4" data-testid="artifact-editor">
      <div className="bg-surface border border-border rounded-lg p-6 w-full max-w-3xl max-h-[85vh] overflow-y-auto">
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-lg font-semibold">
            {mode === 'create' ? 'New Artifact' : 'Edit Artifact'}
          </h2>
          <button onClick={onClose} className="text-muted-foreground hover:text-white">✕</button>
        </div>

        {error && <div className="text-destructive text-sm mb-3">{error}</div>}

        <div className="space-y-3">
          {mode === 'create' && (
            <div>
              <label className="text-xs text-muted-foreground">Type</label>
              <select
                value={artifactType}
                onChange={(e) => setArtifactType(e.target.value)}
                className="w-full bg-background border border-border rounded px-2 py-1.5 text-sm"
                data-testid="editor-type"
              >
                {TYPES.map(t => <option key={t.value} value={t.value}>{t.label}</option>)}
              </select>
            </div>
          )}

          <div>
            <label className="text-xs text-muted-foreground">Title</label>
            <input
              type="text"
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              className="w-full bg-background border border-border rounded px-3 py-1.5 text-sm"
              data-testid="editor-title"
            />
          </div>

          <div>
            <label className="text-xs text-muted-foreground">Description</label>
            <input
              type="text"
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              className="w-full bg-background border border-border rounded px-3 py-1.5 text-sm"
              data-testid="editor-description"
            />
          </div>

          <div>
            <label className="text-xs text-muted-foreground">Status</label>
            <select
              value={status}
              onChange={(e) => setStatus(e.target.value)}
              className="w-full bg-background border border-border rounded px-2 py-1.5 text-sm"
              data-testid="editor-status"
            >
              {STATUSES.map(s => <option key={s.value} value={s.value}>{s.label}</option>)}
            </select>
          </div>

          <div>
            <label className="text-xs text-muted-foreground">Content (Markdown)</label>
            <textarea
              value={contentMarkdown}
              onChange={(e) => setContentMarkdown(e.target.value)}
              rows={12}
              className="w-full bg-background border border-border rounded px-3 py-2 text-sm font-mono"
              placeholder="Write markdown content..."
              data-testid="editor-content"
            />
          </div>

          {/* Track 7: Download/Export actions */}
          {mode === 'edit' && (
            <div className="flex flex-wrap gap-2 items-center pt-2 border-t border-border" data-testid="editor-actions">
              {/* Download — only for file-backed artifacts */}
              {storageObjectId && (
                <button
                  onClick={handleDownload}
                  className="px-3 py-1 bg-indigo-600 text-white rounded text-sm"
                  data-testid="editor-download"
                >
                  ⬇ Download
                </button>
              )}
              {/* Export Markdown — only for inline content */}
              {contentMarkdown && !storageObjectId && (
                <a
                  href={api.exportArtifactUrl(artifactId!, 'markdown')}
                  className="px-3 py-1 bg-muted text-white rounded text-sm no-underline"
                  data-testid="editor-export-md"
                >
                  📄 Export MD
                </a>
              )}
              {/* Export PDF — only for inline content */}
              {contentMarkdown && !storageObjectId && (
                <a
                  href={api.exportArtifactUrl(artifactId!, 'pdf')}
                  className="px-3 py-1 bg-muted text-white rounded text-sm no-underline"
                  data-testid="editor-export-pdf"
                >
                  📕 Export PDF
                </a>
              )}
              {/* Copy Markdown */}
              {contentMarkdown && (
                <button
                  onClick={handleCopyMarkdown}
                  className="px-3 py-1 bg-muted text-white rounded text-sm"
                  data-testid="editor-copy-md"
                >
                  📋 Copy MD
                </button>
              )}
              {/* Download URL + expiry note */}
              {downloadUrl && (
                <div className="text-xs text-muted-foreground" data-testid="editor-download-note">
                  Link valid for 15 minutes
                </div>
              )}
              {downloadError && (
                <div className="text-xs text-destructive" data-testid="editor-download-error">{downloadError}</div>
              )}
            </div>
          )}

          <div className="flex gap-2 justify-end">
            {mode === 'edit' && (
              <button
                onClick={handleArchive}
                disabled={saving}
                className="px-3 py-1.5 bg-destructive/15 text-destructive rounded text-sm disabled:opacity-50"
                data-testid="editor-archive"
              >
                Archive
              </button>
            )}
            <button
              onClick={handleSave}
              disabled={saving}
              className="px-4 py-1.5 bg-primary text-primary-foreground rounded text-sm disabled:opacity-50"
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
