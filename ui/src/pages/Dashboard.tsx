import { useEffect, useState, useMemo } from 'react';
import { useNavigate } from 'react-router-dom';
import { Server, GitBranch, Activity, Network, Clock, Radio } from 'lucide-react';
import { getStatus, getResources, getCollectors } from '../api/client';
import type { SystemStatus, Node, CollectorInfo } from '../api/types';
import { nodeColor, NODE_TYPES } from '../utils/colors';
import Card from '../components/Card';
import EmptyState from '../components/EmptyState';
import StatusBadge from '../components/StatusBadge';

export default function Dashboard() {
  const [status, setStatus] = useState<SystemStatus | null>(null);
  const [nodes, setNodes] = useState<Node[]>([]);
  const [collectors, setCollectors] = useState<CollectorInfo[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const navigate = useNavigate();

  useEffect(() => {
    Promise.all([getStatus(), getResources(), getCollectors()])
      .then(([s, r, c]) => { setStatus(s); setNodes(r.nodes || []); setCollectors(c); })
      .catch((err) => setError(String(err)))
      .finally(() => setLoading(false));
  }, []);

  const typeBreakdown = useMemo(() => {
    const counts: Record<string, number> = {};
    for (const n of nodes) counts[n.type] = (counts[n.type] || 0) + 1;
    return counts;
  }, [nodes]);

  const statusBreakdown = useMemo(() => {
    const counts: Record<string, number> = {};
    for (const n of nodes) counts[n.status] = (counts[n.status] || 0) + 1;
    return counts;
  }, [nodes]);

  const providers = useMemo(() => {
    const s = new Set<string>();
    for (const n of nodes) s.add(n.provider);
    return [...s];
  }, [nodes]);

  if (loading) {
    return (
      <div className="flex h-64 items-center justify-center text-sm text-neutral-400">
        Loading dashboard…
      </div>
    );
  }

  return (
    <div className="h-full overflow-auto">
      {/* Page header */}
      <div className="border-b border-neutral-200 bg-white px-6 py-5">
        <h2 className="text-xl font-semibold text-neutral-700">Dashboard</h2>
        <p className="mt-1 text-sm text-neutral-500">Infrastructure overview</p>
      </div>

      <div className="p-6">
        {error && (
          <div className="mb-4 rounded-md border border-danger-border bg-danger-bg px-4 py-3 text-sm text-danger">
            {error}
          </div>
        )}

        {/* Summary stat cards */}
        <div className="mb-6 grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
          <button
            onClick={() => navigate('/resources')}
            className="rounded-lg border border-neutral-200 bg-white p-5 text-left transition hover:border-brand"
          >
            <div className="mb-2 flex items-center gap-1.5 text-xs font-semibold uppercase tracking-wider text-neutral-500">
              <Server size={13} /> Nodes
            </div>
            <div className="text-3xl font-bold text-neutral-700">{status?.node_count ?? 0}</div>
            <div className="mt-1 text-xs text-neutral-400">Infrastructure resources</div>
          </button>

          <button
            onClick={() => navigate('/graph')}
            className="rounded-lg border border-neutral-200 bg-white p-5 text-left transition hover:border-brand"
          >
            <div className="mb-2 flex items-center gap-1.5 text-xs font-semibold uppercase tracking-wider text-neutral-500">
              <GitBranch size={13} /> Edges
            </div>
            <div className="text-3xl font-bold text-neutral-700">{status?.edge_count ?? 0}</div>
            <div className="mt-1 text-xs text-neutral-400">Dependencies & relationships</div>
          </button>

          <div className="rounded-lg border border-neutral-200 bg-white p-5">
            <div className="mb-2 flex items-center gap-1.5 text-xs font-semibold uppercase tracking-wider text-neutral-500">
              <Activity size={13} /> Version
            </div>
            <div className="text-2xl font-bold text-neutral-700">{status?.version ?? '—'}</div>
            <div className="mt-1 truncate text-xs text-neutral-400">{status?.store_path}</div>
          </div>

          <div className="rounded-lg border border-neutral-200 bg-white p-5">
            <div className="mb-2 flex items-center gap-1.5 text-xs font-semibold uppercase tracking-wider text-neutral-500">
              <Network size={13} /> Providers
            </div>
            <div className="text-3xl font-bold text-neutral-700">{providers.length}</div>
            <div className="mt-1 text-xs text-neutral-400">{providers.join(', ') || 'None'}</div>
          </div>
        </div>

        {/* Breakdown cards */}
        <div className="grid grid-cols-1 gap-4 md:grid-cols-3">
          <Card title="Resources by Type">
            {NODE_TYPES.filter((t) => typeBreakdown[t]).length > 0 ? (
              <div className="divide-y divide-neutral-100">
                {NODE_TYPES.filter((t) => typeBreakdown[t]).map((type) => (
                  <button
                    key={type}
                    onClick={() => navigate('/resources')}
                    className="flex w-full items-center justify-between py-2.5 text-left hover:bg-neutral-50"
                  >
                    <span className="flex items-center gap-2.5">
                      <span
                        className="inline-block h-2.5 w-2.5 rounded-full"
                        style={{ background: nodeColor(type) }}
                      />
                      <span className="text-sm font-medium capitalize text-neutral-700">{type}</span>
                    </span>
                    <span className="text-lg font-bold text-neutral-600">{typeBreakdown[type]}</span>
                  </button>
                ))}
              </div>
            ) : (
              <EmptyState
                icon={<Server size={20} />}
                title="No resources yet"
                description="Start a collector or load static resources to see type breakdowns."
              />
            )}
          </Card>

          <Card title="Health Status">
            {Object.keys(statusBreakdown).length > 0 ? (
              <div className="divide-y divide-neutral-100">
                {Object.entries(statusBreakdown).map(([st, count]) => (
                  <div key={st} className="flex items-center justify-between py-2.5">
                    <span className="flex items-center gap-2.5">
                      <span
                        className={`inline-block h-2.5 w-2.5 rounded-full ${
                          st === 'healthy'
                            ? 'bg-success'
                            : st === 'degraded'
                              ? 'bg-warning'
                              : 'bg-neutral-400'
                        }`}
                      />
                      <span className="text-sm font-medium capitalize text-neutral-700">{st}</span>
                    </span>
                    <span className="text-lg font-bold text-neutral-600">{count}</span>
                  </div>
                ))}
              </div>
            ) : (
              <EmptyState
                icon={<Clock size={20} />}
                title="No status data"
                description="Resource health will appear here once data is collected."
              />
            )}
          </Card>

          <Card title="Collectors">
            {collectors.length > 0 ? (
              <div className="divide-y divide-neutral-100">
                {collectors.map((c) => (
                  <button
                    key={c.name}
                    onClick={() => navigate('/collectors')}
                    className="flex w-full items-center justify-between py-2.5 text-left hover:bg-neutral-50"
                  >
                    <span className="flex items-center gap-2.5">
                      <Radio size={14} className="text-neutral-400" />
                      <span className="text-sm font-medium text-neutral-700">{c.name}</span>
                      <span className="rounded border border-neutral-200 bg-neutral-50 px-1 py-0.5 font-mono text-[10px] uppercase text-neutral-500">
                        {c.type}
                      </span>
                    </span>
                    <StatusBadge status={c.status === 'running' ? 'healthy' : c.status === 'error' ? 'degraded' : c.status} />
                  </button>
                ))}
              </div>
            ) : (
              <EmptyState
                icon={<Radio size={20} />}
                title="No collectors"
                description="Configure collectors in infragraph.hcl to discover infrastructure."
              />
            )}
          </Card>
        </div>
      </div>
    </div>
  );
}
