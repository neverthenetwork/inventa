/**
 * Legend panel — shows the key for node groups (colours, shapes) and edge types
 * (line styles, colours). Appended to the sidebar.
 */

/** Node group entries for the legend. */
interface NodeLegendItem {
  group: string;
  label: string;
  colour: string;
  shape: string;
}

/** Edge type entries for the legend. */
interface EdgeLegendItem {
  type: string;
  label: string;
  colour: string;
  style: string;
}

const NODE_ITEMS: NodeLegendItem[] = [
  { group: 'vpc', label: 'VPC', colour: '#4499a3', shape: 'round-rectangle' },
  { group: 'subnet', label: 'Subnet', colour: '#5a8a4a', shape: 'rectangle' },
  { group: 'instance', label: 'EC2 Instance', colour: '#c2855b', shape: 'ellipse' },
  { group: 'elb', label: 'Load Balancer', colour: '#a15b9e', shape: 'diamond' },
  { group: 'igw', label: 'Internet Gateway', colour: '#5b8aa1', shape: 'triangle' },
  { group: 'security_group', label: 'Security Group', colour: '#c25b5b', shape: 'hexagon' },
  { group: 'internet', label: 'Internet', colour: '#3498db', shape: 'ellipse-dashed' },
];

const EDGE_ITEMS: EdgeLegendItem[] = [
  { type: 'parent', label: 'Contains', colour: '#64748b', style: 'dashed' },
  { type: 'member', label: 'SG Member', colour: '#f87171', style: 'dotted' },
  { type: 'attached', label: 'Attached', colour: '#2dd4bf', style: 'solid' },
  { type: 'target', label: 'LB Target', colour: '#34d399', style: 'solid-arrow' },
  { type: 'egress', label: 'Egress', colour: '#38bdf8', style: 'solid-arrow' },
  { type: 'lb-subnet', label: 'LB → Subnet', colour: '#a78bfa', style: 'solid' },
];

/** Render a small inline swatch for a colour + shape. */
function swatchHTML(colour: string, shape: string): string {
  let borderRadius = '2px';
  const width = 14;
  const height = 14;

  // Map shape names to CSS approximations
  switch (shape) {
    case 'round-rectangle':
      borderRadius = '4px';
      break;
    case 'ellipse':
    case 'ellipse-dashed':
      borderRadius = '50%';
      break;
    case 'diamond':
      return `<span style="display:inline-block;width:0;height:0;border:7px solid transparent;border-bottom:9px solid ${colour};position:relative;top:-6px;margin-right:1px" aria-hidden="true"></span>`;
    case 'triangle':
      return `<span style="display:inline-block;width:0;height:0;border-left:7px solid transparent;border-right:7px solid transparent;border-bottom:10px solid ${colour};position:relative;top:-2px;margin-right:1px" aria-hidden="true"></span>`;
    case 'hexagon':
      // Approximate with a clipped square — close enough for legend
      borderRadius = '3px';
      break;
    default:
      break;
  }

  const borderStyle = shape === 'ellipse-dashed' ? 'border:2px dashed ' + colour : '';
  return `<span style="display:inline-block;width:${width}px;height:${height}px;background:${colour};border-radius:${borderRadius};${borderStyle};margin-right:1px" aria-hidden="true"></span>`;
}

/** Render an edge style indicator. */
function edgeIndicatorHTML(colour: string, style: string): string {
  const lineStyle = style === 'dashed' ? '5,3' : style === 'dotted' ? '2,3' : 'none';
  const dashArray = lineStyle !== 'none' ? `stroke-dasharray="${lineStyle}"` : '';

  if (style === 'solid-arrow') {
    // Thick line with arrowhead hint
    return `<svg width="24" height="12" style="vertical-align:middle;margin-right:3px" aria-hidden="true">
      <line x1="2" y1="6" x2="18" y2="6" stroke="${colour}" stroke-width="2.5" />
      <polygon points="18,6 14,3 14,9" fill="${colour}" />
    </svg>`;
  }

  return `<svg width="24" height="12" style="vertical-align:middle;margin-right:3px" aria-hidden="true">
    <line x1="2" y1="6" x2="22" y2="6" stroke="${colour}" stroke-width="1.5" ${dashArray} />
  </svg>`;
}

export function createLegend(parent: HTMLElement): void {
  const wrapper = document.createElement('details');
  wrapper.className = 'legend';
  wrapper.open = true;

  const summary = document.createElement('summary');
  summary.textContent = '🔑 Legend';
  wrapper.append(summary);

  const body = document.createElement('div');
  body.className = 'legend-body';

  // Node groups
  const nodeTitle = document.createElement('h4');
  nodeTitle.textContent = 'Nodes';
  body.append(nodeTitle);

  const nodeList = document.createElement('ul');
  nodeList.className = 'legend-list';
  NODE_ITEMS.forEach((item) => {
    const li = document.createElement('li');
    li.innerHTML = `${swatchHTML(item.colour, item.shape)} ${item.label}`;
    nodeList.append(li);
  });
  body.append(nodeList);

  // Edge types
  const edgeTitle = document.createElement('h4');
  edgeTitle.textContent = 'Edges';
  body.append(edgeTitle);

  const edgeList = document.createElement('ul');
  edgeList.className = 'legend-list';
  EDGE_ITEMS.forEach((item) => {
    const li = document.createElement('li');
    li.innerHTML = `${edgeIndicatorHTML(item.colour, item.style)} ${item.label}`;
    edgeList.append(li);
  });
  body.append(edgeList);

  wrapper.append(body);
  parent.append(wrapper);
}
