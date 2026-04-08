/// <reference types="vite/client" />

declare module 'react-force-graph-2d' {
  import { Component } from 'react';

  interface GraphData {
    nodes: Array<{ id: string; [key: string]: unknown }>;
    links: Array<{ source: string; target: string; [key: string]: unknown }>;
  }

  interface ForceGraphProps {
    graphData: GraphData;
    width?: number;
    height?: number;
    nodeLabel?: string | ((node: unknown) => string);
    nodeColor?: string | ((node: unknown) => string);
    nodeVal?: string | ((node: unknown) => number);
    nodeRelSize?: number;
    nodeCanvasObject?: (node: unknown, ctx: CanvasRenderingContext2D, globalScale: number) => void;
    nodeCanvasObjectMode?: string | ((node: unknown) => string);
    linkColor?: string | ((link: unknown) => string);
    linkWidth?: number | ((link: unknown) => number);
    linkDirectionalArrowLength?: number | ((link: unknown) => number);
    linkDirectionalArrowRelPos?: number;
    linkLabel?: string | ((link: unknown) => string);
    linkCurvature?: number | ((link: unknown) => number);
    onNodeClick?: (node: unknown, event: MouseEvent) => void;
    onNodeHover?: (node: unknown | null, prevNode: unknown | null) => void;
    backgroundColor?: string;
    cooldownTicks?: number;
    d3VelocityDecay?: number;
    d3AlphaDecay?: number;
    dagMode?: string;
    dagLevelDistance?: number;
    onEngineStop?: () => void;
    enableZoomInteraction?: boolean;
    enablePanInteraction?: boolean;
    ref?: unknown;
  }

  export default class ForceGraph2D extends Component<ForceGraphProps> {}
}
