/**
 * Entry point for the 2D cytoscape view (index.html).
 */

import './style.css';
import { fetchElementData, filterByShow } from './element-data';
import { createGraph } from './graph-renderer';
import {
  showLoading,
  hideLoading,
  showError,
  showEmpty,
  hideEmpty,
} from './loading-states';
import { createComboBox } from './searchable-select';
import { createDetailsPanel } from './details-panel';

let cy: ReturnType<typeof createGraph> | null = null;
let srcBox: ReturnType<typeof createComboBox> | null = null;
let dstBox: ReturnType<typeof createComboBox> | null = null;
let detailsPanel: ReturnType<typeof createDetailsPanel> | null = null;

function populateComboboxes(labels: string[]): void {
  if (srcBox) srcBox.setOptions(labels);
  if (dstBox) dstBox.setOptions(labels);
}

function highlightPathStyle(): void {
  if (!cy) return;
  // Distinct styling for path edges/nodes when src+dst are set
  cy.style()
    .selector('edge.path-highlight')
    .style({
      'line-color': '#2dd4bf',
      'target-arrow-color': '#2dd4bf',
      width: 3,
      opacity: 1,
    })
    .selector('node.path-highlight')
    .style({
      'border-color': '#2dd4bf',
      'border-width': 2,
    })
    .update();
}

async function renderGraph(): Promise<void> {
  const container = document.getElementById('cy');
  if (!container) {
    console.error('Container #cy not found');
    return;
  }

  const graphPane = container.closest('.graph-pane') as HTMLElement | null;
  const statusText = document.getElementById('status_text');
  const nodeCount = document.getElementById('node_count');
  const updateBtn = document.getElementById('update_btn') as HTMLButtonElement | null;

  const src = srcBox?.getValue() ?? '';
  const dst = dstBox?.getValue() ?? '';

  // Disable button during fetch
  if (updateBtn) updateBtn.disabled = true;
  if (detailsPanel) detailsPanel.clear();

  // Show loading
  if (graphPane) showLoading(graphPane);

  try {
    const data = await fetchElementData(src, dst);

    // Populate comboboxes with unique sorted labels
    const labels = data.nodes
      .map((n) => n.data.label)
      .filter((l): l is string => typeof l === 'string')
      .sort();
    populateComboboxes(labels);

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
    highlightPathStyle();

    // --- Cytoscape tap events for details panel ---
    cy.on('tap', 'node', (evt) => {
      const node = evt.target;
      detailsPanel?.showNode(node.data() as import('./element-data').NodeData);
    });

    cy.on('tap', 'edge', (evt) => {
      const edge = evt.target;
      detailsPanel?.showEdge(edge.data() as import('./element-data').EdgeData);
    });

    cy.on('tap', (evt) => {
      if (evt.target === cy) {
        detailsPanel?.clear();
      }
    });

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
  // Create searchable comboboxes
  {
    const srcContainer = document.getElementById('src_container');
    if (srcContainer) {
      srcBox = createComboBox(srcContainer, 'src_select', 'Source', {
        placeholder: 'Search source…',
        onChange: () => renderGraph(),
      });
    }
    const dstContainer = document.getElementById('dst_container');
    if (dstContainer) {
      dstBox = createComboBox(dstContainer, 'dst_select', 'Destination', {
        placeholder: 'Search destination…',
        onChange: () => renderGraph(),
      });
    }
  }

  // Create details panel
  {
    const detailsArea = document.getElementById('details_area');
    if (detailsArea) {
      detailsPanel = createDetailsPanel(detailsArea);
    }
  }

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
