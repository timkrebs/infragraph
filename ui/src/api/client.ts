import type { GraphData, SystemStatus, NodeDetail, ImpactResult, CollectorInfo } from './types';

const BASE = '/v1';

async function fetchJSON<T>(path: string): Promise<T> {
  const res = await fetch(path);
  if (!res.ok) {
    const body = await res.json().catch(() => ({}));
    throw new Error((body as { error?: string }).error || res.statusText);
  }
  return res.json() as Promise<T>;
}

export async function getStatus(): Promise<SystemStatus> {
  return fetchJSON<SystemStatus>(`${BASE}/sys/status`);
}

export async function getGraph(): Promise<GraphData> {
  return fetchJSON<GraphData>(`${BASE}/graph`);
}

export async function getResources(type?: string, namespace?: string): Promise<{ nodes: import('./types').Node[] }> {
  const params = new URLSearchParams();
  if (type) params.set('type', type);
  if (namespace) params.set('namespace', namespace);
  const qs = params.toString();
  return fetchJSON(`${BASE}/resources${qs ? '?' + qs : ''}`);
}

export async function getNode(id: string): Promise<NodeDetail> {
  return fetchJSON<NodeDetail>(`${BASE}/graph/node/${encodeURIComponent(id)}`);
}

export async function getImpact(id: string, direction: 'forward' | 'reverse', depth = 10): Promise<ImpactResult> {
  return fetchJSON<ImpactResult>(
    `${BASE}/graph/impact/${encodeURIComponent(id)}?direction=${direction}&depth=${depth}`,
  );
}

export async function getCollectors(): Promise<CollectorInfo[]> {
  const data = await fetchJSON<{ collectors: CollectorInfo[] }>(`${BASE}/collectors`);
  return data.collectors;
}
