/**
 * Details panel — shows node or edge information when tapped in cytoscape.
 * Appended to the sidebar; hidden when no element is selected.
 */

import type { NodeData, EdgeData } from './element-data';

export interface DetailsPanel {
  showNode: (data: NodeData) => void;
  showEdge: (data: EdgeData) => void;
  clear: () => void;
}

export function createDetailsPanel(parent: HTMLElement): DetailsPanel {
  const panel = document.createElement('div');
  panel.className = 'details-panel hidden';
  panel.id = 'details_panel';

  // Heading
  const heading = document.createElement('h2');
  heading.className = 'details-heading';
  panel.append(heading);

  // Body (key-value list)
  const body = document.createElement('dl');
  body.className = 'details-body';
  panel.append(body);

  parent.append(panel);

  function show(icon: string, title: string, fields: [string, string][]): void {
    heading.innerHTML = `${icon} ${escapeHtml(title)}`;
    body.innerHTML = '';
    fields.forEach(([key, val]) => {
      const dt = document.createElement('dt');
      dt.textContent = key;
      const dd = document.createElement('dd');
      dd.textContent = val || '—';
      body.append(dt, dd);
    });
    panel.classList.remove('hidden');
  }

  return {
    showNode(data: NodeData): void {
      show('🖥', data.label || data.id, [
        ['Router ID', data.id],
        ['Group', data.group || ''],
        ['Cluster', data.cluster != null ? String(data.cluster) : ''],
      ]);
    },

    showEdge(data: EdgeData): void {
      show('🔗', `${data.source} → ${data.target}`, [
        ['Source', data.source],
        ['Target', data.target],
        ['IGP Metric', data.igp_metric || ''],
        ['Adjacency SID', data.adjacency_sid || ''],
      ]);
    },

    clear(): void {
      panel.classList.add('hidden');
    },
  };
}

function escapeHtml(str: string): string {
  const map: Record<string, string> = {
    '&': '&amp;',
    '<': '&lt;',
    '>': '&gt;',
    '"': '&quot;',
    "'": '&#39;',
  };
  return str.replace(/[&<>"']/g, (ch) => map[ch] || ch);
}
