import { ReactNode } from 'react';
import { AlertCircle, Inbox } from 'lucide-react';
import { Skeleton } from '@/components/ui/skeleton';

/**
 * Composable page-state primitives. Every list/detail page should render one of
 * these for its non-happy paths, replacing the hand-rolled
 * `loading ? 'Loading...' : ...` and `.catch(() => {})` patterns.
 *
 * The loading skeletons use React Query's isLoading/isFetching distinction:
 * pass `isPending` for the first-load skeleton, and let stale data keep showing
 * during background refetches (no full-screen flicker).
 */

/** Full-width table skeleton — N rows of shimmering bars. */
export function TableSkeleton({ rows = 5, cols = 4 }: { rows?: number; cols?: number }) {
  return (
    <div className="space-y-2">
      {Array.from({ length: rows }).map((_, r) => (
        <div key={r} className="flex gap-4">
          {Array.from({ length: cols }).map((_, c) => (
            <Skeleton key={c} className="h-8 flex-1" />
          ))}
        </div>
      ))}
    </div>
  );
}

/** Card-grid skeleton — N cards. */
export function CardGridSkeleton({ count = 6 }: { count?: number }) {
  return (
    <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
      {Array.from({ length: count }).map((_, i) => (
        <div key={i} className="space-y-3 rounded-xl border border-border p-4">
          <Skeleton className="h-5 w-2/3" />
          <Skeleton className="h-4 w-full" />
          <Skeleton className="h-4 w-1/2" />
        </div>
      ))}
    </div>
  );
}

/** Generic inline error card for a failed fetch. */
export function ErrorState({
  message = 'Failed to load',
  onRetry,
}: {
  message?: string;
  onRetry?: () => void;
}) {
  return (
    <div
      role="alert"
      data-testid="page-error"
      className="flex flex-col items-center justify-center gap-3 rounded-xl border border-destructive/30 bg-destructive/5 p-8 text-center"
    >
      <AlertCircle className="size-8 text-destructive" />
      <div className="space-y-1">
        <p className="text-sm font-medium text-foreground">{message}</p>
        <p className="text-xs text-muted-foreground">Please try again.</p>
      </div>
      {onRetry && (
        <button
          type="button"
          onClick={onRetry}
          className="rounded-md border border-border bg-background px-3 py-1.5 text-sm font-medium text-foreground hover:bg-muted"
        >
          Retry
        </button>
      )}
    </div>
  );
}

/** Empty-state card shown when a list has no items. */
export function EmptyState({
  title = 'Nothing here yet',
  description,
  action,
  icon,
}: {
  title?: string;
  description?: string;
  action?: ReactNode;
  icon?: ReactNode;
}) {
  return (
    <div
      data-testid="page-empty"
      className="flex flex-col items-center justify-center gap-3 rounded-xl border border-dashed border-border p-12 text-center"
    >
      <div className="flex size-10 items-center justify-center rounded-full bg-muted text-muted-foreground">
        {icon ?? <Inbox className="size-5" />}
      </div>
      <div className="space-y-1">
        <p className="text-sm font-medium text-foreground">{title}</p>
        {description && <p className="text-xs text-muted-foreground">{description}</p>}
      </div>
      {action}
    </div>
  );
}

/** Centered spinner for inline loading (e.g. a button or small panel). */
export function InlineSpinner({ label = 'Loading…' }: { label?: string }) {
  return (
    <div data-testid="page-loading" className="flex items-center justify-center gap-2 p-8 text-sm text-muted-foreground">
      <span className="size-4 animate-spin rounded-full border-2 border-muted border-t-foreground" />
      {label}
    </div>
  );
}
