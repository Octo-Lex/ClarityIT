import { useState, useEffect } from 'react';
import { api, ApiError, type KnowledgeCollection } from '../../api/client';

interface SaveToCollectionDialogProps {
  sourceType: string;
  sourceId: string;
  title?: string;
  knowledgeItemId?: string;
  onClose: () => void;
  onSaved?: () => void;
}

export function SaveToCollectionDialog({ sourceType, sourceId, title, knowledgeItemId, onClose, onSaved }: SaveToCollectionDialogProps) {
  const [collections, setCollections] = useState<KnowledgeCollection[]>([]);
  const [loading, setLoading] = useState(true);
  const [selectedId, setSelectedId] = useState<string>('');
  const [note, setNote] = useState('');
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState(false);
  const [duplicateMsg, setDuplicateMsg] = useState(false);

  useEffect(() => {
    (async () => {
      try {
        const resp = await api.listCollections();
        setCollections(resp.collections);
      } catch {
        setError('Failed to load collections');
      } finally {
        setLoading(false);
      }
    })();
  }, []);

  const handleSave = async () => {
    if (!selectedId) return;
    setSaving(true);
    setError(null);
    setDuplicateMsg(false);
    try {
      const resp = await api.addCollectionItem(selectedId, {
        source_type: sourceType,
        source_id: sourceId,
        knowledge_item_id: knowledgeItemId,
        note: note.trim() || undefined,
      });
      if (resp.duplicate) {
        setDuplicateMsg(true);
      }
      setSuccess(true);
      onSaved?.();
    } catch {
      setError('Failed to save to collection');
    } finally {
      setSaving(false);
    }
  };

  return (
    <div data-testid="save-to-collection-dialog" className="fixed inset-0 bg-black/30 flex items-center justify-center z-50" onClick={onClose}>
      <div className="bg-white rounded-lg p-6 max-w-md w-full mx-4" onClick={(e) => e.stopPropagation()}>
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-lg font-bold">Save to Collection</h2>
          <button onClick={onClose} className="text-gray-400 hover:text-gray-600">✕</button>
        </div>

        {title && <p className="text-gray-500 text-sm mb-4">{title}</p>}

        {loading && <div data-testid="save-dialog-loading" className="text-center py-4 text-gray-500">Loading collections…</div>}

        {error && <div data-testid="save-dialog-error" className="text-red-500 text-sm mb-4">{error}</div>}

        {success ? (
          <div data-testid="save-dialog-success" className="text-center py-4">
            <p className="text-green-600 font-medium">
              {duplicateMsg ? 'Item was already in this collection.' : 'Saved successfully!'}
            </p>
            <button onClick={onClose} className="mt-4 px-4 py-2 border rounded-lg">Done</button>
          </div>
        ) : !loading && collections.length === 0 ? (
          <div data-testid="save-dialog-empty" className="text-center py-4 text-gray-400">
            No collections available. Create one first.
          </div>
        ) : (
          !loading && !error && (
            <>
              <select
                data-testid="save-dialog-select"
                value={selectedId}
                onChange={(e) => setSelectedId(e.target.value)}
                className="w-full px-3 py-2 border rounded-lg mb-3"
              >
                <option value="">Choose a collection…</option>
                {collections.map((c) => (
                  <option key={c.id} value={c.id}>{c.name}</option>
                ))}
              </select>
              <textarea
                data-testid="save-dialog-note"
                placeholder="Add a note (optional)"
                value={note}
                onChange={(e) => setNote(e.target.value)}
                maxLength={1000}
                rows={2}
                className="w-full px-3 py-2 border rounded-lg mb-3"
              />
              <div className="flex gap-2">
                <button
                  data-testid="save-dialog-confirm"
                  onClick={handleSave}
                  disabled={!selectedId || saving}
                  className="px-4 py-2 bg-indigo-600 text-white rounded-lg disabled:opacity-50"
                >
                  {saving ? 'Saving…' : 'Save'}
                </button>
                <button onClick={onClose} className="px-4 py-2 border rounded-lg">Cancel</button>
              </div>
            </>
          )
        )}
      </div>
    </div>
  );
}
