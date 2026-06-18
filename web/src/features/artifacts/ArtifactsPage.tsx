import { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { api, ApiError } from '../../api/client';
import ArtifactEditor from './ArtifactEditor';
import PresentationModal from './PresentationModal';
import MeetingSummaryEditor from './MeetingSummaryEditor';
import StatusReportModal from './StatusReportModal';
import TemplateGallery from './TemplateGallery';
import DocumentGenerateModal from './DocumentGenerateModal';
import { RelatedKnowledgePanel } from '../knowledge/RelatedKnowledgePanel';

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
  document: 'badge badge-blue',
  report: 'badge badge-green',
  presentation: 'badge badge-blue',
  meeting_summary: 'badge badge-yellow',
  status_report: 'badge badge-blue',
  decision_memo: 'badge badge-yellow',
  training_deck: 'badge badge-blue',
};

const STATUS_COLORS: Record<string, string> = {
  draft: 'badge badge-gray',
  published: 'badge badge-green',
  archived: 'badge badge-red',
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
  const [showGenerate, setShowGenerate] = useState(false);
  const [editingId, setEditingId] = useState<string | null>(null);
  const navigate = useNavigate();
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
        <div className="flex flex-wrap gap-2">
          <button
            onClick={() => setShowPresentation(true)}
            className="btn-secondary text-sm"
            data-testid="artifacts-generate-presentation-btn"
          >
            Generate Presentation
          </button>
          <button
            onClick={() => setShowMeeting(true)}
            className="btn-secondary text-sm"
            data-testid="artifacts-new-meeting-btn"
          >
            Meeting Summary
          </button>
          <button
            onClick={() => setShowStatusReport(true)}
            className="btn-secondary text-sm"
            data-testid="artifacts-status-report-btn"
          >
            Status Report
          </button>
          <button
            onClick={() => setShowGenerate(true)}
            className="btn-secondary text-sm"
            data-testid="artifacts-generate-doc-btn"
          >
            Generate Document
          </button>
          <button
            onClick={() => setShowTemplates(true)}
            className="btn-secondary text-sm"
            data-testid="artifacts-templates-btn"
          >
            Templates
          </button>
          <button
            onClick={() => setShowCreate(true)}
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
              <span className="px-1.5 py-0.5 text-xs rounded badge badge-gray">{fmt}: {count}</span>
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
          className="px-3 py-1 btn-secondary text-sm disabled:opacity-50"
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
                    className={`px-1.5 py-0.5 text-xs rounded ${TYPE_COLORS[art.artifact_type] || 'badge badge-gray'}`}
                    data-testid={`artifacts-type-${art.id}`}
                  >
                    {art.artifact_type.replace(/_/g, ' ')}
                  </span>
                  <span
                    className={`px-1.5 py-0.5 text-xs rounded ${STATUS_COLORS[art.status] || 'badge badge-gray'}`}
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

      {/* v1.4 Track 4: Generate Document modal */}
      {showGenerate && (
        <DocumentGenerateModal
          onClose={() => setShowGenerate(false)}
          onGenerated={(artifactId) => {
            setShowGenerate(false);
            navigate(`/artifacts/documents/${artifactId}`);
          }}
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

      {/* v1.5 Track 4: Related Knowledge Panel */}
      {recentArtifacts && recentArtifacts.length > 0 && recentArtifacts[0]?.id && (
        <RelatedKnowledgePanel
          sourceType="artifact"
          sourceId={recentArtifacts[0].id}
        />
      )}
    </div>
  );
}
