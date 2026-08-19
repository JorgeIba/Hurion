import { Problem, UserScore, ProblemDifficulty } from './types';
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
 * Calculates the total number of solved AC problems for a given user.
 * Prefers `u.solvedCount` if present, falling back to `u.solved.length`.
 */
export function getSolvedCount(u: UserScore): number {
  return u.solvedCount ?? u.solved?.length ?? 0;
}

/**
 * Sorts students descending by total solved AC count.
 * Returns a new array without mutating the original user array.
 */
export function sortUsersBySolved(users: UserScore[]): UserScore[] {
  return [...users].sort((a, b) => getSolvedCount(b) - getSolvedCount(a));
}

/**
 * Groups problems into a Map keyed by topic section name (e.g. "// TOPIC: ARRAYS & HASHING").
 * Defaults unassigned topics to DEFAULT_TOPIC_NAME while maintaining problem list sequence.
 */
export function groupProblemsByTopic(problems: Problem[]): Map<string, Problem[]> {
  const groups = new Map<string, Problem[]>();
  problems.forEach((p) => {
    const topicName = p.topic || SCOREBOARD_CONFIG.DEFAULT_TOPIC_NAME;
    if (!groups.has(topicName)) {
      groups.set(topicName, []);
    }
    groups.get(topicName)!.push(p);
  });
  return groups;
}

export interface ProblemDifficultyMetadata {
  rowClass: string;
  badgeClass: string;
  dotClass: string;
  label: string;
}

/**
 * Declarative lookup table mapping normalized difficulty tiers to presentation metadata.
 */
const DIFFICULTY_METADATA_MAP: Record<ProblemDifficulty, ProblemDifficultyMetadata> = {
  easy: {
    rowClass: 'problem-row-easy',
    badgeClass: 'badge-easy',
    dotClass: 'diff-easy',
    label: SCOREBOARD_LABELS.DIFFICULTY.EASY,
  },
  medium: {
    rowClass: 'problem-row-medium',
    badgeClass: 'badge-medium',
    dotClass: 'diff-medium',
    label: SCOREBOARD_LABELS.DIFFICULTY.MEDIUM,
  },
  hard: {
    rowClass: 'problem-row-hard',
    badgeClass: 'badge-hard',
    dotClass: 'diff-hard',
    label: SCOREBOARD_LABELS.DIFFICULTY.HARD,
  },
  unspecified: {
    rowClass: 'problem-row-unspecified',
    badgeClass: 'badge-unspecified',
    dotClass: 'diff-unspecified',
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

