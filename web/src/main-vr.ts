/**
 * Entry point for the VR force-graph view (vrindex.html).
 */

import './style.css';
import 'aframe'; // sets window.AFRAME — required by 3d-force-graph-vr
import ForceGraphVR from '3d-force-graph-vr';
import { fetchGraphData, configureGraph } from './3d-graph';

async function renderVR(): Promise<void> {
  const container = document.getElementById('3d-graph');
  if (!container) {
    console.error('Container #3d-graph not found');
    return;
  }

  try {
    const graphData = await fetchGraphData();

    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const Graph = (ForceGraphVR as any)()(container).graphData(graphData);

    configureGraph(Graph).nodeLabel('label');
  } catch (err) {
    console.error('Failed to render VR graph:', err);
  }
}

window.addEventListener('DOMContentLoaded', renderVR);
