/**
 * Tests for details-panel.ts — node/edge details display.
 */
import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { createDetailsPanel } from '../details-panel';
import type { NodeData, EdgeData } from '../element-data';

function createParent(): HTMLElement {
  const el = document.createElement('div');
  document.body.appendChild(el);
  return el;
}

function cleanup(parent: HTMLElement): void {
  parent.remove();
}

describe('createDetailsPanel', () => {
  let parent: HTMLElement;
  beforeEach(() => { parent = createParent(); });
  afterEach(() => cleanup(parent));

  it('creates panel hidden by default', () => {
    createDetailsPanel(parent);
    const el = document.getElementById('details_panel')!;
    expect(el).not.toBeNull();
    expect(el.classList.contains('hidden')).toBe(true);
  });

  it('shows node details when showNode is called', () => {
    const panel = createDetailsPanel(parent);
    const node: NodeData = {
      id: '10.0.0.1',
      label: 'core-router-1',
      group: 'Core',
      cluster: 1,
    };
    panel.showNode(node);
    const el = document.getElementById('details_panel')!;
    expect(el.classList.contains('hidden')).toBe(false);
    expect(el.textContent).toContain('core-router-1');
    expect(el.textContent).toContain('10.0.0.1');
    expect(el.textContent).toContain('Core');
    expect(el.textContent).toContain('1');
  });

  it('shows edge details when showEdge is called', () => {
    const panel = createDetailsPanel(parent);
    const edge: EdgeData = {
      id: 'edge-1',
      source: '10.0.0.1',
      target: '10.0.0.2',
      igp_metric: '100',
      adjacency_sid: '24001',
    };
    panel.showEdge(edge);
    const el = document.getElementById('details_panel')!;
    expect(el.classList.contains('hidden')).toBe(false);
    expect(el.textContent).toContain('10.0.0.1');
    expect(el.textContent).toContain('10.0.0.2');
    expect(el.textContent).toContain('100');
    expect(el.textContent).toContain('24001');
  });

  it('shows dash for missing optional fields', () => {
    const panel = createDetailsPanel(parent);
    const node: NodeData = { id: '1.2.3.4', label: 'leaf' };
    panel.showNode(node);
    const el = document.getElementById('details_panel')!;
    // Group and cluster are empty — expect dash placeholder
    const dds = el.querySelectorAll('dd');
    const groupDd = Array.from(dds).find((dd) => {
      const prev = dd.previousElementSibling;
      return prev?.textContent === 'Group';
    });
    expect(groupDd).toBeDefined();
    expect(groupDd!.textContent).toBe('—');
  });

  it('clears panel back to hidden', () => {
    const panel = createDetailsPanel(parent);
    panel.showNode({ id: 'x', label: 'y' });
    expect(document.getElementById('details_panel')!.classList.contains('hidden')).toBe(false);
    panel.clear();
    expect(document.getElementById('details_panel')!.classList.contains('hidden')).toBe(true);
  });

  it('escapes HTML in node label', () => {
    const panel = createDetailsPanel(parent);
    panel.showNode({ id: 'x', label: '<script>alert(1)</script>' });
    const heading = document.querySelector('.details-heading')!;
    expect(heading.innerHTML).toContain('&lt;script&gt;');
    expect(heading.innerHTML).not.toContain('<script>');
  });

  it('switching from node to edge updates content', () => {
    const panel = createDetailsPanel(parent);
    panel.showNode({ id: 'a', label: 'Router A' });
    panel.showEdge({ id: 'e1', source: 'a', target: 'b' });
    const el = document.getElementById('details_panel')!;
    expect(el.textContent).not.toContain('Router A');
    expect(el.textContent).toContain('a');
    expect(el.textContent).toContain('b');
  });
});
