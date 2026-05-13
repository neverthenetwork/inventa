/**
 * Entry point for the 3D force-graph view (3dindex.html).
 */

import ForceGraph3D from '3d-force-graph';
import { fetchGraphData, configureGraph } from './3d-graph';

async function render3D(): Promise<void> {
  const container = document.getElementById('3d-graph');
  if (!container) {
    console.error('Container #3d-graph not found');
    return;
  }

  try {
    const graphData = await fetchGraphData();

    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const Graph = (ForceGraph3D as any)()(container).graphData(graphData);

    configureGraph(Graph).nodeLabel('label');
  } catch (err) {
    console.error('Failed to render 3D graph:', err);
  }
}

window.addEventListener('DOMContentLoaded', render3D);
