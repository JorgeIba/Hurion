import { describe, it, expect } from 'vitest';
import { getNextSortState, sortTopicGroupsByStudent } from './sorting';
import { Problem, UserScore, SortState } from './types';

describe('domain/scoreboard/sorting', () => {
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
      { id: 1, name: 'P1', topic: 'DP' },
      { id: 2, name: 'P2', topic: 'DP' },
      { id: 3, name: 'P3', topic: 'Greedy' },
      { id: 4, name: 'P4', topic: 'Greedy' },
      { id: 5, name: 'P5', topic: 'Math' },
    ];

    const users: UserScore[] = [
      {
        userId: 'student1',
        // DP: 2/2 solved, Greedy: 1/2 solved (P3), Math: 0/1 solved
        solved: [1, 2, 3],
      },
    ];

    it('returns default topic order when sortState is null or inactive', () => {
      const nullState: SortState = { userId: null, direction: null };
      const groups = sortTopicGroupsByStudent(problems, users, nullState);
      expect(Array.from(groups.keys())).toEqual(['Dp', 'Greedy', 'Math']);
      expect(groups.get('Dp')?.map((p) => p.id)).toEqual([1, 2]);
    });

    it('sorts topics and problems in desc mode (top solved first)', () => {
      const descState: SortState = { userId: 'student1', direction: 'desc' };
      const groups = sortTopicGroupsByStudent(problems, users, descState);

      // Topic Order: Dp (2 solved) -> Greedy (1 solved) -> Math (0 solved)
      expect(Array.from(groups.keys())).toEqual(['Dp', 'Greedy', 'Math']);

      // Inside Greedy: P3 (solved) comes before P4 (unsolved)
      expect(groups.get('Greedy')?.map((p) => p.id)).toEqual([3, 4]);
    });

    it('sorts topics and problems in asc mode (Echale Ganas / Backlog First)', () => {
      const ascState: SortState = { userId: 'student1', direction: 'asc' };
      const groups = sortTopicGroupsByStudent(problems, users, ascState);

      // Backlog pending counts:
      // Math: 1 pending (0/1 solved, 0% ratio)
      // Greedy: 1 pending (1/2 solved, 50% ratio)
      // Dp: 0 pending (2/2 solved, 100% ratio)
      // Math and Greedy both have 1 pending, but Math has a lower ratio (0% < 50%), so Math comes first!
      expect(Array.from(groups.keys())).toEqual(['Math', 'Greedy', 'Dp']);

      // Inside Greedy: P4 (unsolved/backlog) comes before P3 (solved)
      expect(groups.get('Greedy')?.map((p) => p.id)).toEqual([4, 3]);
    });

    it('handles student not found in users gracefully', () => {
      const ghostState: SortState = { userId: 'nonexistent', direction: 'desc' };
      const groups = sortTopicGroupsByStudent(problems, users, ghostState);
      expect(Array.from(groups.keys())).toEqual(['Dp', 'Greedy', 'Math']);
    });
  });
});
