/**
 * Entry point for the 2D cytoscape view (index.html).
 */

import './style.css';
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
  const statusText = document.getElementById('status_text');
  const nodeCount = document.getElementById('node_count');

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

    // Update status bar
    if (nodeCount) {
      nodeCount.textContent = `${filtered.nodes.length} nodes · ${filtered.edges.length} edges`;
    }
    if (statusText) {
      statusText.textContent = src && dst
        ? `Path: ${src} → ${dst}`
        : 'All topology';
    }
  } catch (err) {
    console.error('Failed to render graph:', err);
    if (statusText) {
      statusText.textContent = '⚠ Failed to load topology';
    }
  }
}

// --- Bootstrap ---
window.addEventListener('DOMContentLoaded', () => {
  // Initial render
  renderGraph();

  // Wire up the Update button
  const updateBtn = document.getElementById('update_btn');

  if (updateBtn) {
    updateBtn.addEventListener('click', () => {
      renderGraph();
    });
  }

  // Mobile menu toggle
  const menuToggle = document.getElementById('menu_toggle');
  const sidebar = document.getElementById('sidebar');

  if (menuToggle && sidebar) {
    menuToggle.addEventListener('click', () => {
      const open = sidebar.classList.toggle('open');
      menuToggle.setAttribute('aria-expanded', String(open));
    });
  }
});
