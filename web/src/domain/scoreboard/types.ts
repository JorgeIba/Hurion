export interface Problem {
  id: number;
  name: string;
  topic?: string;
  difficulty?: string;
}

export interface UserScore {
  userId: string;
  displayName?: string;
  solvedCount?: number;
  solved?: number[];
}

export interface ScoreboardData {
  classCode: string;
  className?: string;
  problems?: Problem[];
  users?: UserScore[];
}
