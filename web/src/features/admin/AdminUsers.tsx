import { useEffect, useState } from 'react';
import { api } from '../../api/client';

export default function AdminUsers() {
  const [users, setUsers] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    setLoading(true);
    api.listUsers().then(setUsers).catch(() => {}).finally(() => setLoading(false));
  }, []);

  const toggleActive = async (id: string, active: boolean) => {
    try { await api.updateUser(id, { is_active: !active }); api.listUsers().then(setUsers); } catch {}
  };

  return (
    <div className="space-y-4">
      <h1 className="text-2xl font-bold">Users</h1>
      {loading ? <p className="text-[var(--text-muted)]">Loading...</p> : (
        <div className="card">
          <table className="w-full text-sm">
            <thead><tr className="text-left text-[var(--text-muted)] border-b border-[var(--border)]">
              <th className="pb-2">Name</th><th className="pb-2">Email</th><th className="pb-2">Role</th><th className="pb-2">Status</th><th className="pb-2">Actions</th>
            </tr></thead>
            <tbody>
              {users.map(u => (
                <tr key={u.id} className="border-b border-[var(--border)]">
                  <td className="py-2">{u.name}</td>
                  <td className="py-2 text-[var(--text-muted)]">{u.email}</td>
                  <td className="py-2">{u.is_platform_owner ? <span className="badge badge-yellow">Owner</span> : <span className="badge badge-gray">User</span>}</td>
                  <td className="py-2">{u.is_active ? <span className="badge badge-green">Active</span> : <span className="badge badge-red">Inactive</span>}</td>
                  <td className="py-2">
                    <button className="btn-secondary text-xs px-2 py-1" onClick={() => toggleActive(u.id, u.is_active)}>
                      {u.is_active ? 'Deactivate' : 'Activate'}
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
