import { useEffect, useState } from 'react';
import { useParams, useNavigate, Link } from 'react-router-dom';
import { ArrowLeft, ArrowRight, ArrowDown } from 'lucide-react';
import { getNode, getImpact } from '../api/client';
import type { NodeDetail as NodeDetailType, ImpactResult } from '../api/types';
import StatusBadge from '../components/StatusBadge';
import TypeBadge from '../components/TypeBadge';

export default function NodeDetail() {
  const { id } = useParams<{ id: string }>();
  const decodedId = decodeURIComponent(id || '');
  const navigate = useNavigate();

  const [detail, setDetail] = useState<NodeDetailType | null>(null);
  const [impact, setImpact] = useState<ImpactResult | null>(null);
  const [impactDir, setImpactDir] = useState<'forward' | 'reverse'>('forward');
  const [tab, setTab] = useState<'overview' | 'impact'>('overview');
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  useEffect(() => {
    if (!decodedId) return;
    setLoading(true);
    setError('');
    getNode(decodedId)
      .then(setDetail)
      .catch((err) => setError(String(err)))
      .finally(() => setLoading(false));
  }, [decodedId]);

  useEffect(() => {
    if (!decodedId) return;
    getImpact(decodedId, impactDir, 10)
      .then(setImpact)
      .catch(() => setImpact(null));
  }, [decodedId, impactDir]);

  if (loading) return <div className="loading">Loading node...</div>;
  if (error) return <div className="page-body"><div className="error-banner">{error}</div></div>;
  if (!detail) return null;

  const { node, outgoing, incoming } = detail;

  return (
    <>
      <div className="page-header">
        <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
          <Link to="/resources" style={{ color: 'var(--color-text-muted)' }}>
            <ArrowLeft size={20} />
          </Link>
          <div>
            <h2>{node.id}</h2>
            <p style={{ display: 'flex', gap: 8, alignItems: 'center', marginTop: 4 }}>
              <TypeBadge type={node.type} />
              <StatusBadge status={node.status} />
              <span style={{ color: 'var(--color-text-muted)' }}>
                via {node.provider}
              </span>
            </p>
          </div>
        </div>
      </div>
      <div className="page-body">
        <div className="tabs">
          <button
            className={`tab${tab === 'overview' ? ' active' : ''}`}
            onClick={() => setTab('overview')}
          >
            Overview
          </button>
          <button
            className={`tab${tab === 'impact' ? ' active' : ''}`}
            onClick={() => setTab('impact')}
          >
            Impact Analysis
          </button>
        </div>

        {tab === 'overview' && (
          <>
            <div className="detail-grid">
              <div className="card">
                <div className="card-header">Properties</div>
                <div className="card-body">
                  <div className="property-list">
                    <span className="property-key">ID</span>
                    <span className="property-value">{node.id}</span>
                    <span className="property-key">Type</span>
                    <span className="property-value">{node.type}</span>
                    <span className="property-key">Provider</span>
                    <span className="property-value">{node.provider}</span>
                    <span className="property-key">Namespace</span>
                    <span className="property-value">{node.namespace || '—'}</span>
                    <span className="property-key">Status</span>
                    <span className="property-value"><StatusBadge status={node.status} /></span>
                    <span className="property-key">Discovered</span>
                    <span className="property-value">{new Date(node.discovered).toLocaleString()}</span>
                    <span className="property-key">Updated</span>
                    <span className="property-value">{new Date(node.updated).toLocaleString()}</span>
                  </div>
                </div>
              </div>

              <div className="card">
                <div className="card-header">Labels</div>
                <div className="card-body">
                  {node.labels && Object.keys(node.labels).length > 0 ? (
                    <div className="property-list">
                      {Object.entries(node.labels).map(([k, v]) => (
                        <span key={k} className="property-key" style={{ gridColumn: '1 / -1' }}>
                          <code style={{ color: 'var(--color-accent)' }}>{k}</code>
                          {' = '}
                          <code>{v}</code>
                        </span>
                      ))}
                    </div>
                  ) : (
                    <span style={{ color: 'var(--color-text-muted)', fontSize: 14 }}>No labels</span>
                  )}
                </div>
              </div>
            </div>

            <div className="detail-grid">
              <div className="card">
                <div className="card-header">
                  <ArrowRight size={14} style={{ display: 'inline', marginRight: 6 }} />
                  Outgoing Edges ({outgoing?.length || 0})
                </div>
                <div className="card-body">
                  {outgoing && outgoing.length > 0 ? (
                    <div className="table-wrapper">
                      <table>
                        <thead>
                          <tr><th>To</th><th>Type</th><th>Weight</th></tr>
                        </thead>
                        <tbody>
                          {outgoing.map((e, i) => (
                            <tr
                              key={i}
                              onClick={() => navigate(`/resources/${encodeURIComponent(e.to)}`)}
                            >
                              <td style={{ fontWeight: 600 }}>{e.to}</td>
                              <td><span className={`badge-type edge-${e.type}`}>{e.type}</span></td>
                              <td>{e.weight.toFixed(1)}</td>
                            </tr>
                          ))}
                        </tbody>
                      </table>
                    </div>
                  ) : (
                    <span style={{ color: 'var(--color-text-muted)', fontSize: 14 }}>None</span>
                  )}
                </div>
              </div>

              <div className="card">
                <div className="card-header">
                  <ArrowDown size={14} style={{ display: 'inline', marginRight: 6 }} />
                  Incoming Edges ({incoming?.length || 0})
                </div>
                <div className="card-body">
                  {incoming && incoming.length > 0 ? (
                    <div className="table-wrapper">
                      <table>
                        <thead>
                          <tr><th>From</th><th>Type</th><th>Weight</th></tr>
                        </thead>
                        <tbody>
                          {incoming.map((e, i) => (
                            <tr
                              key={i}
                              onClick={() => navigate(`/resources/${encodeURIComponent(e.from)}`)}
                            >
                              <td style={{ fontWeight: 600 }}>{e.from}</td>
                              <td><span className={`badge-type edge-${e.type}`}>{e.type}</span></td>
                              <td>{e.weight.toFixed(1)}</td>
                            </tr>
                          ))}
                        </tbody>
                      </table>
                    </div>
                  ) : (
                    <span style={{ color: 'var(--color-text-muted)', fontSize: 14 }}>None</span>
                  )}
                </div>
              </div>
            </div>
          </>
        )}

        {tab === 'impact' && (
          <>
            <div style={{ marginBottom: 16, display: 'flex', gap: 8 }}>
              <button
                className={`btn${impactDir === 'forward' ? ' btn-primary' : ''}`}
                onClick={() => setImpactDir('forward')}
              >
                <ArrowRight size={16} /> Forward
              </button>
              <button
                className={`btn${impactDir === 'reverse' ? ' btn-primary' : ''}`}
                onClick={() => setImpactDir('reverse')}
              >
                <ArrowLeft size={16} /> Reverse
              </button>
            </div>

            <p style={{ fontSize: 14, color: 'var(--color-text-muted)', marginBottom: 16 }}>
              {impactDir === 'forward'
                ? 'What does this resource affect? (follows outgoing edges)'
                : 'What does this resource depend on? (follows incoming edges)'}
            </p>

            {impact && impact.affected && impact.affected.length > 0 ? (
              <div className="impact-list">
                {impact.affected.map((item, i) => (
                  <div
                    key={i}
                    className="impact-item"
                    onClick={() => navigate(`/resources/${encodeURIComponent(item.node.id)}`)}
                  >
                    <div className="impact-depth">{item.depth}</div>
                    <div className="impact-info">
                      <div className="node-id">
                        <TypeBadge type={item.node.type} />{' '}
                        {item.node.id}
                      </div>
                      <div className="via-text">
                        via {item.via_id} ({item.edge.type})
                      </div>
                    </div>
                    <StatusBadge status={item.node.status} />
                  </div>
                ))}
              </div>
            ) : (
              <div style={{ color: 'var(--color-text-muted)', fontSize: 14, padding: 24, textAlign: 'center' }}>
                No {impactDir === 'forward' ? 'affected' : 'dependency'} nodes found
              </div>
            )}
          </>
        )}
      </div>
    </>
  );
}
