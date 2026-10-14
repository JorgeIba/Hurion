import { describe, it, expect } from 'vitest';
import {
  formatTopicTitle,
  groupProblemsByTopic,
  parseDifficulty,
  getProblemDifficultyMetadata,
  filterProblems,
  compareProblems,
} from './problems';
import { Problem } from './types';
import { CLASS_CONFIG, CLASS_LABELS } from './constants';

describe('domain/class/problems', () => {
  it.each([
    ['id-highest-first', [8, 3, 2, 1]],
    ['id-lowest-first', [1, 2, 3, 8]],
    ['difficulty-easy-first', [8, 1, 2, 3]],
    ['difficulty-hard-first', [1, 2, 3, 8]],
    ['priority-essential-first', [1, 2, 3, 8]],
    ['priority-discouraged-first', [8, 1, 2, 3]],
  ] as const)(
    'compares a topic slice by %s with ID ties regardless of timestamps without mutating input',
    (sortBy, expectedIDs) => {
      const problems: Problem[] = [
        {
          id: 8,
          name: 'P8',
          topic: 'Z',
          difficulty: 'easy',
          priority: 'discouraged',
          createdAt: '2026-01-01T00:00:00Z',
        },
        {
          id: 3,
          name: 'P3',
          topic: 'Z',
          difficulty: 'hard',
          priority: 'essential',
          createdAt: '2025-01-01T00:00:00Z',
        },
        {
          id: 2,
          name: 'P2',
          topic: 'Z',
          difficulty: 'hard',
          priority: 'essential',
          createdAt: '2026-01-02T01:00:00+01:00',
        },
        {
          id: 1,
          name: 'P1',
          topic: 'Z',
          difficulty: 'hard',
          priority: 'essential',
          createdAt: '2026-01-02T00:00:00Z',
        },
        {
          id: 9,
          name: 'P9',
          topic: 'A',
          difficulty: 'easy',
          priority: 'essential',
          createdAt: '2026-01-01T00:00:00Z',
        },
      ];
      const groups = groupProblemsByTopic(problems);
      const ordered = [...groups.get('Z')!].sort((a, b) => compareProblems(a, b, sortBy));
      expect(ordered.map((p) => p.id)).toEqual(expectedIDs);
      expect([...groups.keys()]).toEqual(['A', 'Z']);
      expect(
        groupProblemsByTopic(problems, 'id-lowest-first')
          .get('Z')
          ?.map((p) => p.id)
      ).toEqual([1, 2, 3, 8]);
      expect([...groupProblemsByTopic(problems, 'id-lowest-first').keys()]).toEqual([
        'Z',
        'A',
      ]);
      expect(groups.get('Z')?.map((p) => p.id)).toEqual([8, 3, 2, 1]);
      expect(problems.map((p) => p.id)).toEqual([8, 3, 2, 1, 9]);
    }
  );

  it('orders all priority categories and combines priority with difficulty filters', () => {
    const problems: Problem[] = [
      'discouraged',
      null,
      'not_required',
      'supplementary',
      'recommended',
      'essential',
      'future_code',
    ].map((priority, id) => ({
      id,
      name: `P${id}`,
      priority,
      difficulty: id === 5 ? 'hard' : 'easy',
      createdAt: '2026-01-01T00:00:00Z',
    }));
    expect(
      [...problems]
        .sort((a, b) => compareProblems(a, b, 'priority-essential-first'))
        .map((p) => p.priority)
    ).toEqual([
      'essential',
      'recommended',
      'supplementary',
      'not_required',
      'discouraged',
      null,
      'future_code',
    ]);
    expect(filterProblems(problems, '', 'hard', 'essential').map((p) => p.id)).toEqual([
      5,
    ]);
    expect(filterProblems(problems, '', 'easy', 'essential')).toEqual([]);
    expect(filterProblems(problems, '', 'all', 'unset').map((p) => p.id)).toEqual([1]);
    expect(filterProblems(problems, '', 'all', 'not_required').map((p) => p.id)).toEqual([
      2,
    ]);
    expect(filterProblems(problems, '', 'all', 'all')).toEqual(problems);
    expect(
      compareProblems(problems[1], { ...problems[6], id: 1 }, 'priority-essential-first')
    ).toBe(0);
  });

  describe('formatTopicTitle', () => {
    it('preserves exact raw topic identifier strings and underscores', () => {
      expect(formatTopicTitle('Algoritmo_de_Eppstein')).toBe('Algoritmo_de_Eppstein');
      expect(formatTopicTitle('Max_Flow_Min_Cost')).toBe('Max_Flow_Min_Cost');
      expect(formatTopicTitle('two-pointers')).toBe('two-pointers');
      expect(formatTopicTitle('greedy')).toBe('greedy');
    });

    it('falls back to default topic name on empty or non-string inputs', () => {
      expect(formatTopicTitle('')).toBe(CLASS_CONFIG.DEFAULT_TOPIC_NAME);
      expect(formatTopicTitle('   ')).toBe(CLASS_CONFIG.DEFAULT_TOPIC_NAME);
      expect(formatTopicTitle(undefined)).toBe(CLASS_CONFIG.DEFAULT_TOPIC_NAME);
      // @ts-expect-error - testing runtime JavaScript non-string boundary
      expect(formatTopicTitle(1200)).toBe(CLASS_CONFIG.DEFAULT_TOPIC_NAME);
    });
  });

  describe('groupProblemsByTopic', () => {
    it('orders topics by their minimum ID and problems by highest ID by default without mutating input', () => {
      const problems: Problem[] = [
        { id: 8, name: 'P8', topic: 'Z', createdAt: '2025-01-01T00:00:00Z' },
        { id: 2, name: 'P2', topic: 'A', createdAt: '2026-01-01T00:00:00Z' },
        { id: 1, name: 'P1', topic: 'Z', createdAt: '2026-01-01T00:00:00Z' },
      ];
      const groups = groupProblemsByTopic(problems);
      expect([...groups.keys()]).toEqual(['Z', 'A']);
      expect(groups.get('Z')?.map((p) => p.id)).toEqual([8, 1]);
      expect(
        groupProblemsByTopic(problems, 'id-lowest-first')
          .get('Z')
          ?.map((p) => p.id)
      ).toEqual([1, 8]);
      expect(problems.map((p) => p.id)).toEqual([8, 2, 1]);
    });

    it('groups problems by exact topic name', () => {
      const problems: Problem[] = [
        {
          createdAt: '2026-10-04T00:00:00Z',
          id: 1,
          name: 'P1',
          topic: 'Algoritmo_de_Eppstein',
        },
        { createdAt: '2026-10-04T00:00:00Z', id: 2, name: 'P2', topic: 'Greedy' },
        {
          createdAt: '2026-10-04T00:00:00Z',
          id: 3,
          name: 'P3',
          topic: 'Algoritmo_de_Eppstein',
        },
      ];
      const groups = groupProblemsByTopic(problems);
      expect(Array.from(groups.keys())).toEqual(['Algoritmo_de_Eppstein', 'Greedy']);
      expect(groups.get('Algoritmo_de_Eppstein')?.map((p) => p.id)).toEqual([3, 1]);
      expect(
        groupProblemsByTopic(problems, 'id-lowest-first')
          .get('Algoritmo_de_Eppstein')
          ?.map((p) => p.id)
      ).toEqual([1, 3]);
      expect(groups.get('Greedy')?.map((p) => p.id)).toEqual([2]);
    });

    it('preserves topic keys with underscores without converting to title case', () => {
      const problems: Problem[] = [
        { createdAt: '2026-10-04T00:00:00Z', id: 1, name: 'P1', topic: 'two_pointers' },
        { createdAt: '2026-10-04T00:00:00Z', id: 2, name: 'P2', topic: 'two_pointers' },
      ];
      const groups = groupProblemsByTopic(problems);
      expect(Array.from(groups.keys())).toEqual(['two_pointers']);
      expect(groups.get('two_pointers')?.map((p) => p.id)).toEqual([2, 1]);
    });

    it('assigns undefined or blank topic to General', () => {
      const problems: Problem[] = [
        { createdAt: '2026-10-04T00:00:00Z', id: 1, name: 'P1' },
        { createdAt: '2026-10-04T00:00:00Z', id: 2, name: 'P2', topic: '   ' },
      ];
      const groups = groupProblemsByTopic(problems);
      expect(Array.from(groups.keys())).toEqual([CLASS_CONFIG.DEFAULT_TOPIC_NAME]);
      expect(groups.get(CLASS_CONFIG.DEFAULT_TOPIC_NAME)?.map((p) => p.id)).toEqual([
        2, 1,
      ]);
    });

    it('safely handles non-string topic properties and sparse arrays', () => {
      const problems = [
        null as unknown as Problem,
        {
          createdAt: '2026-10-04T00:00:00Z',
          id: 1,
          name: 'P1',
          topic: 1200 as unknown as string,
        },
        undefined as unknown as Problem,
      ];
      const groups = groupProblemsByTopic(problems);
      expect(Array.from(groups.keys())).toEqual([CLASS_CONFIG.DEFAULT_TOPIC_NAME]);
      expect(groups.get(CLASS_CONFIG.DEFAULT_TOPIC_NAME)?.map((p) => p.id)).toEqual([1]);
    });

    it('handles empty problems list', () => {
      const groups = groupProblemsByTopic([]);
      expect(groups.size).toBe(0);
    });
  });

  describe('parseDifficulty & getProblemDifficultyMetadata', () => {
    it('normalizes difficulty tiers accurately', () => {
      expect(parseDifficulty('easy')).toBe('easy');
      expect(parseDifficulty('EASY')).toBe('easy');
      expect(parseDifficulty('medium')).toBe('medium');
      expect(parseDifficulty('hard')).toBe('hard');
      expect(parseDifficulty('unknown')).toBe('unspecified');
      expect(parseDifficulty(undefined)).toBe('unspecified');
      // @ts-expect-error - testing runtime JavaScript non-string boundary
      expect(parseDifficulty(1200)).toBe('unspecified');
    });

    it('resolves correct styling metadata for all difficulty tiers', () => {
      const easyMeta = getProblemDifficultyMetadata('easy');
      expect(easyMeta.badgeClassName).toBe('badge-easy');
      expect(easyMeta.dotClassName).toBe('diff-easy');
      expect(easyMeta.rowClassName).toBe('problem-row-easy');
      expect(easyMeta.label).toBe(CLASS_LABELS.DIFFICULTY.EASY);

      const mediumMeta = getProblemDifficultyMetadata('medium');
      expect(mediumMeta.badgeClassName).toBe('badge-medium');
      expect(mediumMeta.dotClassName).toBe('diff-medium');
      expect(mediumMeta.rowClassName).toBe('problem-row-medium');
      expect(mediumMeta.label).toBe(CLASS_LABELS.DIFFICULTY.MEDIUM);

      const hardMeta = getProblemDifficultyMetadata('hard');
      expect(hardMeta.badgeClassName).toBe('badge-hard');
      expect(hardMeta.dotClassName).toBe('diff-hard');
      expect(hardMeta.rowClassName).toBe('problem-row-hard');
      expect(hardMeta.label).toBe(CLASS_LABELS.DIFFICULTY.HARD);

      const unspecifiedMeta = getProblemDifficultyMetadata('invalid');
      expect(unspecifiedMeta.badgeClassName).toBe('badge-unspecified');
      expect(unspecifiedMeta.dotClassName).toBe('diff-unspecified');
      expect(unspecifiedMeta.rowClassName).toBe('problem-row-unspecified');
      expect(unspecifiedMeta.label).toBe(CLASS_LABELS.DIFFICULTY.UNSPECIFIED);
    });
  });

  describe('filterProblems', () => {
    const sampleProblems: Problem[] = [
      {
        createdAt: '2026-10-04T00:00:00Z',
        id: 1,
        name: 'Two Sum',
        topic: 'Arrays',
        difficulty: 'easy',
      },
      {
        createdAt: '2026-10-04T00:00:00Z',
        id: 2,
        name: 'Add Two Numbers',
        topic: 'Linked List',
        difficulty: 'medium',
      },
      {
        createdAt: '2026-10-04T00:00:00Z',
        id: 3,
        name: 'Median of Two Arrays',
        topic: 'Binary Search',
        difficulty: 'hard',
      },
    ];

    it('filters by difficulty tier', () => {
      expect(filterProblems(sampleProblems, '', 'easy').map((p) => p.id)).toEqual([1]);
      expect(filterProblems(sampleProblems, '', 'medium').map((p) => p.id)).toEqual([2]);
      expect(filterProblems(sampleProblems, '', 'hard').map((p) => p.id)).toEqual([3]);
      expect(filterProblems(sampleProblems, '', 'all').length).toBe(3);
    });

    it('filters by text search query across name, ID, and topic', () => {
      expect(filterProblems(sampleProblems, 'two sum', 'all').map((p) => p.id)).toEqual([
        1,
      ]);
      expect(
        filterProblems(sampleProblems, 'linked list', 'all').map((p) => p.id)
      ).toEqual([2]);
      expect(filterProblems(sampleProblems, '3', 'all').map((p) => p.id)).toEqual([3]);
    });

    it('combines text search and difficulty filter', () => {
      expect(filterProblems(sampleProblems, 'two', 'easy').map((p) => p.id)).toEqual([1]);
      expect(filterProblems(sampleProblems, 'two', 'hard').map((p) => p.id)).toEqual([3]);
      expect(filterProblems(sampleProblems, 'two', 'unspecified').length).toBe(0);
    });

    it('returns empty array when nothing matches', () => {
      expect(filterProblems(sampleProblems, 'nonexistent', 'all')).toEqual([]);
    });

    it('safely handles undefined or non-string queries', () => {
      expect(filterProblems(sampleProblems, undefined, 'all').length).toBe(3);
      // @ts-expect-error - testing runtime JavaScript non-string boundary
      expect(filterProblems(sampleProblems, 12345, 'all').length).toBe(3);
    });
  });
});
