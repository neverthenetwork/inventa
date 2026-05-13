/**
 * Dropdown population and pattern filtering for the 2D graph view.
 */

import type { Node } from './element-data';

/**
 * Remove all options from a select element.
 */
function removeOptions(selectElement: HTMLSelectElement): void {
  for (let i = selectElement.options.length - 1; i >= 0; i--) {
    selectElement.remove(i);
  }
}

/**
 * Populate src and dst dropdowns with sorted node labels.
 * Preserves the currently selected values.
 */
export function updateDropdowns(
  nodes: Node[],
  srcSelect: HTMLSelectElement,
  dstSelect: HTMLSelectElement,
): void {
  const srcValue = srcSelect.value;
  const dstValue = dstSelect.value;

  removeOptions(srcSelect);
  removeOptions(dstSelect);

  // Add "None" option
  const noneOpt = document.createElement('option');
  noneOpt.text = 'None';
  noneOpt.value = '';
  srcSelect.add(noneOpt);

  const noneOptD = document.createElement('option');
  noneOptD.text = 'None';
  noneOptD.value = '';
  dstSelect.add(noneOptD);

  // Collect and sort labels
  const labels = nodes
    .map((n) => n.data.label)
    .filter((l): l is string => typeof l === 'string')
    .sort();

  for (const label of labels) {
    const opt = document.createElement('option');
    opt.text = label;
    opt.value = label;
    srcSelect.add(opt);

    const dOpt = document.createElement('option');
    dOpt.text = label;
    dOpt.value = label;
    dstSelect.add(dOpt);
  }

  // Restore selections
  srcSelect.value = srcValue;
  dstSelect.value = dstValue;
}

/**
 * Check whether a name matches any of the given include patterns.
 * A match occurs when the pattern string is a substring of the name.
 * Empty patterns (empty string lines) are ignored.
 */
export function matchesPatterns(
  name: string,
  patterns: string[],
): boolean {
  if (patterns.length === 0) return true;
  return patterns.some((p) => p.length > 0 && name.includes(p));
}
