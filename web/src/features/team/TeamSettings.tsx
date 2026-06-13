import { useEffect, useState } from 'react';
import { api, type Member, type Invitation } from '../../api/client';
import { useAuth } from '../../auth/context';

export default function TeamSettings() {
  const { activeTeamId, hasPermission } = useAuth();
  const [members, setMembers] = useState<Member[]>([]);
  const [invitations, setInvitations] = useState<Invitation[]>([]);
  const [inviteEmail, setInviteEmail] = useState('');
  const [inviteRole, setInviteRole] = useState('member');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(true);

  const load = () => {
    if (!activeTeamId) return;
    Promise.all([api.listMembers(), api.listInvitations()])
      .then(([m, inv]) => { setMembers(m); setInvitations(inv); }).catch(e => setError(e.message))
      .finally(() => setLoading(false));
  };
  useEffect(load, [activeTeamId]);

  const invite = async (e: React.FormEvent) => {
    e.preventDefault();
    try { await api.createInvitation(inviteEmail, inviteRole); setInviteEmail(''); load(); }
    catch (e: any) { setError(e.message); }
  };

  const removeMember = async (userId: string) => {
    if (!confirm('Remove this member?')) return;
    try { await api.removeMember(userId); load(); } catch (e: any) { setError(e.message); }
  };

  const roleBadge: Record<string, string> = { owner: 'badge-yellow', admin: 'badge-red', manager: 'badge-blue', member: 'badge-gray', viewer: 'badge-gray' };

  if (loading) return <p className="text-[var(--text-muted)]">Loading...</p>;

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold">Team Settings</h1>
      {error && <div className="error-msg">{error}</div>}

      <div className="card">
        <h3 className="font-semibold mb-3">Members ({members.length})</h3>
        <table className="w-full text-sm">
          <thead><tr className="text-left text-[var(--text-muted)] border-b border-[var(--border)]">
            <th className="pb-2">Name</th><th className="pb-2">Email</th><th className="pb-2">Role</th><th className="pb-2">Joined</th><th className="pb-2"></th>
          </tr></thead>
          <tbody>
            {members.map(m => (
              <tr key={m.user_id} className="border-b border-[var(--border)]">
                <td className="py-2">{m.name}</td>
                <td className="py-2 text-[var(--text-muted)]">{m.email}</td>
                <td className="py-2"><span className={`badge ${roleBadge[m.role] || 'badge-gray'}`}>{m.role}</span></td>
                <td className="py-2 text-[var(--text-muted)]">{new Date(m.joined_at).toLocaleDateString()}</td>
                <td className="py-2 text-right">
                  {hasPermission('team.members.remove') && m.role !== 'owner' && (
                    <button className="btn-danger text-xs px-2 py-1" onClick={() => removeMember(m.user_id)}>Remove</button>
                  )}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {hasPermission('team.invitations.create') && (
        <div className="card">
          <h3 className="font-semibold mb-3">Invite Member</h3>
          <form onSubmit={invite} className="flex gap-3">
            <input type="email" placeholder="Email address" value={inviteEmail} onChange={e => setInviteEmail(e.target.value)} required className="flex-1" />
            <select value={inviteRole} onChange={e => setInviteRole(e.target.value)}>
              <option value="member">Member</option><option value="viewer">Viewer</option>
              <option value="manager">Manager</option><option value="admin">Admin</option>
            </select>
            <button type="submit">Invite</button>
          </form>
        </div>
      )}

      {invitations.length > 0 && (
        <div className="card">
          <h3 className="font-semibold mb-3">Pending Invitations</h3>
          {invitations.filter(i => !i.accepted_at).map(inv => (
            <div key={inv.id} className="flex justify-between py-2 border-b border-[var(--border)] last:border-0">
              <span className="text-sm">{inv.email}</span>
              <div className="flex gap-2 items-center">
                <span className="badge badge-gray">{inv.role}</span>
                <span className="text-xs text-[var(--text-muted)]">Expires {new Date(inv.expires_at).toLocaleDateString()}</span>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
