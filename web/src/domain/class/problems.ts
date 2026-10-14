import {
  Problem,
  ProblemDifficulty,
  DifficultyFilter,
  ProblemDifficultyMetadata,
  PriorityFilter,
  ProblemPriority,
  ProblemSort,
} from './types';
import { CLASS_CONFIG, CLASS_LABELS } from './constants';

export const PRIORITY_ORDER = [
  'essential',
  'recommended',
  'supplementary',
  'not_required',
  'discouraged',
  null, // Unset priority stays last in either sort direction
] as const satisfies readonly (ProblemPriority | null)[];

type SortField = 'difficulty' | 'priority';
type SortConfig = { field: SortField; direction: 1 | -1 };

const SORT_CONFIG = {
  'difficulty-easy-first': { field: 'difficulty', direction: 1 },
  'difficulty-hard-first': { field: 'difficulty', direction: -1 },
  'priority-essential-first': { field: 'priority', direction: 1 },
  'priority-discouraged-first': { field: 'priority', direction: -1 },
} satisfies Record<
  Exclude<ProblemSort, 'id' | 'id-lowest-first' | 'id-highest-first'>,
  SortConfig
>;

const DIFFICULTY_ORDER: Exclude<ProblemDifficulty, 'unspecified'>[] = [
  'easy',
  'medium',
  'hard',
];

function getProblemSortRank(problem: Problem, field: SortField): number | null {
  if (field === 'difficulty') {
    const difficulty = parseDifficulty(problem.difficulty);
    return difficulty === 'unspecified' ? null : DIFFICULTY_ORDER.indexOf(difficulty);
  }

  const priority = problem.priority ?? null;
  if (priority === null) return null;

  const index = PRIORITY_ORDER.findIndex((code) => code === priority);
  return index < 0 ? null : index;
}

export function compareProblems(a: Problem, b: Problem, sortBy: ProblemSort): number {
  if (sortBy === 'id-highest-first') return b.id - a.id;
  if (sortBy === 'id-lowest-first' || sortBy === 'id') return a.id - b.id;
  const { field, direction } = SORT_CONFIG[sortBy];
  const rankA = getProblemSortRank(a, field);
  const rankB = getProblemSortRank(b, field);

  const rankComparison =
    rankA === null
      ? rankB === null
        ? 0
        : 1
      : rankB === null
        ? -1
        : (rankA - rankB) * direction;

  return rankComparison || a.id - b.id;
}

/**
 * Preserves the exact topic identifier string (e.g. "Algoritmo_de_Eppstein").
 */
export function formatTopicTitle(rawTopic?: string): string {
  const trimmed = (typeof rawTopic === 'string' ? rawTopic : '').trim();
  if (!trimmed) return CLASS_CONFIG.DEFAULT_TOPIC_NAME;
  return trimmed;
}

/**
 * Groups problems into a Map keyed by topic section name.
 * Orders topics by highest problem ID by default (or lowest problem ID if sortBy is id-lowest-first),
 * and sorts problems within each topic using compareProblems.
 */
export function groupProblemsByTopic(
  problems: Problem[] = [],
  sortBy: ProblemSort = 'id-highest-first'
): Map<string, Problem[]> {
  const rawMap = new Map<string, Problem[]>();

  for (const p of problems || []) {
    if (!p) continue;
    const rawTopic = formatTopicTitle(p.topic);
    let list = rawMap.get(rawTopic);
    if (!list) {
      list = [];
      rawMap.set(rawTopic, list);
    }
    list.push(p);
  }

  for (const entries of rawMap.values()) {
    entries.sort((a, b) => compareProblems(a, b, sortBy));
  }

  const topicEntries = Array.from(rawMap.entries());
  const useLowestId = sortBy === 'id-lowest-first' || sortBy === 'id';

  topicEntries.sort(([topicA, probsA], [topicB, probsB]) => {
    if (useLowestId) {
      const minA = Math.min(...probsA.map((p) => p.id));
      const minB = Math.min(...probsB.map((p) => p.id));
      if (minA !== minB) return minA - minB;
      return topicA.localeCompare(topicB);
    }

    const maxA = Math.max(...probsA.map((p) => p.id));
    const maxB = Math.max(...probsB.map((p) => p.id));
    if (maxB !== maxA) return maxB - maxA;
    return topicA.localeCompare(topicB);
  });

  const groups = new Map<string, Problem[]>();
  for (const [topic, list] of topicEntries) {
    groups.set(topic, list);
  }
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
export function parseDifficulty(raw?: string | null): ProblemDifficulty {
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
  difficulty?: string | null
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
  difficultyFilter: DifficultyFilter = 'all',
  priorityFilter: PriorityFilter = 'all'
): Problem[] {
  const safeProblems = problems || [];
  const cleanQuery = (typeof query === 'string' ? query : '').trim().toLowerCase();

  return safeProblems.filter((p) => {
    if (!p) return false;
    if (
      priorityFilter === 'unset'
        ? p.priority != null
        : priorityFilter !== 'all' && p.priority !== priorityFilter
    ) {
      return false;
    }

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

/** Selects visible rows without redefining the full class's topic or problem order. */
export function filterProblemGroups(
  groups: ReadonlyMap<string, Problem[]>,
  difficultyFilter: DifficultyFilter,
  priorityFilter: PriorityFilter,
  problemIds?: ReadonlySet<number>
): Map<string, Problem[]> {
  const visibleGroups = new Map<string, Problem[]>();
  for (const [topic, entries] of groups) {
    const filtered = filterProblems(entries, '', difficultyFilter, priorityFilter);
    const visible = problemIds
      ? filtered.filter((problem) => problemIds.has(problem.id))
      : filtered;
    if (visible.length > 0) visibleGroups.set(topic, visible);
  }
  return visibleGroups;
}
