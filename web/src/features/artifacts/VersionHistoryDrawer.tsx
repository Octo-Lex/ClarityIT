import { useState, useEffect } from 'react';
import { api } from '../../api/client';

interface VersionItem {
  id: string;
  version_number: number;
  word_count: number;
  source: string;
  change_summary?: string;
  created_by?: string;
  created_at: string;
}

interface VersionDetail {
  id: string;
  version_number: number;
  document_json: any;
  word_count: number;
  source: string;
  change_summary?: string;
  created_at: string;
}

const SOURCE_BADGES: Record<string, { label: string; color: string }> = {
  user_save: { label: 'Saved', color: 'bg-info/15 text-info' },
  agent_assisted_edit: { label: 'AI Assist', color: 'bg-info/15 text-info' },
  generated: { label: 'Generated', color: 'bg-success/15 text-success' },
  template: { label: 'Template', color: 'bg-info/15 text-info' },
  restore: { label: 'Restored', color: 'bg-warning/20 text-warning' },
};

interface Props {
  artifactId: string;
  open: boolean;
  onClose: () => void;
  archived: boolean;
  onRestored: (documentJson: any, wordCount: number) => void;
}

export default function VersionHistoryDrawer({ artifactId, open, onClose, archived, onRestored }: Props) {
  const [versions, setVersions] = useState<VersionItem[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const [selectedVersion, setSelectedVersion] = useState<VersionDetail | null>(null);
  const [previewLoading, setPreviewLoading] = useState(false);
  const [confirmRestore, setConfirmRestore] = useState<string | null>(null);
  const [restoring, setRestoring] = useState(false);

  useEffect(() => {
    if (!open || !artifactId) return;
    loadVersions();
  }, [open, artifactId]);

  const loadVersions = async () => {
    setLoading(true);
    setError('');
    try {
      const resp = await api.listVersions(artifactId);
      setVersions(resp.versions || []);
    } catch {
      setError('Failed to load versions');
    } finally {
      setLoading(false);
    }
  };

  const selectVersion = async (versionId: string) => {
    setPreviewLoading(true);
    setError('');
    try {
      const detail = await api.getVersion(artifactId, versionId);
      setSelectedVersion(detail as unknown as VersionDetail);
    } catch {
      setError('Failed to load version');
    } finally {
      setPreviewLoading(false);
    }
  };

  const doRestore = async (versionId: string) => {
    setRestoring(true);
    setError('');
    try {
      const resp = await api.restoreVersion(artifactId, versionId);
      setConfirmRestore(null);
      setSelectedVersion(null);
      await loadVersions();
      onRestored(resp.document_json, resp.word_count);
    } catch {
      setError('Restore failed');
    } finally {
      setRestoring(false);
    }
  };

  if (!open) return null;

  return (
    <div className="fixed inset-0 z-50 flex" data-testid="version-drawer">
      <div className="absolute inset-0 bg-black/30" onClick={onClose} />
      <div className="ml-auto h-full w-96 bg-[var(--bg)] border-l border-[var(--border)] overflow-y-auto relative">
        {/* Header */}
        <div className="sticky top-0 bg-[var(--bg)] border-b border-[var(--border)] px-4 py-3 flex items-center justify-between z-10">
          <h2 className="text-sm font-semibold">Version History</h2>
          <button onClick={onClose} className="text-[var(--text-muted)] hover:text-[var(--text)]" data-testid="close-drawer">✕</button>
        </div>

        {/* Loading */}
        {loading && (
          <div className="p-4 text-center text-sm text-[var(--text-muted)]" data-testid="version-loading">
            Loading versions...
          </div>
        )}

        {/* Error */}
        {error && (
          <div className="p-4 text-sm text-red-400" data-testid="version-error">{error}</div>
        )}

        {/* Version list */}
        {!loading && !error && (
          <div className="divide-y divide-[var(--border)]" data-testid="version-list">
            {versions.map((v) => {
              const badge = SOURCE_BADGES[v.source] || { label: v.source, color: 'bg-muted text-muted-foreground' };
              return (
                <div
                  key={v.id}
                  className={`p-3 cursor-pointer hover:bg-[var(--card)] ${selectedVersion?.id === v.id ? 'bg-[var(--card)]' : ''}`}
                  onClick={() => selectVersion(v.id)}
                  data-testid={`version-item-${v.version_number}`}
                >
                  <div className="flex items-center gap-2 mb-1">
                    <span className="font-medium text-sm">v{v.version_number}</span>
                    <span className={`text-xs px-1.5 py-0.5 rounded ${badge.color}`} data-testid={`version-badge-${v.version_number}`}>
                      {badge.label}
                    </span>
                  </div>
                  <div className="text-xs text-[var(--text-muted)]">
                    {v.word_count} words · {new Date(v.created_at).toLocaleString()}
                  </div>
                  {v.change_summary && (
                    <div className="text-xs mt-1 text-[var(--text-muted)]" data-testid={`version-summary-${v.version_number}`}>
                      {v.change_summary}
                    </div>
                  )}
                </div>
              );
            })}
            {versions.length === 0 && (
              <div className="p-4 text-center text-sm text-[var(--text-muted)]">No versions yet</div>
            )}
          </div>
        )}

        {/* Preview panel */}
        {selectedVersion && (
          <div className="border-t border-[var(--border)] p-4 bg-[var(--card)]" data-testid="version-preview">
            <div className="text-xs font-semibold mb-2">Preview — v{selectedVersion.version_number}</div>
            {previewLoading ? (
              <div className="text-xs text-[var(--text-muted)]">Loading...</div>
            ) : (
              <div className="text-xs space-y-1 max-h-48 overflow-y-auto">
                {(selectedVersion.document_json?.blocks || []).map((blk: any, i: number) => (
                  <div key={i} className="truncate">
                    {blk.type === 'heading' && `## ${blk.text}`}
                    {blk.type === 'paragraph' && blk.text}
                    {blk.type === 'bullets' && `• ${blk.items?.[0] || ''}`}
                    {(blk.type === 'quote' || blk.type === 'callout') && blk.text}
                  </div>
                ))}
              </div>
            )}
            {!archived && (
              <button
                onClick={() => setConfirmRestore(selectedVersion.id)}
                className="mt-3 px-3 py-1 text-xs bg-[var(--primary)] text-white rounded hover:opacity-90"
                data-testid="restore-button"
              >
                Restore this version
              </button>
            )}
            {archived && (
              <div className="mt-3 text-xs text-[var(--text-muted)]" data-testid="restore-archived-notice">
                Document is archived — restore unavailable
              </div>
            )}
          </div>
        )}

        {/* Restore confirmation */}
        {confirmRestore && (
          <div className="fixed inset-0 z-50 flex items-center justify-center" data-testid="restore-confirm">
            <div className="absolute inset-0 bg-black/50" onClick={() => setConfirmRestore(null)} />
            <div className="relative bg-[var(--bg)] border border-[var(--border)] rounded-lg p-4 max-w-sm">
              <div className="text-sm font-medium mb-2">Restore Version?</div>
              <div className="text-xs text-[var(--text-muted)] mb-4">
                This will create a new version with the selected content. Current content will be preserved in history.
              </div>
              <div className="flex gap-2 justify-end">
                <button
                  onClick={() => setConfirmRestore(null)}
                  className="px-3 py-1 text-xs border border-[var(--border)] rounded"
                  data-testid="restore-cancel"
                >Cancel</button>
                <button
                  onClick={() => doRestore(confirmRestore)}
                  disabled={restoring}
                  className="px-3 py-1 text-xs bg-[var(--primary)] text-white rounded"
                  data-testid="restore-confirm-button"
                >{restoring ? 'Restoring...' : 'Restore'}</button>
              </div>
            </div>
          </div>
        )}

        {/* No delete button — explicitly absent */}
      </div>
    </div>
  );
}
