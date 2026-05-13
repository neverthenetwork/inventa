/**
 * Entry point for the 2D cytoscape view (index.html).
 */

import './style.css';
import { fetchElementData, filterByShow } from './element-data';
import { createGraph } from './graph-renderer';
import { updateDropdowns } from './path-filter';
import {
  showLoading,
  hideLoading,
  showError,
  showEmpty,
  hideEmpty,
} from './loading-states';

let cy: ReturnType<typeof createGraph> | null = null;

async function renderGraph(): Promise<void> {
  const container = document.getElementById('cy');
  if (!container) {
    console.error('Container #cy not found');
    return;
  }

  const graphPane = container.closest('.graph-pane') as HTMLElement | null;
  const srcSelect = document.getElementById('src_select') as HTMLSelectElement | null;
  const dstSelect = document.getElementById('dst_select') as HTMLSelectElement | null;
  const statusText = document.getElementById('status_text');
  const nodeCount = document.getElementById('node_count');
  const updateBtn = document.getElementById('update_btn') as HTMLButtonElement | null;

  const src = srcSelect?.value ?? '';
  const dst = dstSelect?.value ?? '';

  // Disable button during fetch
  if (updateBtn) updateBtn.disabled = true;

  // Show loading
  if (graphPane) showLoading(graphPane);

  try {
    const data = await fetchElementData(src, dst);

    // Populate dropdowns (preserves current selections)
    if (srcSelect && dstSelect) {
      updateDropdowns(data.nodes, srcSelect, dstSelect);
    }

    // Filter by show attribute
    const filtered = filterByShow(data);

    // Handle empty states
    if (filtered.nodes.length === 0) {
      if (cy) { cy.destroy(); cy = null; }
      if (graphPane) {
        const msg = (src || dst)
          ? `No path found${src ? ` from ${src}` : ''}${dst ? ` to ${dst}` : ''}`
          : 'No topology data loaded';
        showEmpty(graphPane, msg);
      }
      if (statusText) statusText.textContent = '';
      if (nodeCount) nodeCount.textContent = '0 nodes · 0 edges';
      return;
    }

    // Data arrived — hide empty state if shown
    if (graphPane) hideEmpty(graphPane);

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
    if (graphPane) {
      showError(
        graphPane,
        err instanceof Error ? err.message : 'Failed to load topology data',
        () => renderGraph(),
      );
    }
    if (statusText) {
      statusText.textContent = '⚠ Failed to load topology';
    }
  } finally {
    if (graphPane) hideLoading(graphPane);
    if (updateBtn) updateBtn.disabled = false;
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
