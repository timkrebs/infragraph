import { useEffect, useState, useMemo } from 'react';
import { useNavigate } from 'react-router-dom';
import { getResources } from '../api/client';
import type { Node } from '../api/types';
import StatusBadge from '../components/StatusBadge';
import TypeBadge from '../components/TypeBadge';

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
  const namespaces = useMemo(() => [...new Set(nodes.map((n) => n.namespace).filter(Boolean))].sort(), [nodes]);

  const filtered = useMemo(() => {
    return nodes.filter((n) => {
      if (filterType && n.type !== filterType) return false;
      if (filterNs && n.namespace !== filterNs) return false;
      if (search && !n.id.toLowerCase().includes(search.toLowerCase())) return false;
      return true;
    });
  }, [nodes, filterType, filterNs, search]);

  if (loading) return <div className="loading">Loading resources...</div>;

  return (
    <>
      <div className="page-header">
        <h2>Resources</h2>
        <p>{nodes.length} infrastructure resources discovered</p>
      </div>
      <div className="page-body">
        {error && <div className="error-banner">{error}</div>}

        <div className="filter-bar">
          <input
            type="text"
            placeholder="Search by ID..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
          />
          <select value={filterType} onChange={(e) => setFilterType(e.target.value)}>
            <option value="">All Types</option>
            {types.map((t) => (
              <option key={t} value={t}>{t}</option>
            ))}
          </select>
          <select value={filterNs} onChange={(e) => setFilterNs(e.target.value)}>
            <option value="">All Namespaces</option>
            {namespaces.map((ns) => (
              <option key={ns} value={ns}>{ns}</option>
            ))}
          </select>
        </div>

        <div className="card">
          <div className="table-wrapper">
            <table>
              <thead>
                <tr>
                  <th>ID</th>
                  <th>Type</th>
                  <th>Provider</th>
                  <th>Namespace</th>
                  <th>Status</th>
                  <th>Updated</th>
                </tr>
              </thead>
              <tbody>
                {filtered.map((n) => (
                  <tr
                    key={n.id}
                    onClick={() => navigate(`/resources/${encodeURIComponent(n.id)}`)}
                  >
                    <td style={{ fontWeight: 600 }}>{n.id}</td>
                    <td><TypeBadge type={n.type} /></td>
                    <td>{n.provider}</td>
                    <td>{n.namespace || '—'}</td>
                    <td><StatusBadge status={n.status} /></td>
                    <td style={{ color: 'var(--color-text-muted)', fontSize: '13px' }}>
                      {new Date(n.updated).toLocaleString()}
                    </td>
                  </tr>
                ))}
                {filtered.length === 0 && (
                  <tr>
                    <td colSpan={6} style={{ textAlign: 'center', color: 'var(--color-text-muted)', padding: 32 }}>
                      No resources found
                    </td>
                  </tr>
                )}
              </tbody>
            </table>
          </div>
        </div>
      </div>
    </>
  );
}
