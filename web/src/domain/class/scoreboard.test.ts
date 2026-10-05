import { describe, it, expect } from 'vitest';
import {
  sanitizeClassCode,
  sanitizeTab,
  getSolvedCount,
  sortUsersBySolved,
  computeAllProblemSolveStats,
  getNextSortState,
  sortTopicGroupsByStudent,
} from './scoreboard';
import { Problem, UserScore, SortState } from './types';
import { CLASS_CONFIG } from './constants';

describe('domain/class/scoreboard', () => {
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
      expect(sanitizeTab('invalid_tab')).toBe(CLASS_CONFIG.DEFAULT_TAB);
      expect(sanitizeTab('')).toBe(CLASS_CONFIG.DEFAULT_TAB);
      expect(sanitizeTab(undefined)).toBe(CLASS_CONFIG.DEFAULT_TAB);
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

  describe('computeAllProblemSolveStats', () => {
    it('computes solve statistics across all students accurately', () => {
      const problems: Problem[] = [
        { createdAt: '2026-10-04T00:00:00Z', id: 101, name: 'P1' },
        { createdAt: '2026-10-04T00:00:00Z', id: 102, name: 'P2' },
        { createdAt: '2026-10-04T00:00:00Z', id: 103, name: 'P3' },
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
      const problems: Problem[] = [
        { createdAt: '2026-10-04T00:00:00Z', id: 101, name: 'P1' },
      ];
      const users: UserScore[] = [{ userId: 'u1', solved: [101, 101, 101] }];

      const statsMap = computeAllProblemSolveStats(problems, users);
      expect(statsMap.get(101)?.solvedCount).toBe(1);
    });

    it('handles zero students without division by zero or errors', () => {
      const problems: Problem[] = [
        { createdAt: '2026-10-04T00:00:00Z', id: 101, name: 'P1' },
      ];
      const statsMap = computeAllProblemSolveStats(problems, []);

      expect(statsMap.get(101)).toEqual({
        problemId: 101,
        solvedCount: 0,
        totalStudents: 0,
      });
    });
  });

  describe('getNextSortState', () => {
    it('sets sort to desc when clicking a new student', () => {
      const initial: SortState = { userId: null, direction: null };
      expect(getNextSortState(initial, 'u1')).toEqual({
        userId: 'u1',
        direction: 'desc',
      });
    });

    it('cycles from desc to asc (Echale Ganas) when clicking the same student', () => {
      const descState: SortState = { userId: 'u1', direction: 'desc' };
      expect(getNextSortState(descState, 'u1')).toEqual({
        userId: 'u1',
        direction: 'asc',
      });
    });

    it('resets sort state to null when clicking the same student in asc mode', () => {
      const ascState: SortState = { userId: 'u1', direction: 'asc' };
      expect(getNextSortState(ascState, 'u1')).toEqual({
        userId: null,
        direction: null,
      });
    });

    it('resets to desc for the new student when switching active students', () => {
      const activeState: SortState = { userId: 'u1', direction: 'asc' };
      expect(getNextSortState(activeState, 'u2')).toEqual({
        userId: 'u2',
        direction: 'desc',
      });
    });
  });

  describe('sortTopicGroupsByStudent', () => {
    const problems: Problem[] = [
      { createdAt: '2026-10-04T00:00:00Z', id: 1, name: 'P1', topic: 'DP' },
      { createdAt: '2026-10-04T00:00:00Z', id: 2, name: 'P2', topic: 'DP' },
      { createdAt: '2026-10-04T00:00:00Z', id: 3, name: 'P3', topic: 'Greedy' },
      { createdAt: '2026-10-04T00:00:00Z', id: 4, name: 'P4', topic: 'Greedy' },
      { createdAt: '2026-10-04T00:00:00Z', id: 5, name: 'P5', topic: 'Math' },
    ];

    const users: UserScore[] = [
      {
        userId: 'student1',
        solved: [1, 2, 3],
      },
    ];

    it('returns default topic order when sortState is null or inactive', () => {
      const nullState: SortState = { userId: null, direction: null };
      const groups = sortTopicGroupsByStudent([...problems].reverse(), users, nullState);
      expect(Array.from(groups.keys())).toEqual(['DP', 'Greedy', 'Math']);
      expect(groups.get('DP')?.map((p) => p.id)).toEqual([1, 2]);
    });

    it('sorts topics and problems in desc mode (top solved first)', () => {
      const descState: SortState = { userId: 'student1', direction: 'desc' };
      const groups = sortTopicGroupsByStudent(problems, users, descState);

      expect(Array.from(groups.keys())).toEqual(['DP', 'Greedy', 'Math']);
      expect(groups.get('Greedy')?.map((p) => p.id)).toEqual([3, 4]);
    });

    it('sorts topics and problems in asc mode (Echale Ganas / Backlog First)', () => {
      const ascState: SortState = { userId: 'student1', direction: 'asc' };
      const groups = sortTopicGroupsByStudent(problems, users, ascState);

      expect(Array.from(groups.keys())).toEqual(['Math', 'Greedy', 'DP']);
      expect(groups.get('Greedy')?.map((p) => p.id)).toEqual([4, 3]);
    });

    it('handles student not found in users gracefully', () => {
      const ghostState: SortState = {
        userId: 'nonexistent',
        direction: 'desc',
      };
      const groups = sortTopicGroupsByStudent(problems, users, ghostState);
      expect(Array.from(groups.keys())).toEqual(['DP', 'Greedy', 'Math']);
    });
  });
});
