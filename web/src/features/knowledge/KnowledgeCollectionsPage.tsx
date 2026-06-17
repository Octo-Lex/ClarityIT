import { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { api, ApiError, type KnowledgeCollection } from '../../api/client';

export function KnowledgeCollectionsPage() {
  const navigate = useNavigate();
  const [collections, setCollections] = useState<KnowledgeCollection[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [showCreate, setShowCreate] = useState(false);
  const [newName, setNewName] = useState('');
  const [newDesc, setNewDesc] = useState('');
  const [creating, setCreating] = useState(false);

  const load = async () => {
    setLoading(true);
    setError(null);
    try {
      const resp = await api.listCollections();
      setCollections(resp.collections);
    } catch (e) {
      setError('Failed to load collections');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => { load(); }, []);

  const handleCreate = async () => {
    if (!newName.trim()) return;
    setCreating(true);
    try {
      await api.createCollection(newName.trim(), newDesc.trim() || undefined);
      setNewName('');
      setNewDesc('');
      setShowCreate(false);
      await load();
    } catch (e) {
      if (e instanceof ApiError && e.status === 409) {
        setError('A collection with this name already exists');
      } else {
        setError('Failed to create collection');
      }
    } finally {
      setCreating(false);
    }
  };

  const handleArchive = async (id: string) => {
    try {
      await api.deleteCollection(id);
      await load();
    } catch {
      setError('Failed to archive collection');
    }
  };

  if (loading) return <div data-testid="collections-loading" className="p-8 text-center text-gray-500">Loading collections…</div>;
  if (error) return <div data-testid="collections-error" className="p-8 text-center text-red-500">{error}</div>;

  return (
    <div className="max-w-4xl mx-auto p-6">
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-2xl font-bold">Knowledge Collections</h1>
        <button
          data-testid="create-collection-btn"
          onClick={() => setShowCreate(!showCreate)}
          className="px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700"
        >
          New Collection
        </button>
      </div>

      {showCreate && (
        <div data-testid="create-collection-dialog" className="mb-6 p-4 border rounded-lg bg-gray-50">
          <input
            data-testid="collection-name-input"
            type="text"
            placeholder="Collection name"
            value={newName}
            onChange={(e) => setNewName(e.target.value)}
            maxLength={200}
            className="w-full px-3 py-2 mb-2 border rounded-lg"
          />
          <textarea
            data-testid="collection-desc-input"
            placeholder="Description (optional)"
            value={newDesc}
            onChange={(e) => setNewDesc(e.target.value)}
            maxLength={2000}
            rows={2}
            className="w-full px-3 py-2 mb-2 border rounded-lg"
          />
          <div className="flex gap-2">
            <button
              data-testid="collection-create-confirm"
              onClick={handleCreate}
              disabled={!newName.trim() || creating}
              className="px-4 py-2 bg-indigo-600 text-white rounded-lg disabled:opacity-50"
            >
              Create
            </button>
            <button
              onClick={() => setShowCreate(false)}
              className="px-4 py-2 border rounded-lg"
            >
              Cancel
            </button>
          </div>
        </div>
      )}

      {collections.length === 0 ? (
        <div data-testid="collections-empty" className="text-center py-16 text-gray-400">
          No collections yet. Create one to organize important knowledge.
        </div>
      ) : (
        <div className="space-y-3">
          {collections.map((c) => (
            <div
              key={c.id}
              data-testid="collection-card"
              className="p-4 border rounded-lg hover:border-indigo-300 cursor-pointer flex items-center justify-between"
              onClick={() => navigate(`/knowledge/collections/${c.id}`)}
            >
              <div>
                <h3 className="font-semibold text-lg">{c.name}</h3>
                {c.description && <p className="text-gray-500 text-sm">{c.description}</p>}
                <p className="text-gray-400 text-xs mt-1">{c.item_count} item{c.item_count !== 1 ? 's' : ''}</p>
              </div>
              <button
                data-testid={`archive-collection-${c.id}`}
                onClick={(e) => { e.stopPropagation(); handleArchive(c.id); }}
                className="text-gray-400 hover:text-red-500 text-sm"
              >
                Archive
              </button>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
