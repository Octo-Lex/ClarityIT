import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import {
  DndContext, DragOverlay, PointerSensor, useSensor, useSensors,
  closestCorners, type DragEndEvent, type DragStartEvent,
} from '@dnd-kit/core';
import { useDroppable, useDraggable } from '@dnd-kit/core';
import { api, type WorkItem } from '@/api/client';
import { keys } from '@/api/keys';
import { useAuth } from '@/auth/context';
import { Card } from '@/components/ui/card';
import { StatusBadge } from '@/components/ui/status-badge';
import { notify } from '@/components/Toaster';
import { CardGridSkeleton, ErrorState, EmptyState } from '@/components/PageState';
import { cn } from '@/lib/utils';

const STATUS_ORDER = ['open', 'in_progress', 'blocked', 'resolved', 'closed'] as const;
type Status = (typeof STATUS_ORDER)[number];

const COLUMN_TONE: Record<Status, string> = {
  open: 'border-t-warning',
  in_progress: 'border-t-info',
  blocked: 'border-t-destructive',
  resolved: 'border-t-success',
  closed: 'border-t-muted-foreground',
};

/** A draggable work-item card. */
function BoardCard({ item, onClick }: { item: WorkItem; onClick: () => void }) {
  const { attributes, listeners, setNodeRef, isDragging } = useDraggable({
    id: item.id,
    data: { item },
  });
  return (
    <div
      ref={setNodeRef}
      {...attributes}
      {...listeners}
      onClick={onClick}
      data-testid={`board-card-${item.id}`}
      className={cn(
        'cursor-grab rounded-md border border-border bg-background p-3 text-left transition-shadow hover:shadow-sm active:cursor-grabbing',
        isDragging && 'opacity-40',
      )}
    >
      <div className="text-sm font-medium">{item.title}</div>
      <div className="mt-1.5 flex items-center gap-2">
        <StatusBadge tone="neutral">{item.work_item_type}</StatusBadge>
        <span className="text-xs text-muted-foreground">{item.priority}</span>
      </div>
    </div>
  );
}

/** A droppable status column. */
function BoardColumn({
  status, items, onCardClick, children,
}: {
  status: Status;
  items: WorkItem[];
  onCardClick: (id: string) => void;
  children?: React.ReactNode;
}) {
  const { setNodeRef, isOver } = useDroppable({ id: status });
  return (
    <div
      ref={setNodeRef}
      data-testid={`board-column-${status}`}
      className={cn(
        'flex min-h-[200px] min-w-[260px] flex-shrink-0 flex-col rounded-xl border border-t-4 bg-card',
        COLUMN_TONE[status],
        isOver && 'ring-2 ring-ring',
      )}
    >
      <div className="flex items-center justify-between border-b border-border px-3 py-2.5">
        <span className="text-sm font-semibold capitalize">{status.replace('_', ' ')}</span>
        <span className="text-xs text-muted-foreground">{items.length}</span>
      </div>
      <div className="flex flex-1 flex-col gap-2 p-2">
        {items.map(item => (
          <BoardCard key={item.id} item={item} onClick={() => onCardClick(item.id)} />
        ))}
        {items.length === 0 && (
          <div className="flex flex-1 items-center justify-center py-6 text-xs text-muted-foreground">
            Drop here
          </div>
        )}
        {children}
      </div>
    </div>
  );
}

export default function BoardPage() {
  const { activeTeamId, hasPermission } = useAuth();
  const nav = useNavigate();
  const queryClient = useQueryClient();
  const [activeId, setActiveId] = useState<string | null>(null);

  const sensors = useSensors(
    useSensor(PointerSensor, { activationConstraint: { distance: 5 } }),
  );

  const { data: board, isPending, error, refetch } = useQuery({
    queryKey: keys.workItems.board(activeTeamId ?? ''),
    queryFn: () => api.getBoard(),
    enabled: !!activeTeamId,
  });

  const canUpdate = hasPermission('work.items.update');

  // Drag → status change. Preserves expected_version for optimistic concurrency;
  // a 409 (someone else moved it) invalidates the board so it refetches.
  const moveMutation = useMutation({
    mutationFn: ({ id, status, version }: { id: string; status: string; version: number }) =>
      api.updateWorkItem(id, { status, expected_version: version }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: keys.workItems.board(activeTeamId ?? '') });
      queryClient.invalidateQueries({ queryKey: keys.workItems.list(activeTeamId ?? '') });
    },
    onError: (err) => {
      notify.mutationError('Move failed', err);
      // Refetch to restore the board to its true server state.
      queryClient.invalidateQueries({ queryKey: keys.workItems.board(activeTeamId ?? '') });
    },
  });

  const itemsByStatus = (status: string) => (board?.[status] ?? []) as WorkItem[];
  const findItem = (id: string): WorkItem | undefined =>
    STATUS_ORDER.flatMap(s => itemsByStatus(s)).find(i => i.id === id);

  const onDragStart = (e: DragStartEvent) => setActiveId(String(e.active.id));
  const onDragEnd = (e: DragEndEvent) => {
    setActiveId(null);
    const { active, over } = e;
    if (!over || !canUpdate) return;
    const item = findItem(String(active.id));
    if (!item) return;
    // Dropped on a column (status) — `over.id` is the status string.
    const targetStatus = String(over.id);
    if (targetStatus === item.status) return;
    moveMutation.mutate({ id: item.id, status: targetStatus, version: item.version });
  };

  if (isPending) {
    return (
      <div className="space-y-4">
        <h1 className="font-heading text-2xl font-semibold tracking-tight">Board</h1>
        <CardGridSkeleton count={5} />
      </div>
    );
  }
  if (error) {
    return (
      <div className="space-y-4">
        <h1 className="font-heading text-2xl font-semibold tracking-tight">Board</h1>
        <ErrorState message="Failed to load board" onRetry={() => refetch()} />
      </div>
    );
  }

  const totalItems = STATUS_ORDER.reduce((n, s) => n + itemsByStatus(s).length, 0);

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h1 className="font-heading text-2xl font-semibold tracking-tight">Board</h1>
        {!canUpdate && (
          <span className="text-xs text-muted-foreground">Read-only — drag disabled</span>
        )}
      </div>

      {totalItems === 0 ? (
        <EmptyState title="Board is empty" description="Work items will appear here grouped by status." />
      ) : (
        <DndContext
          sensors={sensors}
          collisionDetection={closestCorners}
          onDragStart={onDragStart}
          onDragEnd={onDragEnd}
          onDragCancel={() => setActiveId(null)}
        >
          <div className="flex gap-4 overflow-x-auto pb-4">
            {STATUS_ORDER.map(status => (
              <BoardColumn
                key={status}
                status={status}
                items={itemsByStatus(status)}
                onCardClick={(id) => nav(`/objects/${id}`)}
              />
            ))}
          </div>
          <DragOverlay>
            {activeId && findItem(activeId) ? (
              <Card className="w-[240px] p-3 opacity-90 shadow-lg">
                <div className="text-sm font-medium">{findItem(activeId)?.title}</div>
              </Card>
            ) : null}
          </DragOverlay>
        </DndContext>
      )}
    </div>
  );
}
