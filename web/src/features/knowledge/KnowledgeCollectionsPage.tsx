import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Plus, Archive } from 'lucide-react';
import { api, ApiError } from '@/api/client';
import { keys } from '@/api/keys';
import { useAuth } from '@/auth/context';
import { Card } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Textarea } from '@/components/ui/textarea';
import { notify } from '@/components/Toaster';
import { TableSkeleton, ErrorState, EmptyState } from '@/components/PageState';

export function KnowledgeCollectionsPage() {
  const navigate = useNavigate();
  const { activeTeamId } = useAuth();
  const queryClient = useQueryClient();
  const [showCreate, setShowCreate] = useState(false);
  const [newName, setNewName] = useState('');
  const [newDesc, setNewDesc] = useState('');
  const [createError, setCreateError] = useState<string | null>(null);

  const collectionsQ = useQuery({
    queryKey: keys.knowledge.collections.list(activeTeamId ?? ''),
    queryFn: () => api.listCollections(),
  });

  const invalidate = () => queryClient.invalidateQueries({ queryKey: keys.knowledge.collections.list(activeTeamId ?? '') });

  const createMut = useMutation({
    mutationFn: () => api.createCollection(newName.trim(), newDesc.trim() || undefined),
    onSuccess: () => {
      setNewName(''); setNewDesc(''); setShowCreate(false); setCreateError(null);
      invalidate();
      notify.success('Collection created');
    },
    onError: (err) => {
      if (err instanceof ApiError && err.status === 409) {
        setCreateError('A collection with this name already exists');
      } else {
        setCreateError('Failed to create collection');
      }
    },
  });

  const archiveMut = useMutation({
    mutationFn: (id: string) => api.deleteCollection(id),
    onSuccess: () => { invalidate(); notify.success('Collection archived'); },
    onError: () => notify.error('Failed to archive collection'),
  });

  if (collectionsQ.isPending) {
    return <div data-testid="collections-loading" className="p-4"><TableSkeleton rows={4} cols={1} /></div>;
  }
  if (collectionsQ.isError) {
    return <div data-testid="collections-error" className="p-4"><ErrorState message="Failed to load collections" onRetry={() => collectionsQ.refetch()} /></div>;
  }

  const collections = collectionsQ.data?.collections ?? [];

  return (
    <div className="mx-auto max-w-4xl space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="font-heading text-2xl font-semibold tracking-tight">Knowledge Collections</h1>
        <Button data-testid="create-collection-btn" onClick={() => setShowCreate(!showCreate)}>
          <Plus className="size-4" /> New Collection
        </Button>
      </div>

      {showCreate && (
        <Card data-testid="create-collection-dialog" className="space-y-3 p-4">
          <Input
            data-testid="collection-name-input"
            type="text"
            placeholder="Collection name"
            value={newName}
            onChange={(e) => setNewName(e.target.value)}
            maxLength={200}
          />
          <Textarea
            data-testid="collection-desc-input"
            placeholder="Description (optional)"
            value={newDesc}
            onChange={(e) => setNewDesc(e.target.value)}
            maxLength={2000}
            rows={2}
          />
          {createError && <p className="text-sm text-destructive">{createError}</p>}
          <div className="flex gap-2">
            <Button data-testid="collection-create-confirm" onClick={() => createMut.mutate()} disabled={!newName.trim() || createMut.isPending}>
              {createMut.isPending ? 'Creating…' : 'Create'}
            </Button>
            <Button variant="secondary" onClick={() => setShowCreate(false)}>Cancel</Button>
          </div>
        </Card>
      )}

      {collections.length === 0 ? (
        <div data-testid="collections-empty">
          <EmptyState title="No collections yet" description="Create one to organize important knowledge." />
        </div>
      ) : (
        <div className="space-y-3">
          {collections.map((c) => (
            <Card
              key={c.id}
              data-testid="collection-card"
              className="flex cursor-pointer items-center justify-between p-4 transition-colors hover:border-primary/50"
              onClick={() => navigate(`/knowledge/collections/${c.id}`)}
            >
              <div>
                <h3 className="font-heading text-lg font-semibold">{c.name}</h3>
                {c.description && <p className="text-sm text-muted-foreground">{c.description}</p>}
                <p className="mt-1 text-xs text-muted-foreground">{c.item_count} item{c.item_count !== 1 ? 's' : ''}</p>
              </div>
              <Button
                variant="ghost"
                data-testid={`archive-collection-${c.id}`}
                onClick={(e) => { e.stopPropagation(); archiveMut.mutate(c.id); }}
                className="text-muted-foreground hover:text-destructive"
              >
                <Archive className="size-4" /> Archive
              </Button>
            </Card>
          ))}
        </div>
      )}
    </div>
  );
}
