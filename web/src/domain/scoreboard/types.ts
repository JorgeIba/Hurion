export type ProblemDifficulty = 'easy' | 'medium' | 'hard' | 'unspecified';

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

export type SortDirection = 'desc' | 'asc' | null;

export interface SortState {
  userId: string | null;
  direction: SortDirection;
}

export interface ScoreboardData {
  classCode: string;
  className?: string;
  problems?: Problem[];
  users?: UserScore[];
}
