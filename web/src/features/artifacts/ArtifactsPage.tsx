import { useState, useEffect } from 'react';
import { api, ApiError } from '../../api/client';
import ArtifactEditor from './ArtifactEditor';
import PresentationModal from './PresentationModal';
import MeetingSummaryEditor from './MeetingSummaryEditor';
import StatusReportModal from './StatusReportModal';
import TemplateGallery from './TemplateGallery';

const ARTIFACT_TYPES = [
  { value: '', label: 'All Types' },
  { value: 'document', label: 'Document' },
  { value: 'report', label: 'Report' },
  { value: 'presentation', label: 'Presentation' },
  { value: 'meeting_summary', label: 'Meeting Summary' },
  { value: 'status_report', label: 'Status Report' },
  { value: 'decision_memo', label: 'Decision Memo' },
  { value: 'training_deck', label: 'Training Deck' },
];

const TYPE_COLORS: Record<string, string> = {
  document: 'bg-blue-900/40 text-blue-300',
  report: 'bg-green-900/40 text-green-300',
  presentation: 'bg-purple-900/40 text-purple-300',
  meeting_summary: 'bg-orange-900/40 text-orange-300',
  status_report: 'bg-cyan-900/40 text-cyan-300',
  decision_memo: 'bg-yellow-900/40 text-yellow-300',
  training_deck: 'bg-pink-900/40 text-pink-300',
};

const STATUS_COLORS: Record<string, string> = {
  draft: 'bg-gray-700 text-gray-300',
  published: 'bg-green-900/40 text-green-300',
  archived: 'bg-red-900/40 text-red-300',
};

