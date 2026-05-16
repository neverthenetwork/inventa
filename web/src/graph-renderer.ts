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
      // Edge type differentiation
      {
        // Containment: dashed, thin, grey, no arrow
        selector: 'edge[type="parent"]',
        style: {
          'line-style': 'dashed',
          'line-color': '#64748b',
          'target-arrow-shape': 'none',
          width: 1.5,
          opacity: 0.5,
          'curve-style': 'haystack',
        },
      },
      {
        // Security group membership: dotted, thin, red
        selector: 'edge[type="member"]',
        style: {
          'line-style': 'dotted',
          'line-color': '#f87171',
          'target-arrow-shape': 'none',
          width: 1.5,
          opacity: 0.6,
          'curve-style': 'haystack',
        },
      },
      {
        // Attached relationships: thin solid teal
        selector: 'edge[type="attached"]',
        style: {
          'line-color': '#2dd4bf',
          'target-arrow-shape': 'triangle',
          'target-arrow-color': '#2dd4bf',
          width: 1.5,
          opacity: 0.7,
          'curve-style': 'haystack',
        },
      },
      {
        // Traffic/egress: thick, directional arrow, cyan-blue
        selector: 'edge[type="egress"]',
        style: {
          'line-color': '#38bdf8',
          'target-arrow-color': '#38bdf8',
          'target-arrow-shape': 'triangle',
          width: 3,
          opacity: 0.9,
        },
      },
      {
        // LB → instance targets: medium teal
        selector: 'edge[type="target"]',
        style: {
          'line-color': '#34d399',
          'target-arrow-color': '#34d399',
          'target-arrow-shape': 'triangle',
          width: 2,
          opacity: 0.8,
        },
      },
      {
        // LB → subnet attachment: thin purple
        selector: 'edge[type="lb-subnet"]',
        style: {
          'line-color': '#a78bfa',
          'target-arrow-shape': 'diamond',
          'target-arrow-color': '#a78bfa',
          width: 1.5,
          opacity: 0.65,
          'curve-style': 'haystack',
        },
      },
      // Compound (parent) node styling
      {
        selector: '$node > node',
        style: {
          'background-opacity': 0.08,
          'overlay-opacity': 0.02,
          'padding': '40px',
          'border-width': 1.5,
          'border-color': '#475569',
          'font-size': '13px',
          'font-weight': 'bold',
          'text-valign': 'top',
          'text-halign': 'center',
          'text-margin-y': 10,
          'background-color': '#1e293b',
        },
      },
    ],

    elements,
  });

  // Expose on window for debugging parity with original
  (window as unknown as Record<string, unknown>).cy = cy;

  return cy;
}
