import { useQuery } from '@tanstack/react-query';
import { Building2 } from 'lucide-react';
import { api } from '@/api/client';
import { keys } from '@/api/keys';
import { Card } from '@/components/ui/card';
import { CardGridSkeleton, ErrorState, EmptyState } from '@/components/PageState';

export default function AdminTeams() {
  const { data: teams, isPending, error, refetch } = useQuery({
    queryKey: keys.admin.teams(),
    queryFn: () => api.listTeams(),
  });

  return (
    <div className="space-y-4">
      <h1 className="font-heading text-2xl font-semibold tracking-tight">Teams</h1>
      {isPending ? (
        <CardGridSkeleton count={4} />
      ) : error ? (
        <ErrorState message="Failed to load teams" onRetry={() => refetch()} />
      ) : (teams ?? []).length === 0 ? (
        <EmptyState title="No teams" />
      ) : (
        <div className="grid gap-4 sm:grid-cols-2">
          {(teams ?? []).map(t => (
            <Card key={t.id} className="p-5">
              <div className="flex items-start gap-3">
                <div className="flex size-9 items-center justify-center rounded-lg bg-primary/10 text-primary">
                  <Building2 className="size-5" />
                </div>
                <div className="min-w-0">
                  <h3 className="font-heading font-semibold">{t.name}</h3>
                  <p className="text-sm text-muted-foreground">{t.slug}</p>
                  {(t as { description?: string }).description && (
                    <p className="mt-1 text-sm">{(t as { description?: string }).description}</p>
                  )}
                </div>
              </div>
            </Card>
          ))}
        </div>
      )}
    </div>
  );
}
