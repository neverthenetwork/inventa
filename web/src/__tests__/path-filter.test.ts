/**
 * Tests for path-filter.ts
 */

import { describe, it, expect, beforeEach } from 'vitest';
import { updateDropdowns, matchesPatterns } from '../path-filter';
import type { Node } from '../element-data';

function makeSelect(): HTMLSelectElement {
  return document.createElement('select');
}

function makeNodes(labels: string[]): Node[] {
  return labels.map((label, i) => ({
    data: { id: `n${i}`, label },
  }));
}

describe('updateDropdowns', () => {
  let src: HTMLSelectElement;
  let dst: HTMLSelectElement;

  beforeEach(() => {
    src = makeSelect();
    dst = makeSelect();
  });

  it('populates both selects with sorted labels + "None"', () => {
    const nodes = makeNodes(['Z', 'A', 'M']);
    updateDropdowns(nodes, src, dst);

    // Each select should have 4 options: None, A, M, Z
    expect(src.options.length).toBe(4);
    expect(dst.options.length).toBe(4);

    expect(src.options[0].text).toBe('None');
    expect(src.options[0].value).toBe('');

    expect(src.options[1].text).toBe('A');
    expect(src.options[2].text).toBe('M');
    expect(src.options[3].text).toBe('Z');
  });

  it('preserves existing selections when values still exist', () => {
    const nodes = makeNodes(['A', 'B']);
    // Pre-select "B" on src
    const opt = document.createElement('option');
    opt.text = 'B';
    opt.value = 'B';
    src.add(opt);
    src.value = 'B';

    updateDropdowns(nodes, src, dst);

    expect(src.value).toBe('B');
    expect(dst.value).toBe('');
  });

  it('resets selections when previous value no longer exists', () => {
    const nodes = makeNodes(['A', 'B']);
    // Pre-select something that won't be in the new list
    const opt = document.createElement('option');
    opt.text = 'Old';
    opt.value = 'Old';
    src.add(opt);
    src.value = 'Old';

    updateDropdowns(nodes, src, dst);

    // The "None" option is selected by default (value '')
    expect(src.value).toBe('');
  });

  it('handles empty node list', () => {
    updateDropdowns([], src, dst);

    expect(src.options.length).toBe(1); // just "None"
    expect(dst.options.length).toBe(1);
    expect(src.options[0].text).toBe('None');
  });
});

describe('matchesPatterns', () => {
  it('returns true when patterns list is empty', () => {
    expect(matchesPatterns('anything', [])).toBe(true);
  });

  it('returns true when name contains a pattern substring', () => {
    expect(matchesPatterns('hello-world', ['hello'])).toBe(true);
  });

  it('returns false when name does not contain any pattern', () => {
    expect(matchesPatterns('hello-world', ['xyz', 'abc'])).toBe(false);
  });

  it('ignores empty string patterns', () => {
    // Empty strings in patterns should be ignored
    expect(matchesPatterns('hello', [''])).toBe(false);
  });

  it('matches when at least one pattern matches', () => {
    expect(matchesPatterns('foobar', ['baz', 'foo', 'qux'])).toBe(true);
  });

  it('is case sensitive', () => {
    expect(matchesPatterns('Hello', ['hello'])).toBe(false);
  });
});
