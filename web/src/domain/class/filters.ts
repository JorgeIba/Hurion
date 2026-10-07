import type { Problem, UserScore } from './types';
import { formatTopicTitle } from './problems';

export type Filter<T> =
  | { condition: T; anyOf?: never; allOf?: never }
  | { anyOf: readonly Filter<T>[]; condition?: never; allOf?: never }
  | { allOf: readonly Filter<T>[]; condition?: never; anyOf?: never };

export type StudentFilter =
  | { by: 'displayName'; value: string }
  | { by: 'userId'; value: string }
  | { by: 'solvedProblem'; value: number };

export type ProblemFilter = { by: 'name'; value: string } | { by: 'id'; value: number };

export type TopicFilter = { by: 'topicName'; value: string };

export type RowFilter = ProblemFilter | TopicFilter;

export interface ScoreboardFilter {
  students?: Filter<StudentFilter>;
  rows?: Filter<RowFilter>;
}

export interface ScoreboardFilterResult {
  userIds: ReadonlySet<string>;
  problemIds: ReadonlySet<number>;
}

// Names of the fields that can be matched against in a text search. These are used to select the appropriate matching rule.
export type MatchingTarget = 'displayName' | 'userId' | 'problemName' | 'topicName';

export function normalizeFilterText(text: string): string {
  return text
    .normalize('NFD')
    .replace(/[\u0300-\u036f]/g, '')
    .toLowerCase()
    .trim();
}

type TextMatchingRule = (candidate: string, query: string) => boolean;

// Basic substring matching rule. This is the default for all fields except topic names, which ignore underscores.
const contains: TextMatchingRule = (candidate, query) => candidate.includes(query);
const compareTopics: TextMatchingRule = (candidate, query) =>
  contains(candidate.replace(/_/g, ''), query.replace(/_/g, ''));

// Matching rules receive normalized text. Topics also ignore catalog underscores.
const MATCHING_RULES: Record<MatchingTarget, TextMatchingRule> = {
  displayName: contains,
  userId: contains,
  problemName: contains,
  topicName: compareTopics,
};

export function matchesFilterText(
  candidate: string,
  query: string,
  field: MatchingTarget
): boolean {
  return MATCHING_RULES[field](
    normalizeFilterText(candidate),
    normalizeFilterText(query)
  );
}

/** Empty OR is false; empty AND is true, following some/every semantics. */
function matchesFilter<T>(
  filter: Filter<T>,
  matchesCondition: (condition: T) => boolean
): boolean {
  if (filter.anyOf) {
    return filter.anyOf.some((child) => matchesFilter(child, matchesCondition));
  }
  if (filter.allOf) {
    return filter.allOf.every((child) => matchesFilter(child, matchesCondition));
  }
  return matchesCondition(filter.condition);
}

function matchesStudent(user: UserScore, condition: StudentFilter): boolean {
  switch (condition.by) {
    case 'displayName':
      return matchesFilterText(user.displayName ?? '', condition.value, 'displayName');
    case 'userId':
      return matchesFilterText(user.userId, condition.value, 'userId');
    case 'solvedProblem':
      return user.solved?.includes(condition.value) ?? false;
  }
}

function matchesRow(problem: Problem, condition: RowFilter): boolean {
  switch (condition.by) {
    case 'name':
      return matchesFilterText(problem.name, condition.value, 'problemName');
    case 'id':
      return problem.id === condition.value;
    case 'topicName':
      return matchesFilterText(
        formatTopicTitle(problem.topic),
        condition.value,
        'topicName'
      );
  }
}

/** Evaluates independent axes against complete class data without changing it. */
export function evaluateScoreboardFilter({
  problems,
  users,
  filters,
}: {
  problems: readonly Problem[];
  users: readonly UserScore[];
  filters?: ScoreboardFilter;
}): ScoreboardFilterResult {
  const userIds = new Set<string>();
  const problemIds = new Set<number>();

  for (const user of users) {
    if (
      !filters?.students ||
      matchesFilter(filters.students, (condition) => matchesStudent(user, condition))
    ) {
      userIds.add(user.userId);
    }
  }
  for (const problem of problems) {
    if (
      !filters?.rows ||
      matchesFilter(filters.rows, (condition) => matchesRow(problem, condition))
    ) {
      problemIds.add(problem.id);
    }
  }

  return { userIds, problemIds };
}
