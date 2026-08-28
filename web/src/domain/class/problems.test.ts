import { describe, it, expect } from 'vitest';
import {
  formatTopicTitle,
  groupProblemsByTopic,
  parseDifficulty,
  getProblemDifficultyMetadata,
  filterProblems,
} from './problems';
import { Problem } from './types';
import { CLASS_CONFIG, CLASS_LABELS } from './constants';

describe('domain/class/problems', () => {
  describe('formatTopicTitle', () => {
    it('converts lowercase and mixed case strings to canonical Title Case', () => {
      expect(formatTopicTitle('dynamic programming')).toBe('Dynamic Programming');
      expect(formatTopicTitle('greedy')).toBe('Greedy');
      expect(formatTopicTitle('DATA STRUCTURES')).toBe('Data Structures');
    });

    it('handles separators like hyphens and underscores', () => {
      expect(formatTopicTitle('two-pointers')).toBe('Two Pointers');
      expect(formatTopicTitle('binary_search')).toBe('Binary Search');
      expect(formatTopicTitle('depth-first_search')).toBe('Depth First Search');
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
    it('groups problems by topic name', () => {
      const problems: Problem[] = [
        { id: 1, name: 'P1', topic: 'Math' },
        { id: 2, name: 'P2', topic: 'Greedy' },
        { id: 3, name: 'P3', topic: 'Math' },
      ];
      const groups = groupProblemsByTopic(problems);
      expect(Array.from(groups.keys())).toEqual(['Math', 'Greedy']);
      expect(groups.get('Math')?.map((p) => p.id)).toEqual([1, 3]);
      expect(groups.get('Greedy')?.map((p) => p.id)).toEqual([2]);
    });

    it('merges topics that normalize to the same title without resetting arrays', () => {
      const problems: Problem[] = [
        { id: 1, name: 'P1', topic: 'two-pointers' },
        { id: 2, name: 'P2', topic: 'two_pointers' },
        { id: 3, name: 'P3', topic: 'Two Pointers' },
      ];
      const groups = groupProblemsByTopic(problems);
      expect(Array.from(groups.keys())).toEqual(['Two Pointers']);
      expect(groups.get('Two Pointers')?.map((p) => p.id)).toEqual([1, 2, 3]);
    });

    it('assigns undefined or blank topic to General', () => {
      const problems: Problem[] = [
        { id: 1, name: 'P1' },
        { id: 2, name: 'P2', topic: '   ' },
      ];
      const groups = groupProblemsByTopic(problems);
      expect(Array.from(groups.keys())).toEqual([CLASS_CONFIG.DEFAULT_TOPIC_NAME]);
      expect(groups.get(CLASS_CONFIG.DEFAULT_TOPIC_NAME)?.map((p) => p.id)).toEqual([1, 2]);
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
      { id: 1, name: 'Two Sum', topic: 'Arrays', difficulty: 'easy' },
      { id: 2, name: 'Add Two Numbers', topic: 'Linked List', difficulty: 'medium' },
      { id: 3, name: 'Median of Two Arrays', topic: 'Binary Search', difficulty: 'hard' },
    ];

    it('filters by difficulty tier', () => {
      expect(filterProblems(sampleProblems, '', 'easy').map((p) => p.id)).toEqual([1]);
      expect(filterProblems(sampleProblems, '', 'medium').map((p) => p.id)).toEqual([2]);
      expect(filterProblems(sampleProblems, '', 'hard').map((p) => p.id)).toEqual([3]);
      expect(filterProblems(sampleProblems, '', 'all').length).toBe(3);
    });

    it('filters by text search query across name, ID, and topic', () => {
      expect(filterProblems(sampleProblems, 'two sum', 'all').map((p) => p.id)).toEqual([1]);
      expect(filterProblems(sampleProblems, 'linked list', 'all').map((p) => p.id)).toEqual([2]);
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
