import { useEffect, useState } from 'react';
import { getStatus } from '../api/client';
import type { SystemStatus } from '../api/types';
import Card from '../components/Card';

export default function SettingsPage() {
  const [status, setStatus] = useState<SystemStatus | null>(null);

  useEffect(() => {
    getStatus()
      .then(setStatus)
      .catch(() => {});
  }, []);

  return (
    <div className="h-full overflow-auto">
      {/* Page header */}
      <div className="border-b border-neutral-200 bg-white px-6 py-5">
        <h2 className="text-xl font-semibold text-neutral-700">Settings</h2>
        <p className="mt-1 text-sm text-neutral-500">Server configuration and diagnostics</p>
      </div>

      <div className="p-6">
        <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
          <Card title="Server Info">
            <dl className="grid grid-cols-[120px_1fr] gap-x-4 gap-y-2 text-sm">
              <dt className="font-medium text-neutral-500">Version</dt>
              <dd className="text-neutral-700">{status?.version ?? '—'}</dd>
              <dt className="font-medium text-neutral-500">Store Path</dt>
              <dd className="break-all font-mono text-xs text-neutral-700">{status?.store_path ?? '—'}</dd>
              <dt className="font-medium text-neutral-500">Nodes</dt>
              <dd className="text-neutral-700">{status?.node_count ?? '—'}</dd>
              <dt className="font-medium text-neutral-500">Edges</dt>
              <dd className="text-neutral-700">{status?.edge_count ?? '—'}</dd>
            </dl>
          </Card>

          <Card title="API Endpoints">
            <div className="space-y-2 text-sm">
              {[
                ['GET /v1/sys/status', 'System status'],
                ['GET /v1/graph', 'Full graph (nodes + edges)'],
                ['GET /v1/resources', 'List resources'],
                ['GET /v1/graph/node/:id', 'Node detail'],
                ['GET /v1/graph/impact/:id', 'Impact analysis'],
                ['GET /v1/collectors', 'List collectors'],
                ['POST /v1/collector/events', 'Push events'],
                ['POST /v1/collector/register', 'Register collector'],
              ].map(([endpoint, desc]) => (
                <div key={endpoint} className="flex items-center justify-between">
                  <code className="font-mono text-xs text-brand">{endpoint}</code>
                  <span className="text-xs text-neutral-400">{desc}</span>
                </div>
              ))}
            </div>
          </Card>
        </div>
      </div>
    </div>
  );
}
