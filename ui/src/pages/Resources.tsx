import { useEffect, useState, useMemo } from 'react';
import { useNavigate } from 'react-router-dom';
import { Search, Server } from 'lucide-react';
import { getResources } from '../api/client';
import type { Node } from '../api/types';
import StatusBadge from '../components/StatusBadge';
import TypeBadge from '../components/TypeBadge';
import EmptyState from '../components/EmptyState';

export default function Resources() {
  const [nodes, setNodes] = useState<Node[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [filterType, setFilterType] = useState('');
  const [filterNs, setFilterNs] = useState('');
  const [search, setSearch] = useState('');
  const navigate = useNavigate();

  useEffect(() => {
    getResources()
      .then((data) => setNodes(data.nodes || []))
      .catch((err) => setError(String(err)))
      .finally(() => setLoading(false));
  }, []);

  const types = useMemo(() => [...new Set(nodes.map((n) => n.type))].sort(), [nodes]);
  const namespaces = useMemo(
    () => [...new Set(nodes.map((n) => n.namespace).filter(Boolean))].sort(),
    [nodes],
  );

  const filtered = useMemo(() => {
    return nodes.filter((n) => {
      if (filterType && n.type !== filterType) return false;
      if (filterNs && n.namespace !== filterNs) return false;
      if (search && !n.id.toLowerCase().includes(search.toLowerCase())) return false;
      return true;
    });
  }, [nodes, filterType, filterNs, search]);

  if (loading) {
    return (
      <div className="flex h-64 items-center justify-center text-sm text-neutral-400">
        Loading resources…
      </div>
    );
  }

  return (
    <div className="h-full overflow-auto">
      {/* Page header */}
      <div className="border-b border-neutral-200 bg-white px-6 py-5">
        <h2 className="text-xl font-semibold text-neutral-700">Resources</h2>
        <p className="mt-1 text-sm text-neutral-500">
          {nodes.length} infrastructure resources discovered
        </p>
      </div>

      <div className="p-6">
        {error && (
          <div className="mb-4 rounded-md border border-danger-border bg-danger-bg px-4 py-3 text-sm text-danger">
            {error}
          </div>
        )}

        {/* Toolbar / filter bar (Vault-style) */}
        <div className="mb-4 flex flex-wrap items-center gap-3">
          <div className="relative flex-1 min-w-[200px]">
            <Search size={14} className="absolute left-3 top-1/2 -translate-y-1/2 text-neutral-400" />
            <input
              type="text"
              placeholder="Search by ID…"
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              className="w-full rounded-md border border-neutral-200 bg-neutral-50 py-2 pl-9 pr-3 text-sm text-neutral-700 placeholder:text-neutral-400 focus:border-brand focus:outline-none"
            />
          </div>
          <select
            value={filterType}
            onChange={(e) => setFilterType(e.target.value)}
            className="rounded-md border border-neutral-200 bg-neutral-50 px-3 py-2 text-sm text-neutral-700 focus:border-brand focus:outline-none"
          >
            <option value="">All Types</option>
            {types.map((t) => (
              <option key={t} value={t}>{t}</option>
            ))}
          </select>
          <select
            value={filterNs}
            onChange={(e) => setFilterNs(e.target.value)}
            className="rounded-md border border-neutral-200 bg-neutral-50 px-3 py-2 text-sm text-neutral-700 focus:border-brand focus:outline-none"
          >
            <option value="">All Namespaces</option>
            {namespaces.map((ns) => (
              <option key={ns} value={ns}>{ns}</option>
            ))}
          </select>
        </div>

        {/* Table */}
        <div className="overflow-hidden rounded-lg border border-neutral-200 bg-white">
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b-2 border-neutral-200">
                  <th className="px-4 py-3 text-left text-xs font-semibold uppercase tracking-wider text-neutral-500">ID</th>
                  <th className="px-4 py-3 text-left text-xs font-semibold uppercase tracking-wider text-neutral-500">Type</th>
                  <th className="px-4 py-3 text-left text-xs font-semibold uppercase tracking-wider text-neutral-500">Provider</th>
                  <th className="px-4 py-3 text-left text-xs font-semibold uppercase tracking-wider text-neutral-500">Namespace</th>
                  <th className="px-4 py-3 text-left text-xs font-semibold uppercase tracking-wider text-neutral-500">Status</th>
                  <th className="px-4 py-3 text-left text-xs font-semibold uppercase tracking-wider text-neutral-500">Updated</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-neutral-100">
                {filtered.map((n) => (
                  <tr
                    key={n.id}
                    onClick={() => navigate(`/resources/${encodeURIComponent(n.id)}`)}
                    className="cursor-pointer transition hover:bg-neutral-50"
                  >
                    <td className="px-4 py-3 font-semibold text-neutral-700">{n.id}</td>
                    <td className="px-4 py-3"><TypeBadge type={n.type} /></td>
                    <td className="px-4 py-3 text-neutral-600">{n.provider}</td>
                    <td className="px-4 py-3 text-neutral-500">{n.namespace || '—'}</td>
                    <td className="px-4 py-3"><StatusBadge status={n.status} /></td>
                    <td className="px-4 py-3 text-xs text-neutral-400">
                      {new Date(n.updated).toLocaleString()}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          {filtered.length === 0 && (
            <EmptyState
              icon={<Server size={20} />}
              title="No resources found"
              description="Try adjusting your filters or start a collector."
            />
          )}
        </div>
      </div>
    </div>
  );
}
