import { useState, useEffect } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { api, ApiError, type KnowledgeCollectionDetail } from '../../api/client';
import { KnowledgeSourceBadge } from './KnowledgeSourceBadge';

export function KnowledgeCollectionDetailPage() {
  const { collectionId } = useParams<{ collectionId: string }>();
  const navigate = useNavigate();
  const [detail, setDetail] = useState<KnowledgeCollectionDetail | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [editing, setEditing] = useState(false);
  const [editName, setEditName] = useState('');
  const [editDesc, setEditDesc] = useState('');

  const load = async () => {
    if (!collectionId) return;
    setLoading(true);
    setError(null);
    try {
      const resp = await api.getCollection(collectionId);
      setDetail(resp);
      setEditName(resp.name);
      setEditDesc(resp.description || '');
    } catch {
      setError('Failed to load collection');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => { load(); }, [collectionId]);

  const handleSaveEdit = async () => {
    if (!collectionId || !editName.trim()) return;
    try {
      await api.patchCollection(collectionId, {
        name: editName.trim(),
        description: editDesc.trim() || undefined,
      });
      setEditing(false);
      await load();
    } catch (e) {
      if (e instanceof ApiError && e.status === 409) {
        setError('A collection with this name already exists');
      } else {
        setError('Failed to update collection');
      }
    }
  };

  const handleRemoveItem = async (itemId: string) => {
    if (!collectionId) return;
    try {
      await api.removeCollectionItem(collectionId, itemId);
      await load();
    } catch {
      setError('Failed to remove item');
    }
  };

  if (loading) return <div data-testid="collection-detail-loading" className="p-8 text-center text-gray-500">Loading…</div>;
  if (error) return <div data-testid="collection-detail-error" className="p-8 text-center text-red-500">{error}</div>;
  if (!detail) return <div data-testid="collection-detail-error" className="p-8 text-center text-red-500">Collection not found</div>;

  return (
    <div className="max-w-4xl mx-auto p-6">
      <button onClick={() => navigate('/knowledge/collections')} className="text-indigo-600 mb-4 hover:underline">
        ← Back to Collections
      </button>

      {editing ? (
        <div className="mb-6 p-4 border rounded-lg bg-gray-50">
          <input
            data-testid="edit-name-input"
            type="text"
            value={editName}
            onChange={(e) => setEditName(e.target.value)}
            maxLength={200}
            className="w-full px-3 py-2 mb-2 border rounded-lg"
          />
          <textarea
            data-testid="edit-desc-input"
            value={editDesc}
            onChange={(e) => setEditDesc(e.target.value)}
            maxLength={2000}
            rows={2}
            className="w-full px-3 py-2 mb-2 border rounded-lg"
          />
          <div className="flex gap-2">
            <button data-testid="edit-save-btn" onClick={handleSaveEdit} className="px-4 py-2 bg-indigo-600 text-white rounded-lg">Save</button>
            <button onClick={() => setEditing(false)} className="px-4 py-2 border rounded-lg">Cancel</button>
          </div>
        </div>
      ) : (
        <div className="flex items-center justify-between mb-6">
          <div>
            <h1 className="text-2xl font-bold">{detail.name}</h1>
            {detail.description && <p className="text-gray-500 mt-1">{detail.description}</p>}
          </div>
          <button data-testid="edit-collection-btn" onClick={() => setEditing(true)} className="text-indigo-600 hover:underline">
            Edit
          </button>
        </div>
      )}

      {detail.items.length === 0 ? (
        <div data-testid="collection-items-empty" className="text-center py-12 text-gray-400">
          No items in this collection yet. Save search results or Ask Clarity answers here.
        </div>
      ) : (
        <div className="space-y-3">
          {detail.items.map((item) => (
            <div key={item.id} data-testid="collection-item-card" className="p-4 border rounded-lg flex items-start justify-between">
              <div className="flex-1">
                <div className="flex items-center gap-2 mb-1">
                  <KnowledgeSourceBadge sourceType={item.source_type} />
                  <span className="text-gray-400 text-sm">{item.source_id}</span>
                </div>
                {item.title && <h4 className="font-medium">{item.title}</h4>}
                {item.summary && <p className="text-gray-500 text-sm">{item.summary}</p>}
                {item.note && <p className="text-gray-400 text-sm italic mt-1">"{item.note}"</p>}
              </div>
              <button
                data-testid={`remove-item-${item.id}`}
                onClick={() => handleRemoveItem(item.id)}
                className="text-gray-400 hover:text-red-500 text-sm ml-4"
              >
                Remove
              </button>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
