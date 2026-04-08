import { useEffect, useState, useCallback, useRef } from 'react';
import { useNavigate } from 'react-router-dom';
import ForceGraph2D from 'react-force-graph-2d';
import { getGraph } from '../api/client';
import type { Node, Edge } from '../api/types';
import { nodeColor, edgeColor, NODE_TYPES } from '../utils/colors';
import StatusBadge from '../components/StatusBadge';
import TypeBadge from '../components/TypeBadge';

interface GraphNode {
  id: string;
  type: string;
  provider: string;
  namespace: string;
  status: string;
  labels: Record<string, string> | null;
  x?: number;
  y?: number;
  [key: string]: unknown;
}

interface GraphLink {
  source: string;
  target: string;
  edgeType: string;
  weight: number;
  [key: string]: unknown;
}

export default function GraphView() {
  const [nodes, setNodes] = useState<GraphNode[]>([]);
  const [links, setLinks] = useState<GraphLink[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [hovered, setHovered] = useState<GraphNode | null>(null);
  const containerRef = useRef<HTMLDivElement>(null);
  const [dimensions, setDimensions] = useState({ width: 800, height: 600 });
  const navigate = useNavigate();

  useEffect(() => {
    getGraph()
      .then((data) => {
        setNodes(
          data.nodes.map((n: Node) => ({
            id: n.id,
            type: n.type,
            provider: n.provider,
            namespace: n.namespace,
            status: n.status,
            labels: n.labels,
          })),
        );
        setLinks(
          data.edges.map((e: Edge) => ({
            source: e.from,
            target: e.to,
            edgeType: e.type,
            weight: e.weight,
          })),
        );
      })
      .catch((err) => setError(String(err)))
      .finally(() => setLoading(false));
  }, []);

  useEffect(() => {
    function updateSize() {
      if (containerRef.current) {
        setDimensions({
          width: containerRef.current.clientWidth,
          height: containerRef.current.clientHeight,
        });
      }
    }
    updateSize();
    window.addEventListener('resize', updateSize);
    return () => window.removeEventListener('resize', updateSize);
  }, []);

  const handleNodeClick = useCallback(
    (node: unknown) => {
      const n = node as GraphNode;
      navigate(`/resources/${encodeURIComponent(n.id)}`);
    },
    [navigate],
  );

  const handleNodeHover = useCallback((_node: unknown | null) => {
    setHovered(_node ? (_node as GraphNode) : null);
  }, []);

  const paintNode = useCallback(
    (node: unknown, ctx: CanvasRenderingContext2D, globalScale: number) => {
      const n = node as GraphNode & { x: number; y: number };
      const color = nodeColor(n.type);
      const size = 6;
      const fontSize = Math.max(11 / globalScale, 1.5);

      // Glow effect
      ctx.shadowColor = color;
      ctx.shadowBlur = hovered?.id === n.id ? 12 : 4;

      // Node circle
      ctx.beginPath();
      ctx.arc(n.x, n.y, size, 0, 2 * Math.PI);
      ctx.fillStyle = color;
      ctx.fill();

      ctx.shadowBlur = 0;

      // Border
      ctx.strokeStyle = hovered?.id === n.id ? '#3b3d45' : 'rgba(0,0,0,0.15)';
      ctx.lineWidth = hovered?.id === n.id ? 2 / globalScale : 0.5 / globalScale;
      ctx.stroke();

      // Label
      if (globalScale > 0.6) {
        const label = n.id.split('/').pop() || n.id;
        ctx.font = `${fontSize}px -apple-system, sans-serif`;
        ctx.textAlign = 'center';
        ctx.textBaseline = 'top';
        ctx.fillStyle = 'rgba(59,61,69,0.9)';
        ctx.fillText(label, n.x, n.y + size + 2);
      }
    },
    [hovered],
  );

  if (loading) return <div className="loading">Loading graph...</div>;
  if (error) return <div className="page-body"><div className="error-banner">{error}</div></div>;

  return (
    <div style={{ display: 'flex', flexDirection: 'column', height: '100%' }}>
      <div className="page-header">
        <h2>Infrastructure Graph</h2>
        <p>
          Interactive dependency map — {nodes.length} nodes, {links.length} edges.
          Click a node to view details.
        </p>
      </div>
      <div className="graph-container" ref={containerRef}>
        <ForceGraph2D
          graphData={{ nodes, links }}
          width={dimensions.width}
          height={dimensions.height}
          nodeCanvasObject={paintNode}
          nodeCanvasObjectMode={() => 'replace'}
          linkColor={(link: unknown) => {
            const l = link as GraphLink;
            return edgeColor(l.edgeType);
          }}
          linkWidth={(link: unknown) => {
            const l = link as GraphLink;
            return l.weight * 2;
          }}
          linkDirectionalArrowLength={4}
          linkDirectionalArrowRelPos={0.85}
          linkCurvature={0.15}
          linkLabel={(link: unknown) => {
            const l = link as GraphLink;
            return l.edgeType;
          }}
          onNodeClick={handleNodeClick}
          onNodeHover={handleNodeHover}
          backgroundColor="#f5f6f7"
          cooldownTicks={100}
          d3VelocityDecay={0.3}
        />

        {/* Legend */}
        <div className="graph-legend">
          <h4>Node Types</h4>
          {NODE_TYPES.map((t) => (
            <div className="legend-item" key={t}>
              <div className="legend-dot" style={{ background: nodeColor(t) }} />
              <span>{t}</span>
            </div>
          ))}
        </div>

        {/* Hover tooltip */}
        {hovered && (
          <div className="graph-tooltip">
            <h3>{hovered.id}</h3>
            <div className="detail-row">
              <span className="detail-label">Type</span>
              <TypeBadge type={hovered.type} />
            </div>
            <div className="detail-row">
              <span className="detail-label">Provider</span>
              <span>{hovered.provider}</span>
            </div>
            <div className="detail-row">
              <span className="detail-label">Namespace</span>
              <span>{hovered.namespace}</span>
            </div>
            <div className="detail-row">
              <span className="detail-label">Status</span>
              <StatusBadge status={hovered.status} />
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
