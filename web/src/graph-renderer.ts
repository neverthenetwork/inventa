/**
 * Cytoscape graph renderer using the cise clustered layout.
 * Mirrors the exact styling and layout from the original index.html.
 */

import cytoscape, { type Core } from 'cytoscape';
import cise from 'cytoscape-cise';
import type { ElementData } from './element-data';

cytoscape.use(cise);

/**
 * Create a Cytoscape instance with the cise clustered layout.
 */
export function createGraph(
  container: HTMLElement,
  elements: ElementData,
): Core {
  const cy = cytoscape({
    container,

    layout: {
      name: 'cise',
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      clusters: (node: any) => node.data('cluster') as number | undefined,
      nodeSeparation: 50,
      allowNodesInCircle: true,
      idealInterClusterEdgeLengthCoefficient: 4.5,
    } as cytoscape.LayoutOptions,

    style: [
      {
        selector: 'node',
        style: {
          label: 'data(label)',
          'font-size': '12px',
          'background-color': '#ea8a31',
          'text-halign': 'center',
          'text-valign': 'bottom',
          padding: '20px',
        },
      },
      {
        selector: 'edge',
        style: {
          'font-size': '6px',
          'curve-style': 'bezier',
          'control-point-step-size': 10,
          'target-arrow-shape': 'triangle',
          'target-arrow-color': '#fcc694',
          width: 2,
          opacity: 0.666,
          'line-color': '#fcc694',
        },
      },
    ],

    elements,
  });

  // Expose on window for debugging parity with original
  (window as unknown as Record<string, unknown>).cy = cy;

  return cy;
}
