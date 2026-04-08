import { useEffect, useState, useRef, useCallback } from 'react';
import { useNavigate } from 'react-router-dom';
import cytoscape from 'cytoscape';
import { getGraph } from '../api/client';
import type { Node, Edge } from '../api/types';
import { nodeColor, edgeColor, NODE_TYPES } from '../utils/colors';

export default function GraphView() {
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [nodeCount, setNodeCount] = useState(0);
  const [edgeCount, setEdgeCount] = useState(0);
  const [hovered, setHovered] = useState<{ id: string; type: string; provider: string; namespace: string; status: string } | null>(null);
  const [layout, setLayout] = useState<'cose' | 'breadthfirst' | 'circle'>('cose');
  const containerRef = useRef<HTMLDivElement>(null);
  const cyRef = useRef<cytoscape.Core | null>(null);
  const navigate = useNavigate();

  // Build Cytoscape instance
  const buildGraph = useCallback(
    (nodes: Node[], edges: Edge[], layoutName: string) => {
      if (!containerRef.current) return;
      if (cyRef.current) cyRef.current.destroy();

      const elements: cytoscape.ElementDefinition[] = [
        ...nodes.map((n) => ({
          data: {
            id: n.id,
            label: n.id.split('/').pop() || n.id,
            nodeType: n.type,
            provider: n.provider,
            namespace: n.namespace,
            status: n.status,
          },
        })),
        ...edges.map((e, i) => ({
          data: {
            id: `e${i}`,
            source: e.from,
            target: e.to,
            edgeType: e.type,
            weight: e.weight,
          },
        })),
      ];

      const cy = cytoscape({
        container: containerRef.current,
        elements,
        style: [
          {
            selector: 'node',
            style: {
              label: 'data(label)',
              width: 28,
              height: 28,
              'background-color': (ele) => nodeColor(ele.data('nodeType')),
              'border-width': 2,
              'border-color': '#ffffff',
              'font-size': '10px',
              'font-family': '-apple-system, BlinkMacSystemFont, sans-serif',
              color: '#3b3d45',
              'text-valign': 'bottom',
              'text-margin-y': 4,
              'text-outline-color': '#ffffff',
              'text-outline-width': 2,
            } as cytoscape.Css.Node,
          },
          {
            selector: 'node:active',
            style: { 'overlay-opacity': 0.1 } as cytoscape.Css.Node,
          },
          {
            selector: 'node.highlight',
            style: {
              'border-width': 3,
              'border-color': '#1060ff',
              width: 34,
              height: 34,
            } as cytoscape.Css.Node,
          },
          {
            selector: 'edge',
            style: {
              width: 2,
              'line-color': (ele) => edgeColor(ele.data('edgeType')),
              'target-arrow-color': (ele) => edgeColor(ele.data('edgeType')),
              'target-arrow-shape': 'triangle',
              'curve-style': 'bezier',
              opacity: 0.7,
            } as cytoscape.Css.Edge,
          },
          {
            selector: 'edge.highlight',
            style: { opacity: 1, width: 3 } as cytoscape.Css.Edge,
          },
          {
            selector: '.faded',
            style: { opacity: 0.15 } as cytoscape.Css.Node,
          },
        ],
        layout: { name: layoutName, animate: true, animationDuration: 500 } as cytoscape.LayoutOptions,
        minZoom: 0.2,
        maxZoom: 5,
      });

      // Click → navigate to detail
      cy.on('tap', 'node', (evt) => {
        navigate(`/resources/${encodeURIComponent(evt.target.id())}`);
      });

      // Hover → highlight subtree + set hovered
      cy.on('mouseover', 'node', (evt) => {
        const node = evt.target;
        setHovered({
          id: node.id(),
          type: node.data('nodeType'),
          provider: node.data('provider'),
          namespace: node.data('namespace'),
          status: node.data('status'),
        });
        // Highlight connected
        const neighborhood = node.closedNeighborhood();
        cy.elements().addClass('faded');
        neighborhood.removeClass('faded').addClass('highlight');
      });

      cy.on('mouseout', 'node', () => {
        setHovered(null);
        cy.elements().removeClass('faded highlight');
      });

      cyRef.current = cy;
    },
    [navigate],
  );

  // Fetch data
  useEffect(() => {
    getGraph()
      .then((data) => {
        setNodeCount(data.nodes.length);
        setEdgeCount(data.edges.length);
        buildGraph(data.nodes, data.edges, layout);
      })
      .catch((err) => setError(String(err)))
      .finally(() => setLoading(false));
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // Re-layout when layout mode changes
  useEffect(() => {
    if (!cyRef.current) return;
    cyRef.current.layout({ name: layout, animate: true, animationDuration: 500 } as cytoscape.LayoutOptions).run();
  }, [layout]);

  if (loading) {
    return (
      <div className="flex h-64 items-center justify-center text-sm text-neutral-400">
        Loading graph…
      </div>
    );
  }

  if (error) {
    return (
      <div className="p-6">
        <div className="rounded-md border border-danger-border bg-danger-bg px-4 py-3 text-sm text-danger">
          {error}
        </div>
      </div>
    );
  }

  return (
    <div className="flex h-full flex-col">
      {/* Page header */}
      <div className="border-b border-neutral-200 bg-white px-6 py-5">
        <h2 className="text-xl font-semibold text-neutral-700">Infrastructure Graph</h2>
        <p className="mt-1 text-sm text-neutral-500">
          Interactive dependency map — {nodeCount} nodes, {edgeCount} edges. Click a node to view details.
        </p>
      </div>

      {/* Layout controls */}
      <div className="flex items-center gap-2 border-b border-neutral-200 bg-neutral-50 px-6 py-2">
        <span className="text-xs font-semibold uppercase tracking-wider text-neutral-500">Layout:</span>
        {(['cose', 'breadthfirst', 'circle'] as const).map((l) => (
          <button
            key={l}
            onClick={() => setLayout(l)}
            className={`rounded-md border px-3 py-1 text-xs font-medium transition ${
              layout === l
                ? 'border-brand bg-brand text-white'
                : 'border-neutral-200 bg-white text-neutral-600 hover:border-brand hover:text-brand'
            }`}
          >
            {l === 'cose' ? 'Force-Directed' : l === 'breadthfirst' ? 'Hierarchical' : 'Circle'}
          </button>
        ))}
        <button
          onClick={() => cyRef.current?.fit(undefined, 40)}
          className="ml-auto rounded-md border border-neutral-200 bg-white px-3 py-1 text-xs font-medium text-neutral-600 hover:border-brand hover:text-brand"
        >
          Fit
        </button>
      </div>

      {/* Graph container */}
      <div className="relative flex-1 bg-neutral-50">
        <div ref={containerRef} className="cy-container" />

        {/* Legend */}
        <div className="absolute left-4 top-4 z-10 rounded-lg border border-neutral-200 bg-white p-3 shadow-sm">
          <h4 className="mb-2 text-[11px] font-semibold uppercase tracking-wider text-neutral-500">
            Node Types
          </h4>
          {NODE_TYPES.map((t) => (
            <div key={t} className="mb-1 flex items-center gap-2 text-xs text-neutral-600">
              <span className="inline-block h-2.5 w-2.5 rounded-full" style={{ background: nodeColor(t) }} />
              <span className="capitalize">{t}</span>
            </div>
          ))}
        </div>

        {/* Hover tooltip */}
        {hovered && (
          <div className="absolute right-4 top-4 z-10 min-w-[220px] rounded-lg border border-neutral-200 bg-white p-4 shadow-sm">
            <h3 className="mb-2 break-all text-sm font-semibold text-neutral-700">{hovered.id}</h3>
            <dl className="grid grid-cols-[80px_1fr] gap-x-3 gap-y-1 text-xs">
              <dt className="text-neutral-500">Type</dt>
              <dd className="text-neutral-700">{hovered.type}</dd>
              <dt className="text-neutral-500">Provider</dt>
              <dd className="text-neutral-700">{hovered.provider}</dd>
              <dt className="text-neutral-500">Namespace</dt>
              <dd className="text-neutral-700">{hovered.namespace || '—'}</dd>
              <dt className="text-neutral-500">Status</dt>
              <dd className="text-neutral-700">{hovered.status}</dd>
            </dl>
          </div>
        )}
      </div>
    </div>
  );
}
