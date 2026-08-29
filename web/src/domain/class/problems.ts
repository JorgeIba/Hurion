import {
  Problem,
  ProblemDifficulty,
  DifficultyFilter,
  ProblemDifficultyMetadata,
} from './types';
import { CLASS_CONFIG, CLASS_LABELS } from './constants';

/**
 * Formats a raw topic string into canonical Title Case:
 * "dynamic programming" -> "Dynamic Programming"
 * "greedy" -> "Greedy"
 * "two-pointers" -> "Two Pointers"
 */
export function formatTopicTitle(rawTopic?: string): string {
  const trimmed = (typeof rawTopic === 'string' ? rawTopic : '').trim();
  if (!trimmed) return CLASS_CONFIG.DEFAULT_TOPIC_NAME;

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
    if (!p) return;
    const rawTopic = (p.topic || '').trim() || CLASS_CONFIG.DEFAULT_TOPIC_NAME;
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

/**
 * Declarative lookup table mapping normalized difficulty tiers to presentation metadata.
 */
const DIFFICULTY_METADATA_MAP: Record<ProblemDifficulty, ProblemDifficultyMetadata> = {
  easy: {
    rowClassName: 'problem-row-easy',
    badgeClassName: 'badge-easy',
    dotClassName: 'diff-easy',
    label: CLASS_LABELS.DIFFICULTY.EASY,
  },
  medium: {
    rowClassName: 'problem-row-medium',
    badgeClassName: 'badge-medium',
    dotClassName: 'diff-medium',
    label: CLASS_LABELS.DIFFICULTY.MEDIUM,
  },
  hard: {
    rowClassName: 'problem-row-hard',
    badgeClassName: 'badge-hard',
    dotClassName: 'diff-hard',
    label: CLASS_LABELS.DIFFICULTY.HARD,
  },
  unspecified: {
    rowClassName: 'problem-row-unspecified',
    badgeClassName: 'badge-unspecified',
    dotClassName: 'diff-unspecified',
    label: CLASS_LABELS.DIFFICULTY.UNSPECIFIED,
  },
};

/**
 * Normalizes an arbitrary raw difficulty string into a typed ProblemDifficulty tier.
 */
export function parseDifficulty(raw?: string): ProblemDifficulty {
  const normalized = (typeof raw === 'string' ? raw : '').trim().toLowerCase();
  if (normalized === 'easy') return 'easy';
  if (normalized === 'medium') return 'medium';
  if (normalized === 'hard') return 'hard';
  return 'unspecified';
}

/**
 * Resolves structured styling and label metadata for a problem difficulty level via O(1) lookup.
 */
export function getProblemDifficultyMetadata(
  difficulty?: string
): ProblemDifficultyMetadata {
  const tier = parseDifficulty(difficulty);
  return DIFFICULTY_METADATA_MAP[tier];
}

/**
 * Filters problem catalog by search query (matching name, numeric ID, or topic) and difficulty level.
 */
export function filterProblems(
  problems: Problem[] = [],
  query: string = '',
  difficultyFilter: DifficultyFilter = 'all'
): Problem[] {
  const safeProblems = problems || [];
  const cleanQuery = (typeof query === 'string' ? query : '').trim().toLowerCase();

  return safeProblems.filter((p) => {
    if (!p) return false;

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
