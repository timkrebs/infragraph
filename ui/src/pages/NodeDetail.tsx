import { useEffect, useState } from 'react';
import { useParams, useNavigate, Link } from 'react-router-dom';
import { ArrowLeft, ArrowRight, ArrowDown } from 'lucide-react';
import { getNode, getImpact } from '../api/client';
import type { NodeDetail as NodeDetailType, ImpactResult } from '../api/types';
import StatusBadge from '../components/StatusBadge';
import TypeBadge from '../components/TypeBadge';
import Card from '../components/Card';
import EmptyState from '../components/EmptyState';

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

  if (loading) {
    return (
      <div className="flex h-64 items-center justify-center text-sm text-neutral-400">
        Loading node…
      </div>
    );
  }

  if (error) {
    return (
      <div className="p-6">
        <div className="rounded-md border border-danger-border bg-danger-bg px-4 py-3 text-sm text-danger">{error}</div>
      </div>
    );
  }

  if (!detail) return null;
  const { node, outgoing, incoming } = detail;

  return (
    <div className="h-full overflow-auto">
      {/* Page header */}
      <div className="border-b border-neutral-200 bg-white px-6 py-5">
        <div className="flex items-center gap-3">
          <Link to="/resources" className="text-neutral-400 hover:text-brand">
            <ArrowLeft size={20} />
          </Link>
          <div>
            <h2 className="text-xl font-semibold text-neutral-700">{node.id}</h2>
            <div className="mt-1 flex items-center gap-2">
              <TypeBadge type={node.type} />
              <StatusBadge status={node.status} />
              <span className="text-sm text-neutral-500">via {node.provider}</span>
            </div>
          </div>
        </div>
      </div>

      <div className="p-6">
        {/* Tabs (Vault-style) */}
        <div className="mb-5 flex gap-0 border-b-2 border-neutral-200">
          {(['overview', 'impact'] as const).map((t) => (
            <button
              key={t}
              onClick={() => setTab(t)}
              className={`-mb-[2px] border-b-2 px-5 py-2.5 text-sm font-medium transition ${
                tab === t
                  ? 'border-brand text-brand'
                  : 'border-transparent text-neutral-500 hover:text-neutral-700'
              }`}
            >
              {t === 'overview' ? 'Overview' : 'Impact Analysis'}
            </button>
          ))}
        </div>

        {tab === 'overview' && (
          <>
            <div className="mb-5 grid grid-cols-1 gap-4 md:grid-cols-2">
              <Card title="Properties">
                <dl className="grid grid-cols-[120px_1fr] gap-x-4 gap-y-2 text-sm">
                  <dt className="font-medium text-neutral-500">ID</dt>
                  <dd className="break-all text-neutral-700">{node.id}</dd>
                  <dt className="font-medium text-neutral-500">Type</dt>
                  <dd className="text-neutral-700">{node.type}</dd>
                  <dt className="font-medium text-neutral-500">Provider</dt>
                  <dd className="text-neutral-700">{node.provider}</dd>
                  <dt className="font-medium text-neutral-500">Namespace</dt>
                  <dd className="text-neutral-700">{node.namespace || '—'}</dd>
                  <dt className="font-medium text-neutral-500">Status</dt>
                  <dd><StatusBadge status={node.status} /></dd>
                  <dt className="font-medium text-neutral-500">Discovered</dt>
                  <dd className="text-neutral-700">{new Date(node.discovered).toLocaleString()}</dd>
                  <dt className="font-medium text-neutral-500">Updated</dt>
                  <dd className="text-neutral-700">{new Date(node.updated).toLocaleString()}</dd>
                </dl>
              </Card>

              <Card title="Labels">
                {node.labels && Object.keys(node.labels).length > 0 ? (
                  <div className="space-y-1.5 text-sm">
                    {Object.entries(node.labels).map(([k, v]) => (
                      <div key={k}>
                        <code className="font-mono text-xs text-brand">{k}</code>
                        <span className="text-neutral-400"> = </span>
                        <code className="font-mono text-xs text-neutral-700">{v}</code>
                      </div>
                    ))}
                  </div>
                ) : (
                  <p className="text-sm text-neutral-400">No labels</p>
                )}
              </Card>
            </div>

            {/* Edge tables */}
            <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
              <Card title={`Outgoing Edges (${outgoing?.length || 0})`}>
                {outgoing && outgoing.length > 0 ? (
                  <div className="overflow-x-auto">
                    <table className="w-full text-sm">
                      <thead>
                        <tr className="border-b border-neutral-200">
                          <th className="pb-2 text-left text-xs font-semibold uppercase tracking-wider text-neutral-500">To</th>
                          <th className="pb-2 text-left text-xs font-semibold uppercase tracking-wider text-neutral-500">Type</th>
                          <th className="pb-2 text-left text-xs font-semibold uppercase tracking-wider text-neutral-500">Weight</th>
                        </tr>
                      </thead>
                      <tbody className="divide-y divide-neutral-100">
                        {outgoing.map((e, i) => (
                          <tr
                            key={i}
                            onClick={() => navigate(`/resources/${encodeURIComponent(e.to)}`)}
                            className="cursor-pointer hover:bg-neutral-50"
                          >
                            <td className="py-2 font-semibold text-neutral-700">{e.to}</td>
                            <td className="py-2">
                              <span className="rounded border border-neutral-200 bg-neutral-50 px-1.5 py-0.5 font-mono text-xs text-neutral-600">
                                {e.type}
                              </span>
                            </td>
                            <td className="py-2 text-neutral-500">{e.weight.toFixed(1)}</td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                ) : (
                  <p className="text-sm text-neutral-400">None</p>
                )}
              </Card>

              <Card title={`Incoming Edges (${incoming?.length || 0})`}>
                {incoming && incoming.length > 0 ? (
                  <div className="overflow-x-auto">
                    <table className="w-full text-sm">
                      <thead>
                        <tr className="border-b border-neutral-200">
                          <th className="pb-2 text-left text-xs font-semibold uppercase tracking-wider text-neutral-500">From</th>
                          <th className="pb-2 text-left text-xs font-semibold uppercase tracking-wider text-neutral-500">Type</th>
                          <th className="pb-2 text-left text-xs font-semibold uppercase tracking-wider text-neutral-500">Weight</th>
                        </tr>
                      </thead>
                      <tbody className="divide-y divide-neutral-100">
                        {incoming.map((e, i) => (
                          <tr
                            key={i}
                            onClick={() => navigate(`/resources/${encodeURIComponent(e.from)}`)}
                            className="cursor-pointer hover:bg-neutral-50"
                          >
                            <td className="py-2 font-semibold text-neutral-700">{e.from}</td>
                            <td className="py-2">
                              <span className="rounded border border-neutral-200 bg-neutral-50 px-1.5 py-0.5 font-mono text-xs text-neutral-600">
                                {e.type}
                              </span>
                            </td>
                            <td className="py-2 text-neutral-500">{e.weight.toFixed(1)}</td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                ) : (
                  <p className="text-sm text-neutral-400">None</p>
                )}
              </Card>
            </div>
          </>
        )}

        {tab === 'impact' && (
          <>
            <div className="mb-4 flex items-center gap-3">
              <button
                onClick={() => setImpactDir('forward')}
                className={`inline-flex items-center gap-1.5 rounded-md border px-4 py-2 text-sm font-medium transition ${
                  impactDir === 'forward'
                    ? 'border-brand bg-brand text-white'
                    : 'border-neutral-200 bg-white text-neutral-600 hover:border-brand hover:text-brand'
                }`}
              >
                <ArrowRight size={14} /> Forward
              </button>
              <button
                onClick={() => setImpactDir('reverse')}
                className={`inline-flex items-center gap-1.5 rounded-md border px-4 py-2 text-sm font-medium transition ${
                  impactDir === 'reverse'
                    ? 'border-brand bg-brand text-white'
                    : 'border-neutral-200 bg-white text-neutral-600 hover:border-brand hover:text-brand'
                }`}
              >
                <ArrowLeft size={14} /> Reverse
              </button>
            </div>

            <p className="mb-4 text-sm text-neutral-500">
              {impactDir === 'forward'
                ? 'What does this resource affect? (follows outgoing edges)'
                : 'What does this resource depend on? (follows incoming edges)'}
            </p>

            {impact && impact.affected && impact.affected.length > 0 ? (
              <div className="flex flex-col gap-2">
                {impact.affected.map((item, i) => (
                  <button
                    key={i}
                    onClick={() => navigate(`/resources/${encodeURIComponent(item.node.id)}`)}
                    className="flex items-center gap-3 rounded-lg border border-neutral-200 bg-white px-4 py-3 text-left transition hover:border-brand"
                  >
                    <span className="flex h-7 w-7 shrink-0 items-center justify-center rounded-full bg-brand-light text-xs font-bold text-brand">
                      {item.depth}
                    </span>
                    <div className="min-w-0 flex-1">
                      <div className="flex items-center gap-2">
                        <TypeBadge type={item.node.type} />
                        <span className="truncate text-sm font-semibold text-neutral-700">{item.node.id}</span>
                      </div>
                      <div className="mt-0.5 truncate text-xs text-neutral-400">
                        via {item.via_id} ({item.edge.type})
                      </div>
                    </div>
                    <ArrowDown size={14} className="shrink-0 text-neutral-300" />
                  </button>
                ))}
              </div>
            ) : (
              <EmptyState
                icon={<ArrowDown size={20} />}
                title="No impacted resources"
                description={`No ${impactDir} dependencies found from this resource.`}
              />
            )}
          </>
        )}
      </div>
    </div>
  );
}
