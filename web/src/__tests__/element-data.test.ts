/**
 * Tests for element-data.ts
 */

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { fetchElementData, filterByShow, type ElementData } from '../element-data';

// Mock global fetch
const mockFetch = vi.fn();
vi.stubGlobal('fetch', mockFetch);

beforeEach(() => {
  mockFetch.mockReset();
});

afterEach(() => {
  vi.restoreAllMocks();
});

describe('fetchElementData', () => {
  it('calls /elementdata.json with no params when src and dst are empty', async () => {
    mockFetch.mockResolvedValueOnce({
      ok: true,
      json: async () => ({ nodes: [], edges: [] }),
    });

    await fetchElementData();

    expect(mockFetch).toHaveBeenCalledWith('/elementdata.json');
  });

  it('calls /elementdata.json?src=A when only src is provided', async () => {
    mockFetch.mockResolvedValueOnce({
      ok: true,
      json: async () => ({ nodes: [], edges: [] }),
    });

    await fetchElementData('A');

    expect(mockFetch).toHaveBeenCalledWith('/elementdata.json?src=A');
  });

  it('calls /elementdata.json?src=A&dst=B when both are provided', async () => {
    mockFetch.mockResolvedValueOnce({
      ok: true,
      json: async () => ({ nodes: [], edges: [] }),
    });

    await fetchElementData('A', 'B');

    expect(mockFetch).toHaveBeenCalledWith('/elementdata.json?src=A&dst=B');
  });

  it('throws on non-ok response', async () => {
    mockFetch.mockResolvedValueOnce({
      ok: false,
      status: 500,
      statusText: 'Internal Server Error',
    });

    await expect(fetchElementData()).rejects.toThrow('HTTP 500');
  });

  it('returns parsed JSON on success', async () => {
    const payload: ElementData = {
      nodes: [{ data: { id: 'n1', label: 'Node 1', show: true } }],
      edges: [{ data: { id: 'e1', source: 'n1', target: 'n2', show: true } }],
    };

    mockFetch.mockResolvedValueOnce({
      ok: true,
      json: async () => payload,
    });

    const result = await fetchElementData();
    expect(result).toEqual(payload);
  });
});

describe('filterByShow', () => {
  const sample: ElementData = {
    nodes: [
      { data: { id: 'n1', label: 'Visible', show: true } },
      { data: { id: 'n2', label: 'Hidden', show: false } },
      { data: { id: 'n3', label: 'Implicit', show: undefined } },
      { data: { id: 'n4', label: 'No show key' } },
    ],
    edges: [
      { data: { id: 'e1', source: 'n1', target: 'n2', show: true } },
      { data: { id: 'e2', source: 'n1', target: 'n3', show: false } },
      { data: { id: 'e3', source: 'n3', target: 'n4', show: undefined } },
    ],
  };

  it('keeps nodes with show=true', () => {
    const result = filterByShow(sample);
    expect(result.nodes.map((n) => n.data.id)).toContain('n1');
  });

  it('hides nodes with show=false', () => {
    const result = filterByShow(sample);
    expect(result.nodes.map((n) => n.data.id)).not.toContain('n2');
  });

  it('keeps nodes with show=undefined (default visible)', () => {
    const result = filterByShow(sample);
    expect(result.nodes.map((n) => n.data.id)).toContain('n3');
    expect(result.nodes.map((n) => n.data.id)).toContain('n4');
  });

  it('keeps edges only when show=true', () => {
    const result = filterByShow(sample);
    expect(result.edges.map((e) => e.data.id)).toEqual(['e1']);
    // e2 has show=false, e3 has show=undefined — both filtered out
    expect(result.edges.map((e) => e.data.id)).not.toContain('e2');
    expect(result.edges.map((e) => e.data.id)).not.toContain('e3');
  });

  it('returns empty nodes/edges for empty input', () => {
    const empty: ElementData = { nodes: [], edges: [] };
    expect(filterByShow(empty)).toEqual(empty);
  });
});
