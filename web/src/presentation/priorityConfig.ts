import type { ProblemPriority } from '@/domain/class/types';

// "unset" represents missing priority in the UI, not a stored code.
export type PriorityKey = ProblemPriority | 'unset';

export interface PriorityPresentation {
  readonly label: string;
  readonly symbol?: string;
  readonly description?: string;
}

export type PriorityPresentationTable = Readonly<
  Record<PriorityKey, PriorityPresentation>
>;

export interface PriorityPresentationConfig {
  defaults: PriorityPresentationTable;
  classes: Readonly<
    Record<string, Partial<Record<PriorityKey, Partial<PriorityPresentation>>>>
  >;
}

// The default presentation configuration for problem priorities, including class-specific overrides.
export const priorityPresentationConfig = {
  defaults: {
    essential: { label: 'Essential' },
    recommended: { label: 'Recommended' },
    supplementary: { label: 'Supplementary' },
    not_required: { label: 'Not required' },
    discouraged: { label: 'Discouraged' },
    unset: { label: 'No priority assigned' },
  },
  classes: {
    avanzados: {
      essential: { symbol: '⭐⭐⭐' },
      recommended: { label: 'To Qualify to PDA', symbol: '⭐⭐' },
      supplementary: { label: 'For PDA Champ', symbol: '⭐' },
      not_required: { label: 'No Priority' },
      discouraged: { label: 'No lo hagan es basura', symbol: '🌑' },
      unset: { label: 'No Priority' },
    },
  },
} satisfies PriorityPresentationConfig;
