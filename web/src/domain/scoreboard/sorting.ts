import { Problem, UserScore, SortState } from './types';
import { groupProblemsByTopic } from './utils';

/**
 * Computes the next tri-state sort cycle when a student column header is clicked:
 * - Different User clicked -> 'desc' (Top Solves First)
 * - Same User (currently 'desc') -> 'asc' (Échale Ganas / Backlog First)
 * - Same User (currently 'asc')  -> null (Reset to Default Curriculum Order)
 */
export function getNextSortState(currentState: SortState, clickedUserId: string): SortState {
  if (currentState.userId !== clickedUserId) {
    return { userId: clickedUserId, direction: 'desc' };
  }
  if (currentState.direction === 'desc') {
    return { userId: clickedUserId, direction: 'asc' };
  }
  return { userId: null, direction: null };
}

/**
 * Pure domain sorting engine that structures topic sections and problems relative to a student's performance.
 * 
 * - 'desc': Topics with highest solved count first. Within each topic, solved problems first.
 * - 'asc' (Échale Ganas): Topics with lowest solved count / highest pending first. Within each topic, unsolved problems first.
 * - null: Default curriculum topic grouping and problem ID sequence.
 * 
 * Deterministic tie-breakers:
 * 1. Topic completion percentage ratio (solved / total)
 * 2. Original topic chronological position
 * 3. Original problem numerical ID
 */
export function sortTopicGroupsByStudent(
  problems: Problem[],
  users: UserScore[],
  sortState: SortState
): Map<string, Problem[]> {
  const rawGroups = groupProblemsByTopic(problems);
  if (!sortState.userId || !sortState.direction) {
    return rawGroups;
  }

  const targetUser = users.find((u) => u.userId === sortState.userId);
  const targetSolvedSet = new Set(targetUser?.solved || []);
  const isDesc = sortState.direction === 'desc';

  const entries = Array.from(rawGroups.entries()).map(([topicName, probList], originalIndex) => {
    // 1. Sort problems inside this topic
    const sortedProbs = [...probList].sort((a, b) => {
      const aSolved = targetSolvedSet.has(a.id);
      const bSolved = targetSolvedSet.has(b.id);
      if (aSolved !== bSolved) {
        return isDesc ? (aSolved ? -1 : 1) : (!aSolved ? -1 : 1);
      }
      return a.id - b.id; // Deterministic tie-breaker by problem ID
    });

    const solvedCount = probList.filter((p) => targetSolvedSet.has(p.id)).length;
    const totalCount = probList.length;

    return {
      topicName,
      probList: sortedProbs,
      solvedCount,
      totalCount,
      originalIndex,
    };
  });

  // 2. Sort topic sections
  entries.sort((a, b) => {
    if (isDesc) {
      // Top ACs: most solved problems first
      if (a.solvedCount !== b.solvedCount) {
        return b.solvedCount - a.solvedCount;
      }
      const aRatio = a.totalCount > 0 ? a.solvedCount / a.totalCount : 0;
      const bRatio = b.totalCount > 0 ? b.solvedCount / b.totalCount : 0;
      if (bRatio !== aRatio) {
        return bRatio - aRatio;
      }
      return a.originalIndex - b.originalIndex;
    }

    // Echale Ganas (Backlog First): most pending problems first
    const aPending = a.totalCount - a.solvedCount;
    const bPending = b.totalCount - b.solvedCount;
    if (bPending !== aPending) {
      return bPending - aPending;
    }
    const aRatio = a.totalCount > 0 ? a.solvedCount / a.totalCount : 0;
    const bRatio = b.totalCount > 0 ? b.solvedCount / b.totalCount : 0;
    if (aRatio !== bRatio) {
      return aRatio - bRatio; // Lowest completion ratio first
    }
    return a.originalIndex - b.originalIndex;
  });

  const result = new Map<string, Problem[]>();
  for (const entry of entries) {
    result.set(entry.topicName, entry.probList);
  }
  return result;
}
