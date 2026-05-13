/**
 * Tests for searchable-select.ts — combobox component.
 */
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { createComboBox } from '../searchable-select';

function createParent(): HTMLElement {
  const el = document.createElement('div');
  document.body.appendChild(el);
  return el;
}

function cleanup(parent: HTMLElement): void {
  parent.remove();
}

describe('createComboBox', () => {
  let parent: HTMLElement;
  beforeEach(() => { parent = createParent(); });
  afterEach(() => cleanup(parent));

  it('creates DOM structure with combobox role', () => {
    const cb = createComboBox(parent, 'test', 'Test');
    expect(parent.querySelector('.combobox')).not.toBeNull();
    expect(cb.input.getAttribute('role')).toBe('searchbox');
    expect(cb.list.getAttribute('role')).toBe('listbox');
    expect(cb.list.hidden).toBe(true);
  });

  it('opens list on focus when options are set', () => {
    const cb = createComboBox(parent, 'test', 'Test');
    cb.setOptions(['alpha', 'beta', 'gamma']);
    cb.input.focus();
    expect(cb.list.hidden).toBe(false);
    expect(cb.list.querySelectorAll('[role="option"]').length).toBe(3);
  });

  it('filters options on input', () => {
    const cb = createComboBox(parent, 'test', 'Test');
    cb.setOptions(['core-router-1', 'core-router-2', 'edge-switch']);
    // Open with focus first, then type filter
    cb.input.focus();
    cb.input.value = 'core';
    cb.input.dispatchEvent(new Event('input', { bubbles: true }));
    const opts = cb.list.querySelectorAll('[role="option"]:not([aria-disabled])');
    expect(opts.length).toBe(2);
    expect(opts[0].textContent).toContain('core');
    expect(opts[1].textContent).toContain('core');
  });

  it('shows no-matches when filter yields nothing', () => {
    const cb = createComboBox(parent, 'test', 'Test');
    cb.setOptions(['alpha', 'beta']);
    cb.input.focus();
    cb.input.value = 'zzz';
    cb.input.dispatchEvent(new Event('input'));
    expect(cb.list.querySelector('.combobox-no-results')).not.toBeNull();
  });

  it('selects item on click and calls onChange', () => {
    const onChange = vi.fn();
    const cb = createComboBox(parent, 'test', 'Test', { onChange });
    cb.setOptions(['alpha', 'beta']);
    cb.input.focus();
    const opt = cb.list.querySelector('[role="option"]')!;
    (opt as HTMLElement).click();
    expect(cb.getValue()).toBe('alpha');
    expect(cb.input.value).toBe('alpha');
    expect(onChange).toHaveBeenCalledWith('alpha');
    expect(cb.list.hidden).toBe(true);
  });

  it('navigates with arrow keys and selects with Enter', () => {
    const onChange = vi.fn();
    const cb = createComboBox(parent, 'test', 'Test', { onChange });
    cb.setOptions(['alpha', 'beta', 'gamma']);
    cb.input.focus();

    cb.input.dispatchEvent(new KeyboardEvent('keydown', { key: 'ArrowDown' }));
    cb.input.dispatchEvent(new KeyboardEvent('keydown', { key: 'ArrowDown' }));
    cb.input.dispatchEvent(new KeyboardEvent('keydown', { key: 'Enter' }));

    expect(cb.getValue()).toBe('beta');
    expect(onChange).toHaveBeenCalledWith('beta');
  });

  it('closes list on Escape and restores value', () => {
    const cb = createComboBox(parent, 'test', 'Test');
    cb.setOptions(['alpha', 'beta']);
    cb.setValue('alpha');
    cb.input.focus();
    cb.input.value = 'xyz';
    cb.input.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }));
    expect(cb.list.hidden).toBe(true);
    expect(cb.input.value).toBe('alpha');
  });

  it('setValue updates current value and fires onChange', () => {
    const onChange = vi.fn();
    const cb = createComboBox(parent, 'test', 'Test', { onChange });
    cb.setValue('example');
    expect(cb.getValue()).toBe('example');
    expect(onChange).toHaveBeenCalledWith('example');
  });

  it('clears value when selected option is removed from options', () => {
    const onChange = vi.fn();
    const cb = createComboBox(parent, 'test', 'Test', { onChange });
    cb.setOptions(['alpha', 'beta']);
    cb.setValue('beta');
    expect(cb.getValue()).toBe('beta');
    cb.setOptions(['alpha']); // beta removed
    expect(cb.getValue()).toBe('');
    expect(onChange).toHaveBeenCalledWith('');
  });

  it('preserves selection when still in options', () => {
    const cb = createComboBox(parent, 'test', 'Test');
    cb.setOptions(['alpha', 'beta']);
    cb.setValue('beta');
    cb.setOptions(['alpha', 'beta', 'gamma']);
    expect(cb.getValue()).toBe('beta');
  });

  it('announces selection to screen reader', () => {
    const cb = createComboBox(parent, 'test', 'Source Router');
    cb.setOptions(['alpha']);
    cb.input.focus();
    const opt = cb.list.querySelector('[role="option"]')!;
    (opt as HTMLElement).click();
    // Live region is .sr-only, check textContent
    expect(cb.liveRegion.textContent).toContain('Source Router');
    expect(cb.liveRegion.textContent).toContain('alpha');
  });

  it('handles empty options gracefully', () => {
    const cb = createComboBox(parent, 'test', 'Test');
    cb.input.focus();
    expect(cb.list.hidden).toBe(true);
  });
});
