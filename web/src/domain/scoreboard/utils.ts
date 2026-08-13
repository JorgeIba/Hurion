import { Problem, UserScore } from './types';

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
 * Defaults unassigned topics to 'General' while maintaining problem list sequence.
 */
export function groupProblemsByTopic(problems: Problem[]): Map<string, Problem[]> {
  const groups = new Map<string, Problem[]>();
  problems.forEach((p) => {
    const topicName = p.topic || 'General';
    if (!groups.has(topicName)) {
      groups.set(topicName, []);
    }
    groups.get(topicName)!.push(p);
  });
  return groups;
}
