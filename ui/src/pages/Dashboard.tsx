import { useEffect, useState, useMemo } from 'react';
import { useNavigate } from 'react-router-dom';
import { Server, Network, GitBranch, Activity } from 'lucide-react';
import { getStatus, getResources } from '../api/client';
import type { SystemStatus, Node } from '../api/types';
import { nodeColor, NODE_TYPES } from '../utils/colors';

export default function Dashboard() {
  const [status, setStatus] = useState<SystemStatus | null>(null);
  const [nodes, setNodes] = useState<Node[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const navigate = useNavigate();

  useEffect(() => {
    Promise.all([getStatus(), getResources()])
      .then(([s, r]) => {
        setStatus(s);
        setNodes(r.nodes || []);
      })
      .catch((err) => setError(String(err)))
      .finally(() => setLoading(false));
  }, []);

  const typeBreakdown = useMemo(() => {
    const counts: Record<string, number> = {};
    for (const n of nodes) {
      counts[n.type] = (counts[n.type] || 0) + 1;
    }
    return counts;
  }, [nodes]);

  const statusBreakdown = useMemo(() => {
    const counts: Record<string, number> = {};
    for (const n of nodes) {
      counts[n.status] = (counts[n.status] || 0) + 1;
    }
    return counts;
  }, [nodes]);

  const providerBreakdown = useMemo(() => {
    const counts: Record<string, number> = {};
    for (const n of nodes) {
      counts[n.provider] = (counts[n.provider] || 0) + 1;
    }
    return counts;
  }, [nodes]);

  if (loading) return <div className="loading">Loading dashboard...</div>;

  return (
    <>
      <div className="page-header">
        <h2>Dashboard</h2>
        <p>Infrastructure overview</p>
      </div>
      <div className="page-body">
        {error && <div className="error-banner">{error}</div>}

        <div className="stats-grid">
          <div className="stat-card" onClick={() => navigate('/resources')} style={{ cursor: 'pointer' }}>
            <div className="stat-label">
              <Server size={14} style={{ marginRight: 6, verticalAlign: 'text-bottom' }} />
              Nodes
            </div>
            <div className="stat-value">{status?.node_count ?? 0}</div>
            <div className="stat-detail">Infrastructure resources</div>
          </div>
          <div className="stat-card" onClick={() => navigate('/graph')} style={{ cursor: 'pointer' }}>
            <div className="stat-label">
              <GitBranch size={14} style={{ marginRight: 6, verticalAlign: 'text-bottom' }} />
              Edges
            </div>
            <div className="stat-value">{status?.edge_count ?? 0}</div>
            <div className="stat-detail">Dependencies & relationships</div>
          </div>
          <div className="stat-card">
            <div className="stat-label">
              <Activity size={14} style={{ marginRight: 6, verticalAlign: 'text-bottom' }} />
              Version
            </div>
            <div className="stat-value" style={{ fontSize: 24 }}>{status?.version ?? '—'}</div>
            <div className="stat-detail">{status?.store_path}</div>
          </div>
          <div className="stat-card">
            <div className="stat-label">
              <Network size={14} style={{ marginRight: 6, verticalAlign: 'text-bottom' }} />
              Providers
            </div>
            <div className="stat-value">{Object.keys(providerBreakdown).length}</div>
            <div className="stat-detail">{Object.keys(providerBreakdown).join(', ') || 'None'}</div>
          </div>
        </div>

        <div className="detail-grid">
          {/* Resource Types breakdown */}
          <div className="card">
            <div className="card-header">Resources by Type</div>
            <div className="card-body">
              {NODE_TYPES.filter((t) => typeBreakdown[t]).map((type) => (
                <div
                  key={type}
                  style={{
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'space-between',
                    padding: '8px 0',
                    borderBottom: '1px solid var(--color-border)',
                    cursor: 'pointer',
                  }}
                  onClick={() => navigate('/resources')}
                >
                  <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
                    <div
                      style={{
                        width: 10,
                        height: 10,
                        borderRadius: '50%',
                        background: nodeColor(type),
                      }}
                    />
                    <span style={{ textTransform: 'capitalize', fontWeight: 500 }}>{type}</span>
                  </div>
                  <span style={{ fontWeight: 700, fontSize: 18 }}>{typeBreakdown[type]}</span>
                </div>
              ))}
              {Object.keys(typeBreakdown).length === 0 && (
                <span style={{ color: 'var(--color-text-muted)', fontSize: 14 }}>
                  No resources discovered yet
                </span>
              )}
            </div>
          </div>

          {/* Health Status breakdown */}
          <div className="card">
            <div className="card-header">Health Status</div>
            <div className="card-body">
              {Object.entries(statusBreakdown).map(([st, count]) => (
                <div
                  key={st}
                  style={{
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'space-between',
                    padding: '8px 0',
                    borderBottom: '1px solid var(--color-border)',
                  }}
                >
                  <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
                    <div
                      style={{
                        width: 10,
                        height: 10,
                        borderRadius: '50%',
                        background:
                          st === 'healthy'
                            ? 'var(--color-success)'
                            : st === 'degraded'
                              ? 'var(--color-warning)'
                              : 'var(--color-text-muted)',
                      }}
                    />
                    <span style={{ textTransform: 'capitalize', fontWeight: 500 }}>{st}</span>
                  </div>
                  <span style={{ fontWeight: 700, fontSize: 18 }}>{count}</span>
                </div>
              ))}
              {Object.keys(statusBreakdown).length === 0 && (
                <span style={{ color: 'var(--color-text-muted)', fontSize: 14 }}>
                  No data
                </span>
              )}
            </div>
          </div>
        </div>
      </div>
    </>
  );
}
