import { useState, useEffect } from 'react';
import { useNavigate, useLocation, Outlet } from 'react-router-dom';
import {
  LayoutDashboard, ListChecks, Kanban, Flame, Bot, FileText, Search,
  FolderClosed, Save, Gauge, ShieldCheck, Users, Building2, ScrollText,
  Plug, Cpu, BarChart3, Settings, Menu, X, LogOut, Sun, Moon, Command as CommandIcon,
} from 'lucide-react';
import { useAuth } from '@/auth/context';
import { Perm, type Permission } from '@/auth/permissions';
import { useRealtimeConnected } from '@/hooks/useRealtimeInvalidation';
import { useTheme } from '@/components/theme/ThemeProvider';
import { CommandPalette } from '@/components/CommandPalette';
import {
  DropdownMenu, DropdownMenuTrigger, DropdownMenuContent,
  DropdownMenuItem, DropdownMenuSeparator, DropdownMenuLabel,
} from '@/components/ui/dropdown-menu';
import { Avatar, AvatarFallback } from '@/components/ui/avatar';
import { cn } from '@/lib/utils';

interface NavItem {
  path: string;
  label: string;
  icon: React.ComponentType<{ className?: string }>;
  perm?: Permission;
}

const PRIMARY_NAV: NavItem[] = [
  { path: '/', label: 'Dashboard', icon: LayoutDashboard },
  { path: '/queue', label: 'Queue', icon: ListChecks, perm: Perm.WorkItemsView },
  { path: '/board', label: 'Board', icon: Kanban, perm: Perm.WorkItemsView },
  { path: '/incidents', label: 'Incidents', icon: Flame, perm: Perm.IncidentsRead },
  { path: '/agents', label: 'Agents', icon: Bot, perm: Perm.AgentsRead },
  { path: '/artifacts', label: 'Documents', icon: FileText, perm: Perm.ArtifactsRead },
  { path: '/knowledge', label: 'Knowledge', icon: Search, perm: Perm.KnowledgeSearch },
  { path: '/knowledge/collections', label: 'Collections', icon: FolderClosed, perm: Perm.KnowledgeCollectionsRead },
  { path: '/knowledge/saved-answers', label: 'Saved Answers', icon: Save, perm: Perm.KnowledgeCollectionsRead },
  { path: '/knowledge/quality', label: 'Quality', icon: Gauge, perm: Perm.KnowledgeRead },
];

const ACCOUNT_NAV: NavItem[] = [
  { path: '/account/security', label: 'Security', icon: ShieldCheck },
  { path: '/settings/team', label: 'Team', icon: Users, perm: Perm.TeamSettingsRead },
];

const ADMIN_NAV: NavItem[] = [
  { path: '/admin/approvals', label: 'Approvals', icon: ShieldCheck },
  { path: '/admin/asset-actions', label: 'Asset Ops', icon: Bot },
  { path: '/admin/users', label: 'Users', icon: Users },
  { path: '/admin/teams', label: 'Teams', icon: Building2 },
  { path: '/admin/audit', label: 'Audit', icon: ScrollText },
  { path: '/admin/integrations', label: 'Integrations', icon: Plug },
  { path: '/admin/setup', label: 'Setup', icon: Settings },
  { path: '/admin/ops', label: 'Ops', icon: Cpu },
  { path: '/admin/metrics', label: 'Metrics', icon: BarChart3 },
];

function initials(name?: string): string {
  if (!name) return '?';
  return name.split(/\s+/).slice(0, 2).map(p => p[0]?.toUpperCase() ?? '').join('') || '?';
}

function NavLink({ item, active, collapsed, onClick }: {
  item: NavItem;
  active: boolean;
  collapsed: boolean;
  onClick: () => void;
}) {
  const Icon = item.icon;
  return (
    <a
      href={item.path}
      onClick={(e) => { e.preventDefault(); onClick(); }}
      title={collapsed ? item.label : undefined}
      aria-current={active ? 'page' : undefined}
      className={cn(
        'flex items-center gap-2.5 rounded-md px-2.5 py-2 text-sm font-medium transition-colors',
        collapsed && 'justify-center px-2',
        active
          ? 'bg-primary text-primary-foreground'
          : 'text-muted-foreground hover:bg-accent hover:text-foreground',
      )}
    >
      <Icon className="size-4 shrink-0" />
      {!collapsed && <span className="truncate">{item.label}</span>}
    </a>
  );
}

