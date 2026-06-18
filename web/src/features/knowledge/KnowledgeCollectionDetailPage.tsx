import { useState, useEffect } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { ArrowLeft, Trash2 } from 'lucide-react';
import { api, ApiError } from '@/api/client';
import { keys } from '@/api/keys';
import { useAuth } from '@/auth/context';
import { Card } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Textarea } from '@/components/ui/textarea';
import { notify } from '@/components/Toaster';
import { InlineSpinner, ErrorState, EmptyState } from '@/components/PageState';
import { KnowledgeSourceBadge } from './KnowledgeSourceBadge';

export function KnowledgeCollectionDetailPage() {
  const { collectionId } = useParams<{ collectionId: string }>();
  const navigate = useNavigate();
  const { activeTeamId } = useAuth();
  const queryClient = useQueryClient();
  const [editing, setEditing] = useState(false);
  const [editName, setEditName] = useState('');
  const [editDesc, setEditDesc] = useState('');
  const [editError, setEditError] = useState<string | null>(null);

  const detailQ = useQuery({
    queryKey: keys.knowledge.collections.detail(activeTeamId ?? '', collectionId ?? ''),
    queryFn: () => api.getCollection(collectionId!),
    enabled: !!collectionId,
  });

  // Seed the edit fields once the detail loads.
  useEffect(() => {
    if (detailQ.data) { setEditName(detailQ.data.name); setEditDesc(detailQ.data.description || ''); }
  }, [detailQ.data]);

  const invalidate = () => {
    queryClient.invalidateQueries({ queryKey: keys.knowledge.collections.detail(activeTeamId ?? '', collectionId ?? '') });
    queryClient.invalidateQueries({ queryKey: keys.knowledge.collections.list(activeTeamId ?? '') });
  };

  const saveEditMut = useMutation({
    mutationFn: () => api.patchCollection(collectionId!, { name: editName.trim(), description: editDesc.trim() || undefined }),
    onSuccess: () => { setEditing(false); setEditError(null); invalidate(); notify.success('Collection updated'); },
    onError: (err) => {
      setEditError(err instanceof ApiError && err.status === 409 ? 'A collection with this name already exists' : 'Failed to update collection');
    },
  });

  const removeItemMut = useMutation({
    mutationFn: (itemId: string) => api.removeCollectionItem(collectionId!, itemId),
    onSuccess: () => { invalidate(); notify.success('Item removed'); },
    onError: () => notify.error('Failed to remove item'),
  });

  if (detailQ.isPending) return <div data-testid="collection-detail-loading"><InlineSpinner /></div>;
  if (detailQ.isError) return <div data-testid="collection-detail-error" className="p-4"><ErrorState message="Failed to load collection" onRetry={() => detailQ.refetch()} /></div>;
  const detail = detailQ.data;
  if (!detail) return <div data-testid="collection-detail-error"><ErrorState message="Collection not found" /></div>;

  return (
    <div className="mx-auto max-w-4xl space-y-6">
      <button onClick={() => navigate('/knowledge/collections')} className="inline-flex items-center gap-1 text-sm text-muted-foreground hover:text-foreground">
        <ArrowLeft className="size-4" /> Back to Collections
      </button>

      {editing ? (
        <Card className="space-y-3 p-4">
          <Input data-testid="edit-name-input" type="text" value={editName} onChange={(e) => setEditName(e.target.value)} maxLength={200} />
          <Textarea data-testid="edit-desc-input" value={editDesc} onChange={(e) => setEditDesc(e.target.value)} maxLength={2000} rows={2} />
          {editError && <p className="text-sm text-destructive">{editError}</p>}
          <div className="flex gap-2">
            <Button data-testid="edit-save-btn" onClick={() => saveEditMut.mutate()} disabled={saveEditMut.isPending || !editName.trim()}>
              {saveEditMut.isPending ? 'Saving…' : 'Save'}
            </Button>
            <Button variant="secondary" onClick={() => setEditing(false)}>Cancel</Button>
          </div>
        </Card>
      ) : (
        <div className="flex items-center justify-between">
          <div>
            <h1 className="font-heading text-2xl font-semibold tracking-tight">{detail.name}</h1>
            {detail.description && <p className="mt-1 text-sm text-muted-foreground">{detail.description}</p>}
          </div>
          <Button variant="secondary" size="sm" data-testid="edit-collection-btn" onClick={() => setEditing(true)}>Edit</Button>
        </div>
      )}

      {detail.items.length === 0 ? (
        <div data-testid="collection-items-empty">
          <EmptyState title="No items in this collection" description="Save search results or Ask Clarity answers here." />
        </div>
      ) : (
        <div className="space-y-3">
          {detail.items.map((item) => (
            <Card key={item.id} data-testid="collection-item-card" className="flex items-start justify-between p-4">
              <div className="flex-1">
                <div className="mb-1 flex items-center gap-2">
                  <KnowledgeSourceBadge sourceType={item.source_type} />
                  <span className="text-sm text-muted-foreground">{item.source_id}</span>
                </div>
                {item.title && <h4 className="font-medium">{item.title}</h4>}
                {item.summary && <p className="text-sm text-muted-foreground">{item.summary}</p>}
                {item.note && <p className="mt-1 text-sm italic text-muted-foreground">“{item.note}”</p>}
              </div>
              <Button
                variant="ghost"
                data-testid={`remove-item-${item.id}`}
                onClick={() => removeItemMut.mutate(item.id)}
                className="ml-4 text-muted-foreground hover:text-destructive"
              >
                <Trash2 className="size-4" /> Remove
              </Button>
            </Card>
          ))}
        </div>
      )}
    </div>
  );
}
