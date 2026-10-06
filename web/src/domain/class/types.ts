// 'unspecified' is a UI/filter fallback, not a stored difficulty code.
export type ProblemDifficulty = 'easy' | 'medium' | 'hard' | 'unspecified';
export type DifficultyFilter = 'all' | ProblemDifficulty;
export type ProblemPriority =
  | 'essential'
  | 'recommended'
  | 'supplementary'
  | 'not_required'
  | 'discouraged';
export type PriorityFilter = 'all' | 'unset' | ProblemPriority;
export type ProblemSort =
  | 'difficulty-easy-first'
  | 'difficulty-hard-first'
  | 'priority-essential-first'
  | 'priority-discouraged-first';
export type ActiveTab = 'standings' | 'problems';

export interface Problem {
  id: number;
  createdAt: string;
  name: string;
  url?: string;
  topic?: string;
  difficulty?: ProblemDifficulty | string | null;
  priority?: string | null;
}

export interface UserScore {
  userId: string;
  displayName?: string;
  solvedCount?: number;
  solved?: number[];
}

export interface ProblemSolveStats {
  problemId: number;
  solvedCount: number;
  totalStudents: number;
}

export type SortDirection = 'desc' | 'asc' | null;

export interface SortState {
  userId: string | null;
  direction: SortDirection;
}

export interface ClassData {
  classCode: string;
  className?: string;
  problems?: Problem[];
  users?: UserScore[];
}

export interface ProblemDifficultyMetadata {
  rowClassName: string;
  badgeClassName: string;
  dotClassName: string;
  label: string;
}
