/**
 * Types and fetch wrapper for the /elementdata.json endpoint.
 */

export interface NodeData {
  id: string;
  label: string;
  group?: string;
  cluster?: number;
  parent?: string;
  show?: boolean;
  [key: string]: unknown;
}

export interface EdgeData {
  id: string;
  source: string;
  target: string;
  type?: string;
  igp_metric?: string;
  adjacency_sid?: string;
  show?: boolean;
  [key: string]: unknown;
}

export interface Node {
  data: NodeData;
  selectable?: boolean;
}

export interface Edge {
  data: EdgeData;
  selectable?: boolean;
}

export interface ElementData {
  nodes: Node[];
  edges: Edge[];
}

/**
 * Fetch element data from the backend, optionally filtered by src/dst.
 */
export async function fetchElementData(
  src?: string,
  dst?: string,
): Promise<ElementData> {
  const params = new URLSearchParams();
  if (src) params.set('src', src);
  if (dst) params.set('dst', dst);
  const qs = params.toString();
  const url = `/elementdata.json${qs ? '?' + qs : ''}`;
  const res = await fetch(url);
  if (!res.ok) throw new Error(`HTTP ${res.status}: ${res.statusText}`);
  return res.json() as Promise<ElementData>;
}

/**
 * Filter nodes and edges by their `show` attribute.
 * Mirrors original behaviour:
 *  - Nodes with `show === undefined` are kept (default visible).
 *  - Nodes with `show === false` are removed.
 *  - Edges are kept only when `show === true`.
 */
export function filterByShow(data: ElementData): ElementData {
  return {
    nodes: data.nodes.filter((node) => {
      if (node.data.show === undefined) return true;
      return node.data.show;
    }),
    edges: data.edges.filter((edge) => edge.data.show === true),
  };
}
