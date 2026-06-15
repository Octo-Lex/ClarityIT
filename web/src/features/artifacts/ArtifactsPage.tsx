import { useState, useEffect } from 'react';
import { api, ApiError } from '../../api/client';
import ArtifactEditor from './ArtifactEditor';
import PresentationModal from './PresentationModal';

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

export default function ArtifactsPage() {
  const [artifacts, setArtifacts] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [typeFilter, setTypeFilter] = useState('');
  const [search, setSearch] = useState('');
  const [showCreate, setShowCreate] = useState(false);
  const [showPresentation, setShowPresentation] = useState(false);
  const [editingId, setEditingId] = useState<string | null>(null);

  const fetchArtifacts = () => {
    setLoading(true);
    api.listArtifacts({ type: typeFilter || undefined, q: search || undefined })
      .then((data: any[]) => { setArtifacts(data || []); setLoading(false); })
      .catch((e: unknown) => {
        if (e instanceof ApiError) setError(e.message);
        else setError('Failed to load artifacts');
        setLoading(false);
      });
  };

  useEffect(() => { fetchArtifacts(); }, [typeFilter]);

  const handleSearch = () => { fetchArtifacts(); };

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
            onClick={() => setShowCreate(true)}
            className="px-3 py-1.5 bg-blue-600 text-white rounded text-sm hover:bg-blue-700"
            data-testid="artifacts-create-btn"
          >
            + New Artifact
          </button>
        </div>
      </div>

      {/* Filters */}
      <div className="flex gap-3 items-center">
        <select
          value={typeFilter}
          onChange={(e) => setTypeFilter(e.target.value)}
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
          placeholder="Search by title..."
          className="bg-[var(--bg-input)] border border-[var(--border)] rounded px-3 py-1 text-sm flex-1"
          data-testid="artifacts-search"
        />
        <button
          onClick={handleSearch}
          className="px-3 py-1 bg-gray-700 rounded text-sm"
          data-testid="artifacts-search-btn"
        >
          Search
        </button>
      </div>

      {/* Empty state */}
      {artifacts.length === 0 && !error && (
        <div className="card p-8 text-center" data-testid="artifacts-empty">
          <p className="text-[var(--text-muted)]">No artifacts yet. Create your first one!</p>
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
                </div>
                <div className="text-sm font-medium truncate">{art.title}</div>
                {art.description && (
                  <div className="text-xs text-[var(--text-muted)] truncate">{art.description}</div>
                )}
              </div>
              <div className="text-xs text-[var(--text-muted)] ml-2">
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
          onGenerated={() => fetchArtifacts()}
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
