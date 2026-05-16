/**
 * Cytoscape graph renderer using the cise clustered layout.
 * Dark theme styling for the Inventa topology viewer.
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
          color: '#e2e8f0',
          'text-halign': 'center',
          'text-valign': 'bottom',
          padding: '20px',
        },
      },
      // Group-specific node styles
      {
        selector: 'node[group="vpc"]',
        style: { 'background-color': '#4499a3', shape: 'round-rectangle' },
      },
      {
        selector: 'node[group="subnet"]',
        style: { 'background-color': '#5a8a4a', shape: 'rectangle' },
      },
      {
        selector: 'node[group="instance"]',
        style: { 'background-color': '#c2855b', shape: 'ellipse', width: 35, height: 35 },
      },
      {
        selector: 'node[group="elb"]',
        style: { 'background-color': '#a15b9e', shape: 'diamond' },
      },
      {
        selector: 'node[group="igw"]',
        style: { 'background-color': '#5b8aa1', shape: 'triangle' },
      },
      {
        selector: 'node[group="security_group"]',
        style: { 'background-color': '#c25b5b', shape: 'hexagon' },
      },
      {
        selector: 'node[group="internet"]',
        style: {
          'background-color': '#3498db',
          'border-style': 'dashed',
          'border-width': 3,
          'border-color': '#5dade2',
          shape: 'ellipse',
          width: 60,
          height: 60,
          'font-size': '14px',
          'font-weight': 'bold',
        },
      },
      {
        selector: 'edge',
        style: {
          'font-size': '6px',
          'line-color': '#fcc694',
          'target-arrow-color': '#fcc694',
          'target-arrow-shape': 'triangle',
          'curve-style': 'bezier',
          'control-point-step-size': 10,
          width: 2,
          opacity: 0.666,
          color: '#94a3b8',
        },
      },
    ],

    elements,
  });

  // Expose on window for debugging parity with original
  (window as unknown as Record<string, unknown>).cy = cy;

  return cy;
}
