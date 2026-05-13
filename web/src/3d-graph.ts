/**
 * Shared utilities for 3D and VR force-graph views.
 */

import {
  fetchElementData,
  filterByShow,
  type NodeData,
  type EdgeData,
} from './element-data';

/** Pick a random integer in [0, max]. */
function randomInteger(max: number): number {
  return Math.floor(Math.random() * (max + 1));
}

/** Convert an RGB array [r,g,b] to a CSS hex string. */
export function RGBtoHEX(color: [number, number, number]): string {
  return (
    '#' +
    color
      .map((digit) => digit.toString(16).padStart(2, '0'))
      .join('')
  );
}

/** Generate a random hex colour string. */
export function randomRgbColor(): string {
  const r = randomInteger(255);
  const g = randomInteger(255);
  const b = randomInteger(255);
  return RGBtoHEX([r, g, b]);
}

/** Shape returned by the 3D graph libraries. */
export interface GraphData {
  nodes: NodeData[];
  links: EdgeData[];
}

/**
 * Fetch and transform element data into the shape expected by
 * 3d-force-graph / 3d-force-graph-vr.
 */
export async function fetchGraphData(
  src?: string,
  dst?: string,
): Promise<GraphData> {
  const raw = await fetchElementData(src, dst);
  const filtered = filterByShow(raw);
  return {
    nodes: filtered.nodes.map((n) => n.data),
    links: filtered.edges.map((e) => e.data),
  };
}

/**
 * Apply the shared graph configuration chain.
 * Accepts any graph instance (both 3D and VR share the same API shape).
 */
// eslint-disable-next-line @typescript-eslint/no-explicit-any
export function configureGraph(graph: any): any {
  return graph
    .linkWidth(2)
    .linkCurvature(0.3)
    .linkDirectionalParticles(5)
    .linkDirectionalParticleWidth(1.5)
    .linkDirectionalParticleColor(() => randomRgbColor())
    .nodeAutoColorBy('group');
}
