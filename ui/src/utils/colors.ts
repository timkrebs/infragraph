const NODE_COLORS: Record<string, string> = {
  ingress: '#e06010',
  service: '#1563ff',
  pod: '#2a8e38',
  secret: '#c73445',
  configmap: '#8a64d6',
  pvc: '#b07722',
  container: '#2176e8',
};

export function nodeColor(type: string): string {
  return NODE_COLORS[type] ?? '#656a76';
}

const EDGE_COLORS: Record<string, string> = {
  routes_to: '#e06010',
  selected_by: '#1563ff',
  mounts: '#8a64d6',
  depends_on: '#c73445',
};

export function edgeColor(type: string): string {
  return EDGE_COLORS[type] ?? '#8e94a0';
}

export const NODE_TYPES = Object.keys(NODE_COLORS);
export const EDGE_TYPES = Object.keys(EDGE_COLORS);
