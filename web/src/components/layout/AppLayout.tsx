import { ReactNode } from 'react';
import { useNavigate, useLocation, Outlet } from 'react-router-dom';
import { useAuth } from '../../auth/context';
import { useRefetch } from '../../hooks/useRefetch';

export default function AppLayout() {
  const nav = useNavigate();
  const loc = useLocation();
  const { user, activeTeamId, isPlatformOwner, logout, switchTeam, hasPermission } = useAuth();
  const { wsConnected } = useRefetch();

  const links: { path: string; label: string; icon: string; perm?: string }[] = [
    { path: '/', label: 'Dashboard', icon: '📊' },
    { path: '/queue', label: 'Queue', icon: '📋', perm: 'work.items.list' },
    { path: '/board', label: 'Board', icon: '📌', perm: 'work.items.list' },
    { path: '/incidents', label: 'Incidents', icon: '🔥', perm: 'incidents.list' },
    { path: '/agents', label: 'Agents', icon: '🤖', perm: 'agents.read' },
    { path: '/settings/team', label: 'Team', icon: '👥', perm: 'team.settings.view' },
  ];

  const adminLinks: { path: string; label: string; icon: string; perm?: string }[] = isPlatformOwner ? [
    { path: '/admin/users', label: 'Users', icon: '👤' },
    { path: '/admin/teams', label: 'Teams', icon: '🏗️' },
    { path: '/admin/audit', label: 'Audit', icon: '📜' },
    { path: '/admin/integrations', label: 'Integrations', icon: '🔌' },
    { path: '/admin/setup', label: 'Setup', icon: '✅' },
    { path: '/admin/ops', label: 'Ops', icon: '⚙️' },
  ] : [];

  const allLinks = [...links, ...adminLinks];
  const visibleLinks = allLinks.filter(l => !l.perm || hasPermission(l.perm));

  return (
    <div className="flex h-screen">
      {/* Sidebar */}
      <aside className="w-56 bg-[var(--card)] border-r border-[var(--border)] flex flex-col">
        <div className="p-4 border-b border-[var(--border)]">
          <h1 className="text-lg font-bold">ClarityIT</h1>
        </div>

        {/* Team switcher */}
        {user && user.teams?.length > 1 && (
          <div className="p-3 border-b border-[var(--border)]">
            <select value={activeTeamId || ''} onChange={e => switchTeam(e.target.value)} className="text-xs">
              {user.teams.map(t => <option key={t.id} value={t.id}>{t.name}</option>)}
            </select>
          </div>
        )}

        {/* Nav */}
        <nav className="flex-1 p-2 space-y-1">
          {visibleLinks.map(l => (
            <a key={l.path} href={l.path} onClick={e => { e.preventDefault(); nav(l.path); }}
              className={`flex items-center gap-2 px-3 py-2 rounded text-sm ${loc.pathname === l.path ? 'bg-[var(--primary)] text-white' : 'hover:bg-[var(--border)]'}`}>
              <span>{l.icon}</span> {l.label}
            </a>
          ))}
        </nav>

        {/* User */}
        <div className="p-3 border-t border-[var(--border)]">
          <div className="flex items-center justify-between mb-1">
            <span className="text-sm truncate">{user?.name}</span>
            <span className={`text-xs ${wsConnected ? 'text-[var(--success)]' : 'text-[var(--text-muted)]'}`}>{wsConnected ? '● Live' : '○ Offline'}</span>
          </div>
          <button onClick={logout} className="text-xs text-[var(--text-muted)] bg-transparent hover:text-white p-0">Logout</button>
        </div>
      </aside>

      {/* Main content */}
      <main className="flex-1 overflow-auto p-6">
        <Outlet />
      </main>
    </div>
  );
}
