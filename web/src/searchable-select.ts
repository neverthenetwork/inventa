/**
 * Searchable combobox — replaces native <select> with a text input + filtered dropdown.
 * Supports keyboard navigation, ARIA attributes, and screen-reader announcements.
 */

export interface ComboBoxOptions {
  /** Placeholder text when no item is selected */
  placeholder?: string;
  /** Called when the selection changes (empty string = none selected) */
  onChange?: (value: string) => void;
}

interface ComboBoxInstance {
  container: HTMLElement;
  input: HTMLInputElement;
  list: HTMLUListElement;
  liveRegion: HTMLElement;
  /** Set the available options (sorted). Existing selection is preserved. */
  setOptions: (items: string[]) => void;
  /** Get the current value */
  getValue: () => string;
  /** Programmatically set the value */
  setValue: (val: string) => void;
}

export function createComboBox(
  parent: HTMLElement,
  id: string,
  label: string,
  options: ComboBoxOptions = {},
): ComboBoxInstance {
  const placeholder = options.placeholder || 'Search…';

  // Build DOM
  const container = document.createElement('div');
  container.className = 'combobox';
  container.setAttribute('role', 'combobox');
  container.setAttribute('aria-haspopup', 'listbox');
  container.setAttribute('aria-expanded', 'false');

  const input = document.createElement('input');
  input.type = 'text';
  input.id = id;
  input.placeholder = placeholder;
  input.autocomplete = 'off';
  input.setAttribute('role', 'searchbox');
  input.setAttribute('aria-autocomplete', 'list');
  input.setAttribute('aria-controls', `${id}-list`);
  input.setAttribute('aria-activedescendant', '');

  const list = document.createElement('ul');
  list.id = `${id}-list`;
  list.className = 'combobox-list';
  list.setAttribute('role', 'listbox');
  list.hidden = true;

  // Live region for screen readers
  const liveRegion = document.createElement('span');
  liveRegion.className = 'sr-only';
  liveRegion.setAttribute('aria-live', 'polite');
  liveRegion.setAttribute('aria-atomic', 'true');

  container.append(input, list, liveRegion);
  parent.append(container);

  // State
  let allItems: string[] = [];
  let currentValue = '';
  let activeIdx = -1;

  // --- helpers ---

  function announce(text: string): void {
    liveRegion.textContent = text;
  }

  function setActive(idx: number): void {
    const prev = list.querySelector('[aria-selected="true"]');
    if (prev) prev.setAttribute('aria-selected', 'false');

    const items = list.querySelectorAll<HTMLElement>('[role="option"]');
    if (idx >= 0 && idx < items.length) {
      items[idx].setAttribute('aria-selected', 'true');
      // scrollIntoView is not available in jsdom test environment
      if (typeof (items[idx] as HTMLElement).scrollIntoView === 'function') {
        (items[idx] as HTMLElement).scrollIntoView({ block: 'nearest' });
      }
      input.setAttribute('aria-activedescendant', items[idx].id);
      activeIdx = idx;
    } else {
      input.setAttribute('aria-activedescendant', '');
      activeIdx = -1;
    }
  }

  function selectItem(value: string): void {
    currentValue = value;
    input.value = value || '';
    closeList();
    announce(value ? `${label}: ${value}` : `${label}: none`);
    options.onChange?.(value);
  }

  function renderList(filter: string): void {
    list.innerHTML = '';
    const lower = filter.toLowerCase();
    const filtered = filter
      ? allItems.filter((it) => it.toLowerCase().includes(lower))
      : allItems;

    if (filtered.length === 0) {
      const li = document.createElement('li');
      li.className = 'combobox-no-results';
      li.textContent = 'No matches';
      li.setAttribute('role', 'option');
      li.setAttribute('aria-disabled', 'true');
      list.append(li);
      activeIdx = -1;
    } else {
      filtered.forEach((item, i) => {
        const li = document.createElement('li');
        li.id = `${id}-option-${i}`;
        li.className = 'combobox-option';
        li.textContent = item;
        li.setAttribute('role', 'option');
        li.addEventListener('click', () => selectItem(item));
        list.append(li);
      });

      // Preserve active index if still in range
      if (activeIdx >= filtered.length) activeIdx = -1;
      if (activeIdx >= 0) {
        setActive(activeIdx);
      }
    }
  }

  function openList(): void {
    list.hidden = false;
    container.setAttribute('aria-expanded', 'true');
    renderList(input.value);
  }

  function closeList(): void {
    list.hidden = true;
    container.setAttribute('aria-expanded', 'false');
    activeIdx = -1;
    input.setAttribute('aria-activedescendant', '');
  }

  // --- events ---

  // Click outside closes
  document.addEventListener('click', (e) => {
    if (!container.contains(e.target as Node)) closeList();
  });

  input.addEventListener('focus', () => {
    if (allItems.length > 0) openList();
  });

  input.addEventListener('input', () => {
    if (!list.hidden) {
      renderList(input.value);
    } else if (allItems.length > 0) {
      openList();
    }
  });

  input.addEventListener('keydown', (e) => {
    if (list.hidden && (e.key === 'ArrowDown' || e.key === 'ArrowUp')) {
      if (allItems.length > 0) openList();
      e.preventDefault();
      return;
    }

    const items = list.querySelectorAll<HTMLElement>('[role="option"]:not([aria-disabled])');

    switch (e.key) {
      case 'ArrowDown':
        e.preventDefault();
        setActive(Math.min(activeIdx + 1, items.length - 1));
        break;
      case 'ArrowUp':
        e.preventDefault();
        setActive(Math.max(activeIdx - 1, 0));
        break;
      case 'Enter':
        e.preventDefault();
        if (activeIdx >= 0 && activeIdx < items.length) {
          selectItem(items[activeIdx].textContent || '');
        }
        break;
      case 'Escape':
        closeList();
        input.value = currentValue; // revert
        break;
    }
  });

  // --- public API ---

  return {
    container,
    input,
    list,
    liveRegion,

    setOptions(items: string[]): void {
      allItems = [...items].sort();
      // Restore current selection if still available
      if (currentValue && !allItems.includes(currentValue)) {
        currentValue = '';
        input.value = '';
        options.onChange?.('');
      }
    },

    getValue(): string {
      return currentValue;
    },

    setValue(val: string): void {
      currentValue = val;
      input.value = val;
      options.onChange?.(val);
    },
  };
}
