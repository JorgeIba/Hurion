import { describe, it, expect } from 'vitest';
import {
  sanitizeClassCode,
  sanitizeTab,
  getSolvedCount,
  sortUsersBySolved,
  formatTopicTitle,
  groupProblemsByTopic,
  parseDifficulty,
  getProblemDifficultyMetadata,
  computeAllProblemSolveStats,
  filterProblems,
} from './utils';
import { Problem, UserScore } from './types';
import { SCOREBOARD_CONFIG, SCOREBOARD_LABELS } from './constants';

describe('domain/scoreboard/utils', () => {
  describe('sanitizeClassCode', () => {
    it('accepts clean alphanumeric class codes with hyphens and underscores', () => {
      expect(sanitizeClassCode('basicos')).toBe('basicos');
      expect(sanitizeClassCode('avanzados_2024')).toBe('avanzados_2024');
      expect(sanitizeClassCode('class-pro-1')).toBe('class-pro-1');
    });

    it('strips path traversal tokens and dangerous symbols', () => {
      expect(sanitizeClassCode('../../etc/passwd')).toBe('etcpasswd');
      expect(sanitizeClassCode('class#1;drop table')).toBe('class1droptable');
      expect(sanitizeClassCode('  basicos  ')).toBe('basicos');
    });

    it('returns empty string for null, undefined, empty, or non-string inputs', () => {
      expect(sanitizeClassCode('')).toBe('');
      expect(sanitizeClassCode(null)).toBe('');
      expect(sanitizeClassCode(undefined)).toBe('');
      // @ts-expect-error - testing runtime JavaScript boundary
      expect(sanitizeClassCode(12345)).toBe('');
    });
  });

  describe('sanitizeTab', () => {
    it('accepts valid tab names', () => {
      expect(sanitizeTab('standings')).toBe('standings');
      expect(sanitizeTab('problems')).toBe('problems');
    });

    it('normalizes uppercase and whitespace in valid tab names', () => {
      expect(sanitizeTab('STANDINGS')).toBe('standings');
      expect(sanitizeTab('  problems  ')).toBe('problems');
    });

    it('falls back to default tab on invalid or empty inputs', () => {
      expect(sanitizeTab('invalid_tab')).toBe(SCOREBOARD_CONFIG.DEFAULT_TAB);
      expect(sanitizeTab('')).toBe(SCOREBOARD_CONFIG.DEFAULT_TAB);
      expect(sanitizeTab(undefined)).toBe(SCOREBOARD_CONFIG.DEFAULT_TAB);
    });
  });

  describe('getSolvedCount', () => {
    it('returns solvedCount when explicitly provided', () => {
      const user: UserScore = { userId: 'u1', solvedCount: 5, solved: [1, 2] };
      expect(getSolvedCount(user)).toBe(5);
    });

    it('falls back to solved array length if solvedCount is omitted', () => {
      const user: UserScore = { userId: 'u2', solved: [1, 2, 3] };
      expect(getSolvedCount(user)).toBe(3);
    });

    it('deduplicates multiple ACs when calculating fallback length', () => {
      const user: UserScore = { userId: 'u3', solved: [101, 101, 102] };
      expect(getSolvedCount(user)).toBe(2);
    });

    it('returns 0 for empty or undefined user', () => {
      expect(getSolvedCount(undefined)).toBe(0);
      expect(getSolvedCount({ userId: 'u4' })).toBe(0);
    });
  });

  describe('sortUsersBySolved', () => {
    it('sorts users descending by solved AC count', () => {
      const users: UserScore[] = [
        { userId: 'u1', solvedCount: 2 },
        { userId: 'u2', solvedCount: 10 },
        { userId: 'u3', solvedCount: 5 },
      ];
      const sorted = sortUsersBySolved(users);
      expect(sorted.map((u) => u.userId)).toEqual(['u2', 'u3', 'u1']);
    });

    it('does not mutate the original array', () => {
      const users: UserScore[] = [
        { userId: 'u1', solvedCount: 2 },
        { userId: 'u2', solvedCount: 10 },
      ];
      const copy = [...users];
      sortUsersBySolved(users);
      expect(users).toEqual(copy);
    });

    it('handles empty or undefined users', () => {
      expect(sortUsersBySolved([])).toEqual([]);
      expect(sortUsersBySolved(undefined)).toEqual([]);
    });
  });

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
      expect(formatTopicTitle('')).toBe(SCOREBOARD_CONFIG.DEFAULT_TOPIC_NAME);
      expect(formatTopicTitle('   ')).toBe(SCOREBOARD_CONFIG.DEFAULT_TOPIC_NAME);
      expect(formatTopicTitle(undefined)).toBe(SCOREBOARD_CONFIG.DEFAULT_TOPIC_NAME);
      // @ts-expect-error - testing runtime JavaScript non-string boundary
      expect(formatTopicTitle(1200)).toBe(SCOREBOARD_CONFIG.DEFAULT_TOPIC_NAME);
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
      expect(Array.from(groups.keys())).toEqual([SCOREBOARD_CONFIG.DEFAULT_TOPIC_NAME]);
      expect(groups.get(SCOREBOARD_CONFIG.DEFAULT_TOPIC_NAME)?.map((p) => p.id)).toEqual([1, 2]);
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
      expect(easyMeta.label).toBe(SCOREBOARD_LABELS.DIFFICULTY.EASY);

      const mediumMeta = getProblemDifficultyMetadata('medium');
      expect(mediumMeta.badgeClassName).toBe('badge-medium');
      expect(mediumMeta.dotClassName).toBe('diff-medium');
      expect(mediumMeta.rowClassName).toBe('problem-row-medium');
      expect(mediumMeta.label).toBe(SCOREBOARD_LABELS.DIFFICULTY.MEDIUM);

      const hardMeta = getProblemDifficultyMetadata('hard');
      expect(hardMeta.badgeClassName).toBe('badge-hard');
      expect(hardMeta.dotClassName).toBe('diff-hard');
      expect(hardMeta.rowClassName).toBe('problem-row-hard');
      expect(hardMeta.label).toBe(SCOREBOARD_LABELS.DIFFICULTY.HARD);

      const unspecifiedMeta = getProblemDifficultyMetadata('invalid');
      expect(unspecifiedMeta.badgeClassName).toBe('badge-unspecified');
      expect(unspecifiedMeta.dotClassName).toBe('diff-unspecified');
      expect(unspecifiedMeta.rowClassName).toBe('problem-row-unspecified');
      expect(unspecifiedMeta.label).toBe(SCOREBOARD_LABELS.DIFFICULTY.UNSPECIFIED);
    });
  });

  describe('computeAllProblemSolveStats', () => {
    it('computes solve statistics across all students accurately', () => {
      const problems: Problem[] = [
        { id: 101, name: 'P1' },
        { id: 102, name: 'P2' },
        { id: 103, name: 'P3' },
      ];
      const users: UserScore[] = [
        { userId: 'u1', solved: [101, 102] },
        { userId: 'u2', solved: [101] },
        { userId: 'u3', solved: [] },
      ];

      const statsMap = computeAllProblemSolveStats(problems, users);

      expect(statsMap.get(101)).toEqual({
        problemId: 101,
        solvedCount: 2,
        totalStudents: 3,
      });
      expect(statsMap.get(102)).toEqual({
        problemId: 102,
        solvedCount: 1,
        totalStudents: 3,
      });
      expect(statsMap.get(103)).toEqual({
        problemId: 103,
        solvedCount: 0,
        totalStudents: 3,
      });
    });

    it('deduplicates multiple solves of the same problem by the same student', () => {
      const problems: Problem[] = [{ id: 101, name: 'P1' }];
      const users: UserScore[] = [
        { userId: 'u1', solved: [101, 101, 101] }, // Duplicate solves in raw JSON
      ];

      const statsMap = computeAllProblemSolveStats(problems, users);
      expect(statsMap.get(101)?.solvedCount).toBe(1);
    });

    it('handles zero students without division by zero or errors', () => {
      const problems: Problem[] = [{ id: 101, name: 'P1' }];
      const statsMap = computeAllProblemSolveStats(problems, []);

      expect(statsMap.get(101)).toEqual({
        problemId: 101,
        solvedCount: 0,
        totalStudents: 0,
      });
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
