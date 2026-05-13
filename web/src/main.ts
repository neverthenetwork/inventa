/**
 * Entry point for the 2D cytoscape view (index.html).
 */

import { fetchElementData, filterByShow } from './element-data';
import { createGraph } from './graph-renderer';
import { updateDropdowns } from './path-filter';

let cy: ReturnType<typeof createGraph> | null = null;

async function renderGraph(): Promise<void> {
  const container = document.getElementById('cy');
  if (!container) {
    console.error('Container #cy not found');
    return;
  }

  const srcSelect = document.getElementById('src_select') as HTMLSelectElement | null;
  const dstSelect = document.getElementById('dst_select') as HTMLSelectElement | null;

  const src = srcSelect?.value ?? '';
  const dst = dstSelect?.value ?? '';

  try {
    const data = await fetchElementData(src, dst);

    // Populate dropdowns (preserves current selections)
    if (srcSelect && dstSelect) {
      updateDropdowns(data.nodes, srcSelect, dstSelect);
    }

    // Filter by show attribute
    const filtered = filterByShow(data);

    // Destroy previous graph instance if any
    if (cy) {
      cy.destroy();
    }

    cy = createGraph(container, filtered);
  } catch (err) {
    console.error('Failed to render graph:', err);
  }
}

// --- Bootstrap ---
window.addEventListener('DOMContentLoaded', () => {
  // Initial render
  renderGraph();

  // Wire up the Update button
  const updateBtn = document.getElementById('update_btn');
  const patternsTa = document.getElementById(
    'include_patterns',
  ) as HTMLTextAreaElement | null;

  if (updateBtn && patternsTa) {
    updateBtn.addEventListener('click', () => {
      renderGraph();
    });
  }
});
