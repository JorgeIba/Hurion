export type ProblemDifficulty = 'easy' | 'medium' | 'hard' | 'unspecified';
export type DifficultyFilter = 'all' | ProblemDifficulty;
export type ActiveTab = 'standings' | 'problems';

export interface Problem {
  id: number;
  name: string;
  url?: string;
  topic?: string;
  difficulty?: ProblemDifficulty | string;
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