export default function AppLayout() {
  const nav = useNavigate();
  const loc = useLocation();
  const { user, activeTeamId, isPlatformOwner, logout, switchTeam, hasPermission } = useAuth();
  const wsConnected = useRealtimeConnected();
  const { resolvedTheme, toggleTheme } = useTheme();
  const [collapsed, setCollapsed] = useState(false);
  const [mobileOpen, setMobileOpen] = useState(false);
  const [cmdOpen, setCmdOpen] = useState(false);

  // Cmd/Ctrl+K opens the palette
  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === 'k') {
        e.preventDefault();
        setCmdOpen((o) => !o);
      }
    };
    window.addEventListener('keydown', handler);
    return () => window.removeEventListener('keydown', handler);
  }, []);

  const filterVisible = (items: NavItem[]) =>
    items.filter(i => !i.perm || hasPermission(i.perm));

  const primary = filterVisible(PRIMARY_NAV);
  const account = filterVisible(ACCOUNT_NAV);
  const admin = isPlatformOwner ? ADMIN_NAV : [];

  const isActive = (path: string) =>
    path === '/' ? loc.pathname === '/' : loc.pathname.startsWith(path);

  const SidebarContent = (
    <>
      {/* Brand */}
      <div className={cn('flex items-center gap-2 border-b border-border px-4 py-3.5', collapsed && 'justify-center px-2')}>
        <div className="flex size-7 items-center justify-center rounded-md bg-primary text-sm font-bold text-primary-foreground">
          C
        </div>
        {!collapsed && <span className="font-heading text-sm font-semibold tracking-tight">ClarityIT</span>}
      </div>

      {/* Team switcher */}
      {!collapsed && user && user.teams?.length > 1 && (
        <div className="border-b border-border px-3 py-2">
          <select
            value={activeTeamId || ''}
            onChange={(e) => switchTeam(e.target.value)}
            className="h-8 w-full rounded-md border border-border bg-background px-2 text-xs text-foreground"
            aria-label="Switch team"
          >
            {user.teams.map((t) => <option key={t.id} value={t.id}>{t.name}</option>)}
          </select>
        </div>
      )}

      {/* Primary nav */}
      <nav className="flex-1 space-y-0.5 overflow-y-auto p-2">
        {primary.map((item) => (
          <NavLink key={item.path} item={item} active={isActive(item.path)} collapsed={collapsed}
            onClick={() => { nav(item.path); setMobileOpen(false); }} />
        ))}
      </nav>

      {/* Account + admin */}
      <div className="space-y-0.5 border-t border-border p-2">
        {account.map((item) => (
          <NavLink key={item.path} item={item} active={isActive(item.path)} collapsed={collapsed}
            onClick={() => { nav(item.path); setMobileOpen(false); }} />
        ))}
        {admin.length > 0 && (
          <>
            <div className={cn('px-2.5 pb-1 pt-2 text-[10px] font-semibold uppercase tracking-wider text-muted-foreground/70', collapsed && 'hidden')}>
              Admin
            </div>
            {admin.map((item) => (
              <NavLink key={item.path} item={item} active={isActive(item.path)} collapsed={collapsed}
                onClick={() => { nav(item.path); setMobileOpen(false); }} />
            ))}
          </>
        )}
      </div>
    </>
  );

  const UserMenu = (
    <DropdownMenu>
      <DropdownMenuTrigger
        render={
          <button
            type="button"
            className="flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-left hover:bg-accent"
            aria-label="User menu"
          >
            <Avatar className="size-7">
              <AvatarFallback className="text-[10px]">{initials(user?.name)}</AvatarFallback>
            </Avatar>
            {!collapsed && (
              <div className="min-w-0 flex-1">
                <div className="truncate text-xs font-medium text-foreground">{user?.name}</div>
                <div className={cn('flex items-center gap-1 text-[10px]', wsConnected ? 'text-success' : 'text-muted-foreground')}>
                  <span className={cn('size-1.5 rounded-full', wsConnected ? 'bg-success' : 'bg-muted-foreground')} />
                  {wsConnected ? 'Live' : 'Offline'}
                </div>
              </div>
            )}
          </button>
        }
      />
      <DropdownMenuContent side="top" align="end" className="w-52">
        <DropdownMenuLabel>{user?.email}</DropdownMenuLabel>
        <DropdownMenuSeparator />
        <DropdownMenuItem onClick={() => nav('/account/security')}>
          <ShieldCheck className="size-4" /> Security
        </DropdownMenuItem>
        <DropdownMenuItem onClick={toggleTheme}>
          {resolvedTheme === 'dark' ? <Sun className="size-4" /> : <Moon className="size-4" />}
          {resolvedTheme === 'dark' ? 'Light theme' : 'Dark theme'}
        </DropdownMenuItem>
        <DropdownMenuSeparator />
        <DropdownMenuItem onClick={logout} className="text-destructive">
          <LogOut className="size-4" /> Logout
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );

  return (
    <div className="flex h-screen overflow-hidden bg-background text-foreground">
      {/* Desktop sidebar */}
      <aside
        className={cn(
          'hidden shrink-0 flex-col border-r border-border bg-sidebar md:flex',
          collapsed ? 'w-16' : 'w-60',
        )}
      >
        {SidebarContent}
        <div className="border-t border-border p-2">
          {UserMenu}
          <button
            type="button"
            onClick={() => setCollapsed((c) => !c)}
            className="mt-1 flex w-full items-center justify-center gap-2 rounded-md px-2 py-1.5 text-xs text-muted-foreground hover:bg-accent hover:text-foreground"
            aria-label={collapsed ? 'Expand sidebar' : 'Collapse sidebar'}
          >
            <Menu className="size-3.5" />
            {!collapsed && 'Collapse'}
          </button>
        </div>
      </aside>

      {/* Mobile sidebar (slide-over) */}
      {mobileOpen && (
        <div className="fixed inset-0 z-50 md:hidden" data-testid="mobile-sidebar">
          <div className="absolute inset-0 bg-black/40" onClick={() => setMobileOpen(false)} />
          <aside className="absolute left-0 top-0 flex h-full w-64 flex-col border-r border-border bg-sidebar">
            <button
              type="button"
              onClick={() => setMobileOpen(false)}
              className="absolute right-2 top-2 rounded-md p-1.5 text-muted-foreground hover:bg-accent"
              aria-label="Close menu"
            >
              <X className="size-4" />
            </button>
            {SidebarContent}
          </aside>
        </div>
      )}

      {/* Main column */}
      <div className="flex min-w-0 flex-1 flex-col">
        {/* Top bar */}
        <header className="flex h-12 shrink-0 items-center gap-2 border-b border-border bg-background px-3">
          <button
            type="button"
            onClick={() => setMobileOpen(true)}
            className="rounded-md p-1.5 text-muted-foreground hover:bg-accent md:hidden"
            aria-label="Open menu"
          >
            <Menu className="size-5" />
          </button>

          {/* Command palette trigger */}
          <button
            type="button"
            onClick={() => setCmdOpen(true)}
            data-testid="command-trigger"
            className="ml-auto flex items-center gap-2 rounded-md border border-border bg-muted/40 px-2.5 py-1.5 text-xs text-muted-foreground hover:bg-muted md:ml-0"
          >
            <CommandIcon className="size-3.5" />
            <span className="hidden sm:inline">Search…</span>
            <kbd className="hidden rounded border border-border bg-background px-1 text-[10px] sm:inline">⌘K</kbd>
          </button>

          <button
            type="button"
            onClick={toggleTheme}
            className="ml-auto rounded-md p-1.5 text-muted-foreground hover:bg-accent md:ml-0"
            aria-label="Toggle theme"
          >
            {resolvedTheme === 'dark' ? <Sun className="size-4" /> : <Moon className="size-4" />}
          </button>
        </header>

        {/* Page content */}
        <main className="flex-1 overflow-auto p-6">
          <Outlet />
        </main>
      </div>

      <CommandPalette
        open={cmdOpen}
        onOpenChange={setCmdOpen}
        hasPermission={hasPermission}
        isPlatformOwner={isPlatformOwner}
      />
    </div>
  );
}
