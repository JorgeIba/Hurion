import { describe, expect, it } from 'vitest';
import { getPriorityPresentation, resolvePriorityPresentation } from './priority';
import {
  priorityPresentationConfig,
  type PriorityPresentationConfig,
} from './priorityConfig';

describe('class priority overrides', () => {
  it('uses defaults for an inherited object property used as a class code', () => {
    expect(resolvePriorityPresentation('constructor')).toEqual(
      resolvePriorityPresentation()
    );
  });

  const config = {
    defaults: {
      ...priorityPresentationConfig.defaults,
      essential: { label: 'Essential', symbol: '!', description: 'Core practice' },
    },
    classes: {
      symbol_only: { essential: { symbol: '⭐' } },
      another_class: { essential: { label: 'Core exercises' } },
      no_marker: { essential: { symbol: '', description: '' } },
    },
  } satisfies PriorityPresentationConfig;

  it.each([
    {
      classCode: 'symbol_only',
      expected: { label: 'Essential', symbol: '⭐', description: 'Core practice' },
    },
    {
      classCode: 'another_class',
      expected: { label: 'Core exercises', symbol: '!', description: 'Core practice' },
    },
    {
      classCode: 'no_marker',
      expected: { label: 'Essential', symbol: '', description: '' },
    },
    {
      classCode: 'unknown_class',
      expected: { label: 'Essential', symbol: '!', description: 'Core practice' },
    },
  ])(
    'resolves $classCode without changing shared defaults',
    ({ classCode, expected }) => {
      const presentation = resolvePriorityPresentation(classCode, config);

      expect(getPriorityPresentation('essential', presentation)).toEqual(expected);
      expect(getPriorityPresentation('recommended', presentation)).toEqual({
        label: 'Recommended',
      });
      expect(config.defaults.essential).toEqual({
        label: 'Essential',
        symbol: '!',
        description: 'Core practice',
      });
    }
  );
});
