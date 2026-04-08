export interface Node {
  id: string;
  type: string;
  provider: string;
  namespace: string;
  labels: Record<string, string> | null;
  annotations: Record<string, string> | null;
  status: string;
  discovered: string;
  updated: string;
}

export interface Edge {
  from: string;
  to: string;
  type: string;
  weight: number;
}

export interface GraphData {
  nodes: Node[];
  edges: Edge[];
}

export interface SystemStatus {
  version: string;
  node_count: number;
  edge_count: number;
  store_path: string;
}

export interface NodeDetail {
  node: Node;
  outgoing: Edge[];
  incoming: Edge[];
  neighbors: Node[];
  predecessors: Node[];
}

export interface ImpactNode {
  node: Node;
  via_id: string;
  edge: Edge;
  depth: number;
}

export interface ImpactResult {
  root: Node;
  affected: ImpactNode[];
}
