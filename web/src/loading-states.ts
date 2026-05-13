/**
 * UI state management for loading, error, and empty states.
 * Manages overlay visibility in the graph pane without touching cytoscape.
 */

// CSS class names (keep in sync with style.css)
const LOADING_CLS = 'loading-overlay';
const ERROR_CLS = 'error-overlay';
const EMPTY_CLS = 'empty-overlay';
const HIDDEN_CLS = 'hidden';
const GRAPH_PANE_CLS = 'graph-pane';

type RetryFn = () => void;

/**
 * Show a loading spinner overlay.
 */
export function showLoading(container: HTMLElement): void {
  clearStates(container);
  const el = findOrCreate(container, LOADING_CLS, 'spinner-container');
  el.innerHTML = `<div class="spinner"></div><p class="state-message">Loading topology…</p>`;
  el.classList.remove(HIDDEN_CLS);
}

/**
 * Hide the loading overlay.
 */
export function hideLoading(container: HTMLElement): void {
  const el = container.querySelector(`.${LOADING_CLS}`);
  if (el) el.classList.add(HIDDEN_CLS);
}

/**
 * Show an error overlay with retry button.
 */
export function showError(
  container: HTMLElement,
  message: string,
  onRetry: RetryFn,
): void {
  clearStates(container);
  const el = findOrCreate(container, ERROR_CLS, 'error-container');
  el.innerHTML = `
    <div class="state-icon">⚠</div>
    <p class="state-message">${escapeHtml(message)}</p>
    <button class="btn btn-retry" type="button">Retry</button>
  `;
  el.classList.remove(HIDDEN_CLS);

  // Wire up retry button (innerHTML reassigns DOM, so old listeners are destroyed — no leak)
  const btn = el.querySelector('.btn-retry');
  if (btn) {
    btn.addEventListener('click', () => {
      hideError(container);
      onRetry();
    });
  }
}

/**
 * Hide the error overlay.
 */
export function hideError(container: HTMLElement): void {
  const el = container.querySelector(`.${ERROR_CLS}`);
  if (el) el.classList.add(HIDDEN_CLS);
}

/**
 * Show an empty-state overlay (no data, no path, etc.).
 */
export function showEmpty(container: HTMLElement, message: string): void {
  clearStates(container);
  const el = findOrCreate(container, EMPTY_CLS, 'empty-container');
  el.innerHTML = `
    <div class="state-icon">📭</div>
    <p class="state-message">${escapeHtml(message)}</p>
  `;
  el.classList.remove(HIDDEN_CLS);
}

/**
 * Hide the empty-state overlay.
 */
export function hideEmpty(container: HTMLElement): void {
  const el = container.querySelector(`.${EMPTY_CLS}`);
  if (el) el.classList.add(HIDDEN_CLS);
}

/**
 * Clear all state overlays.
 */
export function clearStates(container: HTMLElement): void {
  hideLoading(container);
  hideError(container);
  hideEmpty(container);
}

// -- helpers --

const cache = new WeakMap<HTMLElement, Map<string, HTMLElement>>();

function findOrCreate(
  parent: HTMLElement,
  cls: string,
  testId: string,
): HTMLElement {
  // Walk up to graph-pane
  const pane =
    parent.classList.contains(GRAPH_PANE_CLS)
      ? parent
      : (parent.querySelector(`.${GRAPH_PANE_CLS}`) as HTMLElement | null);

  const root = pane || parent;

  // Check cache
  let map = cache.get(root);
  if (!map) {
    map = new Map();
    cache.set(root, map);
  }
  if (map.has(testId)) return map.get(testId)!;

  // Create overlay
  const el = document.createElement('div');
  el.className = `state-overlay ${cls} ${HIDDEN_CLS}`;
  el.setAttribute('data-testid', testId);
  root.appendChild(el);
  map.set(testId, el);
  return el;
}

function escapeHtml(str: string): string {
  const map: Record<string, string> = {
    '&': '&amp;',
    '<': '&lt;',
    '>': '&gt;',
    '"': '&quot;',
    "'": '&#39;',
  };
  // The regex only matches mapped characters, so map[ch] is always defined;
  // the '|| ch' fallback is a type-safety guard for the TypeScript compiler.
  return str.replace(/[&<>"']/g, (ch) => map[ch] || ch);
}
