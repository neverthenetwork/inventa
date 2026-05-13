/**
 * Tests for loading-states.ts — overlay management.
 */
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import {
  showLoading,
  hideLoading,
  showError,
  hideError,
  showEmpty,
  hideEmpty,
  clearStates,
} from '../loading-states';

function createContainer(): HTMLElement {
  const el = document.createElement('div');
  el.className = 'graph-pane';
  document.body.appendChild(el);
  return el;
}

function cleanup(container: HTMLElement): void {
  container.remove();
}

describe('showLoading / hideLoading', () => {
  let container: HTMLElement;
  beforeEach(() => {
    container = createContainer();
  });
  afterEach(() => cleanup(container));

  it('creates a loading overlay with spinner', () => {
    showLoading(container);
    const overlay = container.querySelector('.loading-overlay')!;
    expect(overlay).not.toBeNull();
    expect(overlay.classList.contains('hidden')).toBe(false);
    expect(overlay.querySelector('.spinner')).not.toBeNull();
    expect(overlay.textContent).toContain('Loading topology');
  });

  it('hides loading overlay', () => {
    showLoading(container);
    hideLoading(container);
    const overlay = container.querySelector('.loading-overlay')!;
    expect(overlay.classList.contains('hidden')).toBe(true);
  });

  it('clears other states when showing loading', () => {
    showEmpty(container, 'test');
    showLoading(container);
    expect(container.querySelector('.empty-overlay')!.classList.contains('hidden')).toBe(true);
    expect(container.querySelector('.loading-overlay')!.classList.contains('hidden')).toBe(false);
  });
});

describe('showError / hideError', () => {
  let container: HTMLElement;
  beforeEach(() => {
    container = createContainer();
  });
  afterEach(() => cleanup(container));

  it('creates an error overlay with message and retry button', () => {
    showError(container, 'Connection refused', () => {});
    const overlay = container.querySelector('.error-overlay')!;
    expect(overlay).not.toBeNull();
    expect(overlay.textContent).toContain('Connection refused');
    expect(overlay.querySelector('.btn-retry')).not.toBeNull();
  });

  it('calls onRetry when retry button clicked', () => {
    const onRetry = vi.fn();
    showError(container, 'fail', onRetry);
    const btn = container.querySelector('.btn-retry') as HTMLButtonElement;
    btn.click();
    expect(onRetry).toHaveBeenCalledTimes(1);
  });

  it('hides error overlay', () => {
    showError(container, 'fail', () => {});
    hideError(container);
    expect(container.querySelector('.error-overlay')!.classList.contains('hidden')).toBe(true);
  });

  it('escapes HTML in error message', () => {
    showError(container, '<script>alert(1)</script>', () => {});
    const msg = container.querySelector('.state-message')!;
    expect(msg.innerHTML).toContain('&lt;script&gt;');
    expect(msg.innerHTML).not.toContain('<script>');
  });
});

describe('showEmpty / hideEmpty', () => {
  let container: HTMLElement;
  beforeEach(() => {
    container = createContainer();
  });
  afterEach(() => cleanup(container));

  it('creates an empty overlay with message', () => {
    showEmpty(container, 'No data found');
    const overlay = container.querySelector('.empty-overlay')!;
    expect(overlay).not.toBeNull();
    expect(overlay.textContent).toContain('No data found');
    expect(overlay.classList.contains('hidden')).toBe(false);
  });

  it('hides empty overlay', () => {
    showEmpty(container, 'nothing');
    hideEmpty(container);
    expect(container.querySelector('.empty-overlay')!.classList.contains('hidden')).toBe(true);
  });

  it('escapes HTML in empty message', () => {
    showEmpty(container, '<img src=x onerror=alert(1)>');
    const msg = container.querySelector('.state-message')!;
    expect(msg.innerHTML).toContain('&lt;img');
    expect(msg.innerHTML).not.toContain('<img');
  });
});

describe('clearStates', () => {
  let container: HTMLElement;
  beforeEach(() => {
    container = createContainer();
  });
  afterEach(() => cleanup(container));

  it('hides all state overlays', () => {
    showLoading(container);
    showError(container, 'err', () => {});
    showEmpty(container, 'empty');

    clearStates(container);

    expect(container.querySelector('.loading-overlay')!.classList.contains('hidden')).toBe(true);
    expect(container.querySelector('.error-overlay')!.classList.contains('hidden')).toBe(true);
    expect(container.querySelector('.empty-overlay')!.classList.contains('hidden')).toBe(true);
  });
});

describe('reuse of overlay elements', () => {
  let container: HTMLElement;
  beforeEach(() => {
    container = createContainer();
  });
  afterEach(() => cleanup(container));

  it('reuses the same overlay element on subsequent calls', () => {
    showLoading(container);
    const first = container.querySelector('.loading-overlay')!;
    hideLoading(container);
    showLoading(container);
    const second = container.querySelector('.loading-overlay')!;
    expect(second).toBe(first);
  });
});
