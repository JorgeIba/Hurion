import type { ProblemPriority } from '@/domain/class/types';
import {
  priorityPresentationConfig,
  type PriorityKey,
  type PriorityPresentation,
  type PriorityPresentationConfig,
  type PriorityPresentationTable,
} from './priorityConfig';

export function resolvePriorityPresentation(
  classCode?: string,
  config: PriorityPresentationConfig = priorityPresentationConfig
): PriorityPresentationTable {
  const presentation = { ...config.defaults };
  const overrides = Object.prototype.hasOwnProperty.call(config.classes, classCode ?? '')
    ? config.classes[classCode ?? '']
    : undefined;

  for (const [key, override] of Object.entries(overrides ?? {})) {
    if (Object.prototype.hasOwnProperty.call(presentation, key)) {
      const priority = key as PriorityKey;
      presentation[priority] = { ...presentation[priority], ...override };
    }
  }

  return presentation;
}

export function getPriorityPresentation(
  priority?: string | null,
  presentation: PriorityPresentationTable = priorityPresentationConfig.defaults
): PriorityPresentation {
  if (priority == null) return presentation.unset;
  if (
    priority === 'unset' ||
    !Object.prototype.hasOwnProperty.call(presentation, priority)
  ) {
    return { label: priority };
  }
  return presentation[priority as ProblemPriority];
}

export function getProblemPriorityLabel(
  priority?: string | null,
  presentation?: PriorityPresentationTable
): string {
  const { label, symbol } = getPriorityPresentation(priority, presentation);
  return symbol ? `${symbol} ${label}` : label;
}
