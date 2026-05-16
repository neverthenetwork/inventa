/**
 * Details panel — shows node or edge information when tapped in cytoscape.
 * Appended to the sidebar; hidden when no element is selected.
 * Context-aware — different fields shown based on node group / edge type.
 */

import type { NodeData, EdgeData } from './element-data';

export interface DetailsPanel {
  showNode: (data: NodeData) => void;
  showEdge: (data: EdgeData) => void;
  clear: () => void;
}

/** Look up the group-specific icon and title for a node. */
function nodeHeading(data: NodeData): { icon: string; title: string } {
  switch (data.group) {
    case 'vpc':
      return { icon: '☁️', title: `VPC: ${data.label || data.id}` };
    case 'subnet':
      return { icon: '📡', title: `Subnet: ${data.label || data.id}` };
    case 'instance':
      return { icon: '🖥️', title: `Instance: ${data.label || data.id}` };
    case 'security_group':
      return { icon: '🔒', title: `Security Group: ${data.label || data.id}` };
    case 'elb':
      return { icon: '⚖️', title: `Load Balancer: ${data.label || data.id}` };
    case 'igw':
      return { icon: '🌐', title: `Internet Gateway: ${data.label || data.id}` };
    case 'internet':
      return { icon: '🌍', title: 'Internet' };
    default:
      return { icon: '🖥', title: data.label || data.id };
  }
}

/** Build the key-value field list for a node based on its group. */
function nodeFields(data: NodeData): [string, string][] {
  const fields: [string, string][] = [];
  const str = (v: unknown): string => (v != null ? String(v) : '—');

  fields.push(['ID', data.id]);

  switch (data.group) {
    case 'vpc':
      fields.push(['CIDR', str(data.cidr)]);
      fields.push(['Default', data.isDefault ? 'Yes' : 'No']);
      break;
    case 'subnet':
      fields.push(['CIDR', str(data.cidr)]);
      fields.push(['AZ', str(data.az)]);
      fields.push(['Public IPs', data.public ? 'Yes' : 'No']);
      fields.push(['VPC', str(data.vpcId)]);
      break;
    case 'instance':
      fields.push(['Type', str(data.instanceType)]);
      fields.push(['Private IP', str(data.privateIp)]);
      fields.push(['Public IP', str(data.publicIp)]);
      fields.push(['State', str(data.state)]);
      fields.push(['Subnet', str(data.subnetId)]);
      fields.push(['VPC', str(data.vpcId)]);
      break;
    case 'security_group':
      fields.push(['Name', str(data.groupName)]);
      fields.push(['Description', str(data.description)]);
      fields.push(['Ingress Rules', str(data.ingressRules)]);
      fields.push(['Egress Rules', str(data.egressRules)]);
      fields.push(['VPC', str(data.vpcId)]);
      break;
    case 'elb':
      fields.push(['DNS', str(data.dns)]);
      fields.push(['Type', str(data.type)]);
      fields.push(['Scheme', str(data.scheme)]);
      fields.push(['VPC', str(data.vpcId)]);
      break;
    case 'igw':
      fields.push(['VPC', str(data.vpcId)]);
      break;
    case 'internet':
      // No extra fields — synthetic node
      break;
    default:
      // BGP-LS / generic node
      fields.push(['Group', str(data.group)]);
      fields.push(['Cluster', str(data.cluster)]);
      break;
  }

  return fields;
}

/** Build the key-value field list for an edge. */
function edgeFields(data: EdgeData): [string, string][] {
  const str = (v: unknown): string => (v != null ? String(v) : '—');
  const fields: [string, string][] = [
    ['Source', data.source],
    ['Target', data.target],
    ['Type', str(data.type)],
  ];

  if (data.igp_metric) {
    fields.push(['IGP Metric', str(data.igp_metric)]);
  }
  if (data.adjacency_sid) {
    fields.push(['Adjacency SID', str(data.adjacency_sid)]);
  }

  // Type-specific fields
  if (data.type === 'member') {
    fields.push(['SG Name', str(data.groupName)]);
  }

  return fields;
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
      const { icon, title } = nodeHeading(data);
      show(icon, title, nodeFields(data));
    },

    showEdge(data: EdgeData): void {
      // Choose icon based on edge type
      let icon = '🔗';
      if (data.type === 'parent') icon = '📦';
      else if (data.type === 'member') icon = '🔒';
      else if (data.type === 'egress') icon = '⬆️';
      else if (data.type === 'target') icon = '🎯';
      else if (data.type === 'attached') icon = '📌';
      else if (data.type === 'lb-subnet') icon = '🔀';

      show(icon, `${data.source} → ${data.target}`, edgeFields(data));
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
