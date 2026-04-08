import { useState, useEffect } from 'react';
import { Radio, RefreshCw, AlertCircle, Loader2 } from 'lucide-react';
import Card from '../components/Card';
import EmptyState from '../components/EmptyState';
import StatusBadge from '../components/StatusBadge';
import { getCollectors } from '../api/client';
import type { CollectorInfo } from '../api/types';

function formatTime(iso: string): string {
  if (!iso) return 'never';
  const d = new Date(iso);
  if (isNaN(d.getTime())) return 'never';
  const diff = Date.now() - d.getTime();
  if (diff < 60_000) return 'just now';
  if (diff < 3_600_000) return `${Math.floor(diff / 60_000)} min ago`;
  if (diff < 86_400_000) return `${Math.floor(diff / 3_600_000)} hr ago`;
  return d.toLocaleDateString();
}

function collectorStatusToDisplay(status: CollectorInfo['status']): string {
  if (status === 'running') return 'healthy';
  if (status === 'error') return 'degraded';
  return status;
}

export default function Collectors() {
  const [collectors, setCollectors] = useState<CollectorInfo[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  const load = () => {
    setLoading(true);
    setError('');
    getCollectors()
      .then(setCollectors)
      .catch((e) => setError(e.message || 'Failed to load collectors'))
      .finally(() => setLoading(false));
  };

  useEffect(() => {
    load();
    const id = setInterval(load, 15_000);
    return () => clearInterval(id);
  }, []);

  return (
    <div className="h-full overflow-auto">
      {/* Page header */}
      <div className="border-b border-neutral-200 bg-white px-6 py-5">
        <div className="flex items-center justify-between">
          <div>
            <h2 className="text-xl font-semibold text-neutral-700">Collectors</h2>
            <p className="mt-1 text-sm text-neutral-500">
              Active infrastructure collectors registered with this server
            </p>
          </div>
          <button
            onClick={load}
            disabled={loading}
            className="flex items-center gap-2 rounded-md border border-neutral-200 bg-white px-3 py-1.5 text-sm text-neutral-600 transition hover:border-brand hover:text-brand disabled:opacity-50"
          >
            <RefreshCw size={14} className={loading ? 'animate-spin' : ''} />
            Refresh
          </button>
        </div>
      </div>

      <div className="p-6">
        {error && (
          <div className="mb-4 flex items-center gap-2 rounded-md border border-danger-border bg-danger-bg px-4 py-3 text-sm text-danger">
            <AlertCircle size={16} />
            {error}
          </div>
        )}

        {loading && collectors.length === 0 ? (
          <div className="flex items-center justify-center py-16 text-neutral-400">
            <Loader2 size={24} className="animate-spin" />
          </div>
        ) : collectors.length === 0 ? (
          <EmptyState
            icon={<Radio size={24} />}
            title="No collectors configured"
            description="Configure collectors in your infragraph.hcl file to start discovering infrastructure."
          />
        ) : (
          <div className="flex flex-col gap-4">
            {collectors.map((c) => (
              <Card key={c.name}>
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-4">
                    <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-neutral-100 text-neutral-500">
                      <Radio size={18} />
                    </div>
                    <div>
                      <div className="flex items-center gap-2">
                        <h3 className="text-sm font-semibold text-neutral-700">{c.name}</h3>
                        <span className="rounded border border-neutral-200 bg-neutral-50 px-1.5 py-0.5 font-mono text-[10px] uppercase text-neutral-500">
                          {c.type}
                        </span>
                      </div>
                      <div className="mt-0.5 flex items-center gap-3 text-xs text-neutral-400">
                        <span>Last sync: {formatTime(c.last_sync)}</span>
                        <span>·</span>
                        <span>{c.resources} resources</span>
                        {c.started_at && (
                          <>
                            <span>·</span>
                            <span>Started: {formatTime(c.started_at)}</span>
                          </>
                        )}
                      </div>
                    </div>
                  </div>
                  <StatusBadge status={collectorStatusToDisplay(c.status)} />
                </div>
                {c.error && (
                  <div className="mt-3 rounded-md border border-danger-border bg-danger-bg px-3 py-2 text-xs text-danger">
                    {c.error}
                  </div>
                )}
              </Card>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
