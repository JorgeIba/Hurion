/**
 * Single Source of Truth constants for Scoreboard UI Table Labels.
 * Centralizes UI string constants to prevent hardcoded string repetition across components
 * and prepare the application for future internationalization (i18n).
 */
export const SCOREBOARD_LABELS = {
  PROBLEM_HEADER: '(ID) Problem',
  DIFF_HEADER: 'Diff',
  TOTAL_SOLVED: 'Total Solved (ACs)',
  TOPIC_PREFIX: '// TOPIC:',
  NO_PROBLEMS: 'No problems recorded for this class',
  MASTERY_BADGE: '✨ 100%',
  DIFFICULTY: {
    EASY: 'Easy',
    MEDIUM: 'Medium',
    HARD: 'Hard',
    UNSPECIFIED: 'Not specified',
  },
  MEDALS: {
    GOLD: '👑',
    SILVER: '🥈',
    BRONZE: '🥉',
  },
} as const;
