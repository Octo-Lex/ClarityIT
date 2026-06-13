import { useState, useEffect } from 'react';
import { api, type Agent, type AgentGrant, type AgentRun, type AgentIntention } from '../../api/client';
import { usePermissions } from '../../hooks/usePermissions';

export function AgentsPage() {
  const { hasPermission: has } = usePermissions();
  const [agents, setAgents] = useState<Agent[]>([]);
  const [selected, setSelected] = useState<string | null>(null);
  const [grants, setGrants] = useState<AgentGrant[]>([]);
  const [runs, setRuns] = useState<AgentRun[]>([]);
  const [intentions, setIntentions] = useState<AgentIntention[]>([]);
  const [showCreate, setShowCreate] = useState(false);
  const [loading, setLoading] = useState(true);

  useEffect(() => { loadAgents(); loadRuns(); }, []);

  async function loadAgents() {
    try { setAgents(await api.listAgents()); } catch {}
    setLoading(false);
  }

  async function loadRuns() {
    try { setRuns(await api.listRuns()); } catch {}
  }

  async function selectAgent(id: string) {
    setSelected(id);
    try {
      setGrants(await api.listGrants(id));
    } catch { setGrants([]); }
  }

  async function loadIntentions(runId: string) {
    try { setIntentions(await api.listIntentions(runId)); } catch { setIntentions([]); }
  }

  async function createAgent(e: React.FormEvent<HTMLFormElement>) {
    e.preventDefault();
    const fd = new FormData(e.currentTarget);
    try {
      await api.createAgent({
        name: fd.get('name') as string,
        max_autonomy: fd.get('max_autonomy') as string,
        description: fd.get('description') as string || '',
      });
      setShowCreate(false);
      loadAgents();
    } catch {}
  }

  async function disableAgent(id: string) {
    if (!confirm('Disable this agent?')) return;
    try { await api.disableAgent(id); loadAgents(); setSelected(null); } catch {}
  }

  async function createGrant(e: React.FormEvent<HTMLFormElement>) {
    e.preventDefault();
    if (!selected) return;
    const fd = new FormData(e.currentTarget);
    try {
      await api.createGrant(selected, {
        tool_name: fd.get('tool_name') as string,
        max_autonomy_level: fd.get('max_autonomy_level') as string,
        requires_approval: fd.get('requires_approval') === 'on',
      });
      setGrants(await api.listGrants(selected));
      e.currentTarget.reset();
    } catch {}
  }

  async function revokeGrant(grantId: string) {
    if (!selected) return;
    try { await api.revokeGrant(selected, grantId); setGrants(await api.listGrants(selected)); } catch {}
  }

  if (!has('agents.read')) return <div className="p-8"><p>Access denied</p></div>;
  if (loading) return <div className="p-8"><p>Loading agents...</p></div>;

  const selectedAgent = agents.find(a => a.id === selected);

  return (
    <div className="p-6 space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold">Agent Console</h1>
        {has('agents.create') && (
          <button onClick={() => setShowCreate(true)} className="px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700">
            + Create Agent
          </button>
        )}
      </div>

      {showCreate && (
        <form onSubmit={createAgent} className="bg-white border rounded-lg p-4 space-y-3 shadow-sm">
          <h2 className="font-semibold">New Agent</h2>
          <input name="name" placeholder="Agent name" required className="w-full border rounded px-3 py-2" />
          <textarea name="description" placeholder="Description (optional)" className="w-full border rounded px-3 py-2" />
          <select name="max_autonomy" defaultValue="A3" className="border rounded px-3 py-2">
            {['A0','A1','A2','A3','A4','A5'].map(a => <option key={a} value={a}>{a}</option>)}
          </select>
          <div className="flex gap-2">
            <button type="submit" className="px-4 py-2 bg-blue-600 text-white rounded">Create</button>
            <button type="button" onClick={() => setShowCreate(false)} className="px-4 py-2 border rounded">Cancel</button>
          </div>
        </form>
      )}

      <div className="grid grid-cols-3 gap-4">
        {/* Agent List */}
        <div className="col-span-1">
          <h2 className="font-semibold mb-2">Agents</h2>
          {agents.length === 0 ? <p className="text-gray-500 text-sm">No agents</p> : (
            <ul className="space-y-2">
              {agents.map(a => (
                <li key={a.id}
                  onClick={() => selectAgent(a.id)}
                  className={`p-3 border rounded cursor-pointer hover:bg-gray-50 ${selected === a.id ? 'border-blue-500 bg-blue-50' : ''}`}>
                  <div className="font-medium">{a.name}</div>
                  <div className="flex gap-2 text-xs text-gray-500">
                    <span className={`px-1.5 py-0.5 rounded ${a.status === 'active' ? 'bg-green-100 text-green-700' : 'bg-red-100 text-red-700'}`}>
                      {a.status}
                    </span>
                    <span>Max: {a.max_autonomy}</span>
                  </div>
                </li>
              ))}
            </ul>
          )}
        </div>

        {/* Agent Detail */}
        <div className="col-span-2 space-y-4">
          {selectedAgent ? (
            <>
              <div className="bg-white border rounded-lg p-4 shadow-sm">
                <div className="flex justify-between items-start">
                  <div>
                    <h2 className="text-lg font-semibold">{selectedAgent.name}</h2>
                    <p className="text-sm text-gray-500">{selectedAgent.description || 'No description'}</p>
                    <p className="text-xs text-gray-400 mt-1">ID: {selectedAgent.id}</p>
                  </div>
                  {has('agents.disable') && selectedAgent.status === 'active' && (
                    <button onClick={() => disableAgent(selectedAgent.id)} className="px-3 py-1 text-sm border border-red-300 text-red-600 rounded hover:bg-red-50">
                      Disable
                    </button>
                  )}
                </div>
              </div>

              {/* Grants */}
              <div className="bg-white border rounded-lg p-4 shadow-sm">
                <h3 className="font-semibold mb-2">Tool Grants</h3>
                {has('agents.grants.create') && (
                  <form onSubmit={createGrant} className="flex gap-2 mb-3">
                    <input name="tool_name" placeholder="Tool name" required className="border rounded px-2 py-1 text-sm flex-1" />
                    <select name="max_autonomy_level" defaultValue="A3" className="border rounded px-2 py-1 text-sm">
                      {['A0','A1','A2','A3','A4','A5'].map(a => <option key={a} value={a}>{a}</option>)}
                    </select>
                    <label className="flex items-center gap-1 text-sm">
                      <input type="checkbox" name="requires_approval" /> Approval
                    </label>
                    <button type="submit" className="px-3 py-1 text-sm bg-blue-600 text-white rounded">Add</button>
                  </form>
                )}
                {grants.length === 0 ? <p className="text-sm text-gray-400">No grants</p> : (
                  <table className="w-full text-sm">
                    <thead><tr className="text-left text-gray-500 border-b"><th>Tool</th><th>Autonomy</th><th>Approval</th><th>Status</th><th></th></tr></thead>
                    <tbody>
                      {grants.map(g => (
                        <tr key={g.id} className="border-b">
                          <td>{g.tool_name}</td>
                          <td>{g.max_autonomy_level}</td>
                          <td>{g.requires_approval ? 'Yes' : 'No'}</td>
                          <td>{g.revoked_at ? <span className="text-red-500">Revoked</span> : <span className="text-green-600">Active</span>}</td>
                          <td>{!g.revoked_at && has('agents.grants.revoke') && (
                            <button onClick={() => revokeGrant(g.id)} className="text-red-500 text-xs hover:underline">Revoke</button>
                          )}</td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                )}
              </div>
            </>
          ) : (
            <p className="text-gray-400">Select an agent to view details</p>
          )}

          {/* Runs */}
          <div className="bg-white border rounded-lg p-4 shadow-sm">
            <h3 className="font-semibold mb-2">Agent Runs</h3>
            {runs.length === 0 ? <p className="text-sm text-gray-400">No runs</p> : (
              <table className="w-full text-sm">
                <thead><tr className="text-left text-gray-500 border-b"><th>ID</th><th>Status</th><th>Created</th><th></th></tr></thead>
                <tbody>
                  {runs.slice(0, 10).map(r => (
                    <tr key={r.id} className="border-b">
                      <td className="font-mono text-xs">{r.id.slice(0, 8)}...</td>
                      <td><span className={`px-1.5 py-0.5 rounded text-xs ${r.status === 'completed' ? 'bg-green-100 text-green-700' : r.status === 'failed' ? 'bg-red-100 text-red-700' : 'bg-yellow-100 text-yellow-700'}`}>{r.status}</span></td>
                      <td className="text-gray-400">{new Date(r.created_at).toLocaleString()}</td>
                      <td><button onClick={() => loadIntentions(r.id)} className="text-blue-500 text-xs hover:underline">View</button></td>
                    </tr>
                  ))}
                </tbody>
              </table>
            )}
          </div>

          {/* Intentions */}
          {intentions.length > 0 && (
            <div className="bg-white border rounded-lg p-4 shadow-sm">
              <h3 className="font-semibold mb-2">Intentions</h3>
              <table className="w-full text-sm">
                <thead><tr className="text-left text-gray-500 border-b"><th>Tool</th><th>Autonomy</th><th>Status</th><th>Reason</th></tr></thead>
                <tbody>
                  {intentions.map(i => (
                    <tr key={i.id} className="border-b">
                      <td>{i.requested_tool}</td>
                      <td>{i.autonomy_level}</td>
                      <td><span className={`px-1.5 py-0.5 rounded text-xs ${
                        i.status === 'executed' ? 'bg-green-100 text-green-700' :
                        i.status === 'blocked' ? 'bg-red-100 text-red-700' :
                        'bg-yellow-100 text-yellow-700'
                      }`}>{i.status}</span></td>
                      <td className="text-gray-400">{i.blocked_reason || '—'}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
