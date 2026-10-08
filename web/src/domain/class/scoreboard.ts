import { Problem, UserScore, ActiveTab, ProblemSolveStats, SortState } from './types';
import { ACTIVE_TABS, CLASS_CONFIG } from './constants';
import { groupProblemsByTopic } from './problems';

/**
 * Sanitizes a raw class code string.
 * Permits letters, numbers, dashes, and underscores (e.g. "basicos" or "Class123").
 * Strips invalid characters to prevent file path injection issues.
 */
export function sanitizeClassCode(rawCode: string | null | undefined): string {
  if (!rawCode || typeof rawCode !== 'string') return '';
  return rawCode.trim().replace(/[^a-zA-Z0-9_-]/g, '');
}

function isActiveTab(value: string): value is ActiveTab {
  return ACTIVE_TABS.some((tab) => tab === value);
}

/** Sanitizes a URL tab against the same values that define ActiveTab. */
export function sanitizeTab(
  rawTab: string | null | undefined,
  fallback: ActiveTab = CLASS_CONFIG.DEFAULT_TAB
): ActiveTab {
  if (!rawTab || typeof rawTab !== 'string') return fallback;
  const tab = rawTab.trim().toLowerCase();
  return isActiveTab(tab) ? tab : fallback;
}

/**
 * Calculates the total number of solved AC problems for a given user.
 * Prefers `u.solvedCount` if present, falling back to unique length of `u.solved`.
 */
export function getSolvedCount(u?: UserScore): number {
  if (!u) return 0;
  return u.solvedCount ?? (Array.isArray(u.solved) ? new Set(u.solved).size : 0);
}

/**
 * Sorts students descending by total solved AC count.
 * Returns a new array without mutating the original user array.
 */
export function sortUsersBySolved(users: UserScore[] = []): UserScore[] {
  return [...(users || [])].sort((a, b) => getSolvedCount(b) - getSolvedCount(a));
}

/**
 * Computes aggregate class solve statistics for all problems in a single O(U * S + P) pass.
 * Uses a per-user Set to guarantee idempotency and prevent overcounting duplicate solve IDs.
 */
export function computeAllProblemSolveStats(
  problems: Problem[] = [],
  users: UserScore[] = []
): Map<number, ProblemSolveStats> {
  const safeUsers = users || [];
  const safeProblems = problems || [];
  const totalStudents = safeUsers.length;
  const solveCountMap = new Map<number, number>();

  for (const user of safeUsers) {
    if (user && user.solved && Array.isArray(user.solved)) {
      const uniqueSolved = new Set(user.solved);
      for (const pid of uniqueSolved) {
        solveCountMap.set(pid, (solveCountMap.get(pid) || 0) + 1);
      }
    }
  }

  const result = new Map<number, ProblemSolveStats>();
  for (const p of safeProblems) {
    if (!p) continue;
    const solvedCount = solveCountMap.get(p.id) || 0;
    result.set(p.id, {
      problemId: p.id,
      solvedCount,
      totalStudents,
    });
  }

  return result;
}

/**
 * Computes the next tri-state sort cycle when a student column header is clicked:
 * - Different User clicked -> 'desc' (Top Solves First)
 * - Same User (currently 'desc') -> 'asc' (Échale Ganas / Backlog First)
 * - Same User (currently 'asc')  -> null (Reset to Default Curriculum Order)
 */
export function getNextSortState(
  currentState: SortState,
  clickedUserId: string
): SortState {
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

  const entries = Array.from(rawGroups.entries()).map(
    ([topicName, probList], originalIndex) => {
      // 1. Sort problems inside this topic
      const sortedProbs = [...probList].sort((a, b) => {
        const aSolved = targetSolvedSet.has(a.id);
        const bSolved = targetSolvedSet.has(b.id);
        if (aSolved !== bSolved) {
          return isDesc ? (aSolved ? -1 : 1) : !aSolved ? -1 : 1;
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
    }
  );

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