function formatBytes(bytes: number | null | undefined): string {
  if (!bytes) return '—';
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1048576) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / 1048576).toFixed(1)} MB`;
}

export default function ArtifactsPage() {
  const [artifacts, setArtifacts] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [typeFilter, setTypeFilter] = useState('');
  const [search, setSearch] = useState('');
  const [searching, setSearching] = useState(false);
  const [showCreate, setShowCreate] = useState(false);
  const [showPresentation, setShowPresentation] = useState(false);
  const [showMeeting, setShowMeeting] = useState(false);
  const [showStatusReport, setShowStatusReport] = useState(false);
  const [showTemplates, setShowTemplates] = useState(false);
  const [editingId, setEditingId] = useState<string | null>(null);
  const [storageSummary, setStorageSummary] = useState<any>(null);
  const [recentArtifacts, setRecentArtifacts] = useState<any[]>([]);

  const fetchArtifacts = () => {
    setLoading(true);
    api.listArtifacts({ type: typeFilter || undefined })
      .then((data: any[]) => { setArtifacts(data || []); setLoading(false); })
      .catch((e: unknown) => {
        if (e instanceof ApiError) setError(e.message);
        else setError('Failed to load artifacts');
        setLoading(false);
      });
  };

  const fetchStorageSummary = () => {
    api.getStorageSummary()
      .then((data: any) => setStorageSummary(data))
      .catch(() => {});
  };

  const fetchRecent = () => {
    api.getRecentArtifacts()
      .then((data: any[]) => setRecentArtifacts((data || []).slice(0, 5)))
      .catch(() => {});
  };

  useEffect(() => {
    fetchArtifacts();
    fetchStorageSummary();
    fetchRecent();
  }, [typeFilter]);

  const handleSearch = () => {
    if (!search.trim()) { fetchArtifacts(); return; }
    setSearching(true);
    api.searchArtifacts(search.trim())
      .then((data: any[]) => { setArtifacts(data || []); setSearching(false); })
      .catch(() => {
        setSearching(false);
        fetchArtifacts();
      });
  };

  if (loading) return <div className="p-4 text-[var(--text-muted)]">Loading...</div>;

  if (error && artifacts.length === 0) {
    return (
      <div className="p-4" data-testid="artifacts-unauthorized">
        <div className="text-red-400 text-sm">{error}</div>
      </div>
    );
  }

  return (
    <div className="space-y-4" data-testid="artifacts-page">
      <div className="flex items-center justify-between">
        <h1 className="text-xl font-bold">Artifacts</h1>
        <div className="flex gap-2">
          <button
            onClick={() => setShowPresentation(true)}
            className="px-3 py-1.5 bg-purple-600 text-white rounded text-sm hover:bg-purple-700"
            data-testid="artifacts-generate-presentation-btn"
          >
            ⮤ Generate Presentation
          </button>
          <button
            onClick={() => setShowMeeting(true)}
            className="px-3 py-1.5 bg-green-600 text-white rounded text-sm hover:bg-green-700"
            data-testid="artifacts-new-meeting-btn"
          >
            📋 Meeting Summary
          </button>
          <button
            onClick={() => setShowStatusReport(true)}
            className="px-3 py-1.5 bg-teal-600 text-white rounded text-sm hover:bg-teal-700"
            data-testid="artifacts-status-report-btn"
          >
            📊 Status Report
          </button>
          <button
            onClick={() => setShowTemplates(true)}
            className="px-3 py-1.5 bg-indigo-600 text-white rounded text-sm hover:bg-indigo-700"
            data-testid="artifacts-templates-btn"
          >
            📁 Templates
          </button>
          <button
            onClick={() => setShowCreate(true)}
            className="px-3 py-1.5 bg-blue-600 text-white rounded text-sm hover:bg-blue-700"
            data-testid="artifacts-create-btn"
          >
            + New Artifact
          </button>
        </div>
      </div>

      {/* Storage summary */}
      {storageSummary && (
        <div className="card p-3 flex gap-6 items-center" data-testid="artifacts-storage-summary">
          <div className="text-sm">
            <span className="text-[var(--text-muted)]">Total:</span>{' '}
            <span className="font-medium" data-testid="storage-total">{storageSummary.total_artifacts}</span>
          </div>
          <div className="text-sm">
            <span className="text-[var(--text-muted)]">Files:</span>{' '}
            <span className="font-medium" data-testid="storage-files">{storageSummary.file_artifacts}</span>
          </div>
          <div className="text-sm">
            <span className="text-[var(--text-muted)]">Size:</span>{' '}
            <span className="font-medium" data-testid="storage-size">{formatBytes(storageSummary.total_file_size_bytes)}</span>
          </div>
          {storageSummary.by_format && Object.entries(storageSummary.by_format).map(([fmt, count]: [string, any]) => (
            <div key={fmt} className="text-sm" data-testid={`storage-format-${fmt}`}>
              <span className="px-1.5 py-0.5 text-xs rounded bg-gray-700 text-gray-300">{fmt}: {count}</span>
            </div>
          ))}
        </div>
      )}

      {/* Search bar */}
      <div className="flex gap-3 items-center">
        <select
          value={typeFilter}
          onChange={(e) => { setTypeFilter(e.target.value); setSearch(''); }}
          className="bg-[var(--bg-input)] border border-[var(--border)] rounded px-2 py-1 text-sm"
          data-testid="artifacts-type-filter"
        >
          {ARTIFACT_TYPES.map(t => <option key={t.value} value={t.value}>{t.label}</option>)}
        </select>
        <input
          type="text"
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          onKeyDown={(e) => e.key === 'Enter' && handleSearch()}
          placeholder="Search title, description, content..."
          className="bg-[var(--bg-input)] border border-[var(--border)] rounded px-3 py-1 text-sm flex-1"
          data-testid="artifacts-search-input"
        />
        <button
          onClick={handleSearch}
          disabled={searching}
          className="px-3 py-1 bg-gray-700 rounded text-sm disabled:opacity-50"
          data-testid="artifacts-search-btn"
        >
          {searching ? 'Searching...' : 'Search'}
        </button>
        {search && (
          <button
            onClick={() => { setSearch(''); fetchArtifacts(); }}
            className="px-2 py-1 text-xs text-[var(--text-muted)]"
            data-testid="artifacts-search-clear"
          >
            ✕ Clear
          </button>
        )}
      </div>

      {/* Recent artifacts widget */}
      {recentArtifacts.length > 0 && !search && (
        <div className="card p-3" data-testid="artifacts-recent-widget">
          <div className="text-xs font-semibold text-[var(--text-muted)] mb-2">Recent</div>
          <div className="flex gap-2 overflow-x-auto">
            {recentArtifacts.map((art) => (
              <div
                key={art.id}
                className="border border-[var(--border)] rounded px-2 py-1 text-xs cursor-pointer hover:border-blue-600 whitespace-nowrap"
                onClick={() => setEditingId(art.id)}
                data-testid={`artifacts-recent-${art.id}`}
              >
                {art.title}
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Empty state */}
      {artifacts.length === 0 && !error && (
        <div className="card p-8 text-center" data-testid="artifacts-empty">
          <p className="text-[var(--text-muted)]">No artifacts found. Create your first one!</p>
        </div>
      )}

      {/* Artifact list */}
      {artifacts.length > 0 && (
        <div className="space-y-2" data-testid="artifacts-list">
          {artifacts.map((art) => (
            <div
              key={art.id}
              className="card p-3 flex items-center justify-between cursor-pointer hover:border-blue-600"
              onClick={() => setEditingId(art.id)}
              data-testid={`artifacts-item-${art.id}`}
            >
              <div className="flex-1 min-w-0">
                <div className="flex items-center gap-2 mb-1">
                  <span
                    className={`px-1.5 py-0.5 text-xs rounded ${TYPE_COLORS[art.artifact_type] || 'bg-gray-700 text-gray-300'}`}
                    data-testid={`artifacts-type-${art.id}`}
                  >
                    {art.artifact_type.replace(/_/g, ' ')}
                  </span>
                  <span
                    className={`px-1.5 py-0.5 text-xs rounded ${STATUS_COLORS[art.status] || 'bg-gray-700'}`}
                    data-testid={`artifacts-status-${art.id}`}
                  >
                    {art.status}
                  </span>
                  {/* File metadata badges */}
                  {art.file_format && (
                    <span
                      className="px-1.5 py-0.5 text-xs rounded bg-indigo-900/40 text-indigo-300"
                      data-testid={`artifacts-file-format-${art.id}`}
                    >
                      {art.file_format.toUpperCase()}
                    </span>
                  )}
                  {art.storage_object_id && (
                    <span
                      className="text-xs text-[var(--text-muted)]"
                      data-testid={`artifacts-file-size-${art.id}`}
                    >
                      📎 File
                    </span>
                  )}
                </div>
                <div className="text-sm font-medium truncate">{art.title}</div>
                {art.description && (
                  <div className="text-xs text-[var(--text-muted)] truncate">{art.description}</div>
                )}
              </div>
              <div className="text-xs text-[var(--text-muted)] ml-2" data-testid={`artifacts-date-${art.id}`}>
                {new Date(art.updated_at).toLocaleDateString()}
              </div>
            </div>
          ))}
        </div>
      )}

      {/* Create modal */}
      {showCreate && (
        <ArtifactEditor
          mode="create"
          onClose={() => { setShowCreate(false); fetchArtifacts(); }}
        />
      )}

      {/* Presentation modal */}
      {showPresentation && (
        <PresentationModal
          onClose={() => setShowPresentation(false)}
          onGenerated={() => { fetchArtifacts(); fetchStorageSummary(); }}
        />
      )}

      {/* Meeting summary modal */}
      {showMeeting && (
        <MeetingSummaryEditor
          mode="create"
          onClose={() => setShowMeeting(false)}
          onSaved={() => fetchArtifacts()}
        />
      )}

      {/* Status report modal */}
      {showStatusReport && (
        <StatusReportModal
          onClose={() => setShowStatusReport(false)}
          onGenerated={() => fetchArtifacts()}
        />
      )}

      {/* Template gallery */}
      {showTemplates && (
        <TemplateGallery
          onClose={() => setShowTemplates(false)}
          onInstantiated={() => fetchArtifacts()}
        />
      )}

      {/* Edit modal */}
      {editingId && (
        <ArtifactEditor
          mode="edit"
          artifactId={editingId}
          onClose={() => { setEditingId(null); fetchArtifacts(); }}
        />
      )}
    </div>
  );
}
