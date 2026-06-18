import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { X } from 'lucide-react';
import { api } from '@/api/client';
import { keys } from '@/api/keys';
import { useAuth } from '@/auth/context';
import { Button } from '@/components/ui/button';
import { Textarea } from '@/components/ui/textarea';
import { InlineSpinner } from '@/components/PageState';

interface SaveToCollectionDialogProps {
  sourceType: string;
  sourceId: string;
  title?: string;
  knowledgeItemId?: string;
  onClose: () => void;
  onSaved?: () => void;
}

export function SaveToCollectionDialog({ sourceType, sourceId, title, knowledgeItemId, onClose, onSaved }: SaveToCollectionDialogProps) {
  const { activeTeamId } = useAuth();
  const queryClient = useQueryClient();
  const [selectedId, setSelectedId] = useState('');
  const [note, setNote] = useState('');
  const [duplicateMsg, setDuplicateMsg] = useState(false);

  const collectionsQ = useQuery({
    queryKey: keys.knowledge.collections.list(activeTeamId ?? ''),
    queryFn: () => api.listCollections(),
  });

  const saveMut = useMutation({
    mutationFn: () => api.addCollectionItem(selectedId, {
      source_type: sourceType,
      source_id: sourceId,
      knowledge_item_id: knowledgeItemId,
      note: note.trim() || undefined,
    }),
    onSuccess: (resp) => {
      if (resp.duplicate) setDuplicateMsg(true);
      queryClient.invalidateQueries({ queryKey: keys.knowledge.collections.list(activeTeamId ?? '') });
      onSaved?.();
    },
  });

  const collections = collectionsQ.data?.collections ?? [];
  const loading = collectionsQ.isPending;
  const error = collectionsQ.isError || saveMut.isError;
  const success = saveMut.isSuccess;

  return (
    <div
      data-testid="save-to-collection-dialog"
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/40"
      onClick={onClose}
    >
      <div
        className="mx-4 w-full max-w-md rounded-lg border border-border bg-popover p-6 text-popover-foreground"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="mb-4 flex items-center justify-between">
          <h2 className="font-heading text-lg font-semibold">Save to Collection</h2>
          <button onClick={onClose} className="rounded p-1 text-muted-foreground hover:bg-accent"><X className="size-4" /></button>
        </div>

        {title && <p className="mb-4 text-sm text-muted-foreground">{title}</p>}

        {loading && <div data-testid="save-dialog-loading"><InlineSpinner label="Loading collections…" /></div>}

        {error && !loading && !success && (
          <div data-testid="save-dialog-error" className="mb-4 text-sm text-destructive">
            {collectionsQ.isError ? 'Failed to load collections' : 'Failed to save to collection'}
          </div>
        )}

        {success ? (
          <div data-testid="save-dialog-success" className="py-4 text-center">
            <p className="font-medium text-success">
              {duplicateMsg ? 'Item was already in this collection.' : 'Saved successfully!'}
            </p>
            <Button className="mt-4" variant="secondary" onClick={onClose}>Done</Button>
          </div>
        ) : !loading && collections.length === 0 ? (
          <div data-testid="save-dialog-empty" className="py-4 text-center text-sm text-muted-foreground">
            No collections available. Create one first.
          </div>
        ) : (
          !loading && !collectionsQ.isError && (
            <>
              <select
                data-testid="save-dialog-select"
                value={selectedId}
                onChange={(e) => setSelectedId(e.target.value)}
                className="mb-3 w-full rounded-md border border-border bg-background px-3 py-2 text-sm"
              >
                <option value="">Choose a collection…</option>
                {collections.map((c) => (
                  <option key={c.id} value={c.id}>{c.name}</option>
                ))}
              </select>
              <Textarea
                data-testid="save-dialog-note"
                placeholder="Add a note (optional)"
                value={note}
                onChange={(e) => setNote(e.target.value)}
                maxLength={1000}
                rows={2}
                className="mb-3"
              />
              <div className="flex gap-2">
                <Button
                  data-testid="save-dialog-confirm"
                  onClick={() => saveMut.mutate()}
                  disabled={!selectedId || saveMut.isPending}
                >
                  {saveMut.isPending ? 'Saving…' : 'Save'}
                </Button>
                <Button variant="secondary" onClick={onClose}>Cancel</Button>
              </div>
            </>
          )
        )}
      </div>
    </div>
  );
}
