import { useEffect, useState, useMemo } from 'react';
import { useNavigate } from 'react-router-dom';
import { Zap, ArrowRight, ArrowLeft, Search, AlertTriangle } from 'lucide-react';
import { getResources, getImpact } from '../api/client';
import type { Node, ImpactResult } from '../api/types';
import TypeBadge from '../components/TypeBadge';
import EmptyState from '../components/EmptyState';

function severityBadge(count: number) {
  if (count >= 10)
    return <span className="rounded-full bg-danger-bg border border-danger-border px-2 py-0.5 text-xs font-bold text-danger">CRITICAL</span>;
  if (count >= 5)
    return <span className="rounded-full bg-warning-bg border border-warning-border px-2 py-0.5 text-xs font-bold text-warning">HIGH</span>;
  if (count >= 2)
    return <span className="rounded-full bg-brand-light border border-neutral-200 px-2 py-0.5 text-xs font-bold text-brand">MEDIUM</span>;
  return <span className="rounded-full bg-success-bg border border-success-border px-2 py-0.5 text-xs font-bold text-success">LOW</span>;
}

export default function ImpactAnalysis() {
  const navigate = useNavigate();
  const [nodes, setNodes] = useState<Node[]>([]);
  const [search, setSearch] = useState('');
  const [selectedId, setSelectedId] = useState('');
  const [direction, setDirection] = useState<'forward' | 'reverse'>('forward');
  const [depth, setDepth] = useState(5);
  const [result, setResult] = useState<ImpactResult | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  useEffect(() => {
    getResources()
      .then((d) => setNodes(d.nodes || []))
      .catch(() => {});
  }, []);

  const filtered = useMemo(() => {
    if (!search) return nodes.slice(0, 20);
    return nodes.filter((n) => n.id.toLowerCase().includes(search.toLowerCase())).slice(0, 20);
  }, [nodes, search]);

  const runAnalysis = () => {
    if (!selectedId) return;
    setLoading(true);
    setError('');
    getImpact(selectedId, direction, depth)
      .then(setResult)
      .catch((err) => setError(String(err)))
      .finally(() => setLoading(false));
  };

  // Group affected by type
  const typeSummary = useMemo(() => {
    if (!result?.affected) return {};
    const counts: Record<string, number> = {};
    for (const a of result.affected) counts[a.node.type] = (counts[a.node.type] || 0) + 1;
    return counts;
  }, [result]);

  return (
    <div className="h-full overflow-auto">
      {/* Page header */}
      <div className="border-b border-neutral-200 bg-white px-6 py-5">
        <h2 className="text-xl font-semibold text-neutral-700">Impact Analysis</h2>
        <p className="mt-1 text-sm text-neutral-500">
          Explore the blast radius of any infrastructure resource
        </p>
      </div>

      <div className="p-6">
        {/* Controls */}
        <div className="mb-6 rounded-lg border border-neutral-200 bg-white p-5">
          <div className="mb-4 grid grid-cols-1 gap-4 md:grid-cols-3">
            {/* Resource selector */}
            <div>
              <label className="mb-1 block text-xs font-semibold uppercase tracking-wider text-neutral-500">
                Resource
              </label>
              <div className="relative">
                <Search size={14} className="absolute left-3 top-1/2 -translate-y-1/2 text-neutral-400" />
                <input
                  type="text"
                  placeholder="Search resource…"
                  value={search}
                  onChange={(e) => { setSearch(e.target.value); setSelectedId(''); }}
                  className="w-full rounded-md border border-neutral-200 bg-neutral-50 py-2 pl-9 pr-3 text-sm text-neutral-700 placeholder:text-neutral-400 focus:border-brand focus:outline-none"
                />
              </div>
              {search && !selectedId && (
                <div className="mt-1 max-h-40 overflow-y-auto rounded-md border border-neutral-200 bg-white shadow-sm">
                  {filtered.map((n) => (
                    <button
                      key={n.id}
                      onClick={() => { setSelectedId(n.id); setSearch(n.id); }}
                      className="flex w-full items-center gap-2 px-3 py-2 text-left text-sm hover:bg-neutral-50"
                    >
                      <TypeBadge type={n.type} />
                      <span className="truncate text-neutral-700">{n.id}</span>
                    </button>
                  ))}
                  {filtered.length === 0 && (
                    <p className="px-3 py-2 text-sm text-neutral-400">No matches</p>
                  )}
                </div>
              )}
            </div>

            {/* Direction */}
            <div>
              <label className="mb-1 block text-xs font-semibold uppercase tracking-wider text-neutral-500">
                Direction
              </label>
              <div className="flex gap-2">
                <button
                  onClick={() => setDirection('forward')}
                  className={`flex-1 inline-flex items-center justify-center gap-1.5 rounded-md border px-3 py-2 text-sm font-medium transition ${
                    direction === 'forward'
                      ? 'border-brand bg-brand text-white'
                      : 'border-neutral-200 bg-white text-neutral-600 hover:border-brand'
                  }`}
                >
                  <ArrowRight size={14} /> Forward
                </button>
                <button
                  onClick={() => setDirection('reverse')}
                  className={`flex-1 inline-flex items-center justify-center gap-1.5 rounded-md border px-3 py-2 text-sm font-medium transition ${
                    direction === 'reverse'
                      ? 'border-brand bg-brand text-white'
                      : 'border-neutral-200 bg-white text-neutral-600 hover:border-brand'
                  }`}
                >
                  <ArrowLeft size={14} /> Reverse
                </button>
              </div>
            </div>

            {/* Depth */}
            <div>
              <label className="mb-1 block text-xs font-semibold uppercase tracking-wider text-neutral-500">
                Depth: {depth}
              </label>
              <input
                type="range"
                min={1}
                max={10}
                value={depth}
                onChange={(e) => setDepth(Number(e.target.value))}
                className="mt-2 w-full accent-brand"
              />
              <div className="mt-1 flex justify-between text-[10px] text-neutral-400">
                <span>1</span><span>5</span><span>10</span>
              </div>
            </div>
          </div>

          <button
            onClick={runAnalysis}
            disabled={!selectedId || loading}
            className="inline-flex items-center gap-2 rounded-md border border-brand bg-brand px-5 py-2 text-sm font-semibold text-white transition hover:bg-brand-hover disabled:opacity-50"
          >
            <Zap size={14} />
            {loading ? 'Analyzing…' : 'Run Analysis'}
          </button>
        </div>

        {error && (
          <div className="mb-4 rounded-md border border-danger-border bg-danger-bg px-4 py-3 text-sm text-danger">
            {error}
          </div>
        )}

        {/* Results */}
        {result && (
          <>
            {/* Summary */}
            <div className="mb-4 flex items-center gap-4">
              <h3 className="text-base font-semibold text-neutral-700">
                Blast Radius: {result.affected?.length || 0} resources
              </h3>
              {severityBadge(result.affected?.length || 0)}
            </div>

            {/* Type breakdown */}
            {Object.keys(typeSummary).length > 0 && (
              <div className="mb-5 flex flex-wrap gap-2">
                {Object.entries(typeSummary).map(([type, count]) => (
                  <span key={type} className="inline-flex items-center gap-1.5 rounded-md border border-neutral-200 bg-white px-3 py-1.5 text-sm">
                    <TypeBadge type={type} />
                    <span className="font-bold text-neutral-600">{count}</span>
                  </span>
                ))}
              </div>
            )}

            {/* Affected list */}
            {result.affected && result.affected.length > 0 ? (
              <div className="flex flex-col gap-2">
                {result.affected.map((item, i) => (
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
                      <div className="mt-0.5 text-xs text-neutral-400">
                        via {item.via_id} ({item.edge.type})
                      </div>
                    </div>
                  </button>
                ))}
              </div>
            ) : (
              <EmptyState
                icon={<AlertTriangle size={20} />}
                title="No impacted resources"
                description={`No ${direction} dependencies found from this resource.`}
              />
            )}
          </>
        )}

        {!result && !error && (
          <EmptyState
            icon={<Zap size={24} />}
            title="Select a resource to analyze"
            description="Choose a resource above, set direction and depth, then run the analysis to see the blast radius."
          />
        )}
      </div>
    </div>
  );
}
