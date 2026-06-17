import { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { Command } from 'cmdk';
import {
  LayoutDashboard, ListChecks, Kanban, Flame, Bot, FileText, Search,
  FolderClosed, Save, Gauge, ShieldCheck, Users, Building2, ScrollText,
  Plug, CheckCircle2, Settings, Cpu, BarChart3, Server,
  CornerDownLeft,
} from 'lucide-react';

/**
 * ⌘K command palette. Fast keyboard navigation across all routes, replacing
 * the crowded 21-item sidebar as the primary power-user affordance.
 *
 * Mount once inside AppLayout. Toggle with Cmd/Ctrl+K.
 */

interface CmdItem {
  id: string;
  label: string;
  path: string;
  icon: React.ComponentType<{ className?: string }>;
  group: 'Navigate' | 'Admin' | 'Account';
  /** Optional permission gate; hidden if the user lacks it. */
  perm?: string;
  ownerOnly?: boolean;
}

const NAV_ITEMS: CmdItem[] = [
  { id: 'dashboard', label: 'Dashboard', path: '/', icon: LayoutDashboard, group: 'Navigate' },
  { id: 'queue', label: 'Queue', path: '/queue', icon: ListChecks, group: 'Navigate', perm: 'work.items.list' },
  { id: 'board', label: 'Board', path: '/board', icon: Kanban, group: 'Navigate', perm: 'work.items.list' },
  { id: 'incidents', label: 'Incidents', path: '/incidents', icon: Flame, group: 'Navigate', perm: 'incidents.list' },
  { id: 'agents', label: 'Agents', path: '/agents', icon: Bot, group: 'Navigate', perm: 'agents.read' },
  { id: 'documents', label: 'Documents', path: '/artifacts', icon: FileText, group: 'Navigate', perm: 'artifacts.read' },
  { id: 'knowledge', label: 'Knowledge Search', path: '/knowledge', icon: Search, group: 'Navigate', perm: 'knowledge.search' },
  { id: 'collections', label: 'Collections', path: '/knowledge/collections', icon: FolderClosed, group: 'Navigate', perm: 'knowledge.collections.read' },
  { id: 'saved-answers', label: 'Saved Answers', path: '/knowledge/saved-answers', icon: Save, group: 'Navigate', perm: 'knowledge.collections.read' },
  { id: 'quality', label: 'Knowledge Quality', path: '/knowledge/quality', icon: Gauge, group: 'Navigate', perm: 'knowledge.read' },
  { id: 'security', label: 'Security & MFA', path: '/account/security', icon: ShieldCheck, group: 'Account' },
  { id: 'team', label: 'Team Settings', path: '/settings/team', icon: Users, group: 'Account', perm: 'team.settings.view' },
];

const ADMIN_ITEMS: CmdItem[] = [
  { id: 'approvals', label: 'Approvals', path: '/admin/approvals', icon: CheckCircle2, group: 'Admin', ownerOnly: true },
  { id: 'asset-actions', label: 'Asset Ops', path: '/admin/asset-actions', icon: Server, group: 'Admin', ownerOnly: true },
  { id: 'users', label: 'Users', path: '/admin/users', icon: Users, group: 'Admin', ownerOnly: true },
  { id: 'teams', label: 'Teams', path: '/admin/teams', icon: Building2, group: 'Admin', ownerOnly: true },
  { id: 'audit', label: 'Audit Log', path: '/admin/audit', icon: ScrollText, group: 'Admin', ownerOnly: true },
  { id: 'integrations', label: 'Integrations', path: '/admin/integrations', icon: Plug, group: 'Admin', ownerOnly: true },
  { id: 'ops', label: 'Ops Dashboard', path: '/admin/ops', icon: Cpu, group: 'Admin', ownerOnly: true },
  { id: 'metrics', label: 'Metrics', path: '/admin/metrics', icon: BarChart3, group: 'Admin', ownerOnly: true },
  { id: 'setup', label: 'Setup Checklist', path: '/admin/setup', icon: Settings, group: 'Admin', ownerOnly: true },
];

export function CommandPalette({
  open,
  onOpenChange,
  hasPermission,
  isPlatformOwner,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  hasPermission: (perm: string) => boolean;
  isPlatformOwner: boolean;
}) {
  const nav = useNavigate();

  // Global Cmd/Ctrl+K toggle (defensive — also bound in AppLayout)
  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === 'k') {
        e.preventDefault();
        onOpenChange(!open);
      }
    };
    window.addEventListener('keydown', handler);
    return () => window.removeEventListener('keydown', handler);
  }, [open, onOpenChange]);

  // Close on Escape is handled by cmdk; also reset on close
  useEffect(() => {
    if (!open) return;
    const onEsc = (e: KeyboardEvent) => { if (e.key === 'Escape') onOpenChange(false); };
    window.addEventListener('keydown', onEsc);
    return () => window.removeEventListener('keydown', onEsc);
  }, [open, onOpenChange]);

  if (!open) return null;

  const visible = (item: CmdItem) =>
    (!item.perm || hasPermission(item.perm)) && (!item.ownerOnly || isPlatformOwner);

  const allItems = [...NAV_ITEMS, ...ADMIN_ITEMS].filter(visible);
  const groups = Array.from(new Set(allItems.map(i => i.group)));

  const run = (path: string) => {
    nav(path);
    onOpenChange(false);
  };

  return (
    <div
      className="fixed inset-0 z-50 flex items-start justify-center bg-black/40 pt-[15vh]"
      onClick={() => onOpenChange(false)}
      data-testid="command-palette"
    >
      <Command
        loop
        label="Command Palette"
        className="w-full max-w-xl overflow-hidden rounded-xl border border-border bg-popover text-popover-foreground shadow-lg"
        onClick={(e) => e.stopPropagation()}
      >
        <Command.Input
          autoFocus
          placeholder="Search pages and actions…"
          data-testid="command-input"
          className="w-full border-b border-border bg-transparent px-4 py-3 text-sm outline-none placeholder:text-muted-foreground"
        />
        <Command.List className="max-h-[50vh] overflow-auto p-2">
          <Command.Empty className="p-4 text-center text-sm text-muted-foreground">
            No results found.
          </Command.Empty>
          {groups.map(group => (
            <Command.Group key={group} heading={group} className="text-xs font-medium text-muted-foreground [&_[cmdk-group-heading]]:px-2 [&_[cmdk-group-heading]]:py-1.5">
              {allItems.filter(i => i.group === group).map(item => {
                const Icon = item.icon;
                return (
                  <Command.Item
                    key={item.id}
                    value={`${item.label} ${item.group}`}
                    onSelect={() => run(item.path)}
                    className="flex cursor-pointer items-center gap-2.5 rounded-md px-2 py-2 text-sm outline-none data-[selected=true]:bg-accent data-[selected=true]:text-accent-foreground"
                  >
                    <Icon className="size-4 text-muted-foreground" />
                    <span className="flex-1">{item.label}</span>
                    <CornerDownLeft className="size-3 text-muted-foreground/50" />
                  </Command.Item>
                );
              })}
            </Command.Group>
          ))}
        </Command.List>
      </Command>
    </div>
  );
}
