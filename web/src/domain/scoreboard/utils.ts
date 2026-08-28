import { Problem, UserScore, ProblemDifficulty, ActiveTab, DifficultyFilter, ProblemSolveStats } from './types';
import { SCOREBOARD_LABELS, SCOREBOARD_CONFIG } from './constants';

/**
 * Sanitizes a raw class code string.
 * Permits letters, numbers, dashes, and underscores (e.g. "basicos" or "Class123").
 * Strips invalid characters to prevent file path injection issues.
 */
export function sanitizeClassCode(rawCode: string | null | undefined): string {
  if (!rawCode || typeof rawCode !== 'string') return '';
  return rawCode.trim().replace(/[^a-zA-Z0-9_-]/g, '');
}

/**
 * Sanitizes a raw tab query parameter string against canonical ActiveTab identifiers.
 */
export function sanitizeTab(rawTab: string | null | undefined): ActiveTab {
  if (!rawTab || typeof rawTab !== 'string') return SCOREBOARD_CONFIG.DEFAULT_TAB;
  return rawTab.trim().toLowerCase() === 'problems' ? 'problems' : SCOREBOARD_CONFIG.DEFAULT_TAB;
}

/**
 * Calculates the total number of solved AC problems for a given user.
 * Prefers `u.solvedCount` if present, falling back to `u.solved.length`.
 */
export function getSolvedCount(u: UserScore): number {
  return u.solvedCount ?? (u.solved ? new Set(u.solved).size : 0);
}

/**
 * Sorts students descending by total solved AC count.
 * Returns a new array without mutating the original user array.
 */
export function sortUsersBySolved(users: UserScore[] = []): UserScore[] {
  return [...(users || [])].sort((a, b) => getSolvedCount(b) - getSolvedCount(a));
}

/**
 * Formats a raw topic string into canonical Title Case:
 * "dynamic programming" -> "Dynamic Programming"
 * "greedy" -> "Greedy"
 */
export function formatTopicTitle(rawTopic?: string): string {
  const trimmed = (rawTopic || '').trim();
  if (!trimmed) return SCOREBOARD_CONFIG.DEFAULT_TOPIC_NAME;

  return trimmed
    .split(/[\s_-]+/)
    .filter(Boolean)
    .map((word) => word.charAt(0).toUpperCase() + word.slice(1).toLowerCase())
    .join(' ');
}

/**
 * Groups problems into a Map keyed by topic section name.
 * Normalizes topic names case-insensitively while preserving canonical Title Case display names.
 */
export function groupProblemsByTopic(problems: Problem[] = []): Map<string, Problem[]> {
  const groups = new Map<string, Problem[]>();
  const canonicalNameMap = new Map<string, string>(); // lowerKey -> Canonical Title

  (problems || []).forEach((p) => {
    const rawTopic = (p.topic || '').trim() || SCOREBOARD_CONFIG.DEFAULT_TOPIC_NAME;
    const canonical = formatTopicTitle(rawTopic);
    const lowerKey = canonical.toLowerCase();

    if (!canonicalNameMap.has(lowerKey)) {
      canonicalNameMap.set(lowerKey, canonical);
      if (!groups.has(canonical)) {
        groups.set(canonical, []);
      }
    }

    const canonicalTitle = canonicalNameMap.get(lowerKey)!;
    groups.get(canonicalTitle)!.push(p);
  });

  return groups;
}

export interface ProblemDifficultyMetadata {
  rowClassName: string;
  badgeClassName: string;
  dotClassName: string;
  label: string;
}

/**
 * Declarative lookup table mapping normalized difficulty tiers to presentation metadata.
 */
const DIFFICULTY_METADATA_MAP: Record<ProblemDifficulty, ProblemDifficultyMetadata> = {
  easy: {
    rowClassName: 'problem-row-easy',
    badgeClassName: 'badge-easy',
    dotClassName: 'diff-easy',
    label: SCOREBOARD_LABELS.DIFFICULTY.EASY,
  },
  medium: {
    rowClassName: 'problem-row-medium',
    badgeClassName: 'badge-medium',
    dotClassName: 'diff-medium',
    label: SCOREBOARD_LABELS.DIFFICULTY.MEDIUM,
  },
  hard: {
    rowClassName: 'problem-row-hard',
    badgeClassName: 'badge-hard',
    dotClassName: 'diff-hard',
    label: SCOREBOARD_LABELS.DIFFICULTY.HARD,
  },
  unspecified: {
    rowClassName: 'problem-row-unspecified',
    badgeClassName: 'badge-unspecified',
    dotClassName: 'diff-unspecified',
    label: SCOREBOARD_LABELS.DIFFICULTY.UNSPECIFIED,
  },
};

/**
 * Normalizes an arbitrary raw difficulty string into a typed ProblemDifficulty tier.
 */
export function parseDifficulty(raw?: string): ProblemDifficulty {
  const normalized = (raw || '').trim().toLowerCase();
  if (normalized === 'easy') return 'easy';
  if (normalized === 'medium') return 'medium';
  if (normalized === 'hard') return 'hard';
  return 'unspecified';
}

/**
 * Resolves structured styling and label metadata for a problem difficulty level via O(1) lookup.
 */
export function getProblemDifficultyMetadata(difficulty?: string): ProblemDifficultyMetadata {
  const tier = parseDifficulty(difficulty);
  return DIFFICULTY_METADATA_MAP[tier];
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
    if (user.solved && Array.isArray(user.solved)) {
      const uniqueSolved = new Set(user.solved);
      for (const pid of uniqueSolved) {
        solveCountMap.set(pid, (solveCountMap.get(pid) || 0) + 1);
      }
    }
  }

  const result = new Map<number, ProblemSolveStats>();
  for (const p of safeProblems) {
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
 * Filters problem catalog by search query (matching name, numeric ID, or topic) and difficulty level.
 */
export function filterProblems(
  problems: Problem[] = [],
  query: string,
  difficultyFilter: DifficultyFilter
): Problem[] {
  const safeProblems = problems || [];
  const cleanQuery = query.trim().toLowerCase();

  return safeProblems.filter((p) => {
    // 1. Difficulty filter match
    if (difficultyFilter !== 'all') {
      const pDiff = parseDifficulty(p.difficulty);
      if (pDiff !== difficultyFilter) {
        return false;
      }
    }

    // 2. Text query match
    if (!cleanQuery) return true;

    const nameMatch = (p.name || '').toLowerCase().includes(cleanQuery);
    const idMatch = String(p.id).includes(cleanQuery);
    const topicMatch = (p.topic || '').toLowerCase().includes(cleanQuery);

    return nameMatch || idMatch || topicMatch;
  });
}
