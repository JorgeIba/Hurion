/**
 * Single Source of Truth for Scoreboard Configuration & UI Constants.
 * Centralizes UI string constants, dimensions, API fallbacks, and labels
 * to eliminate scattered hardcoded values and prepare for internationalization (i18n).
 */

export const SCOREBOARD_CONFIG = {
  DEFAULT_CLASS_CODE: 'basicos',
  DEFAULT_TOPIC_NAME: 'General',
  DEFAULT_PROBLEM_NAME: 'Problem',
  DEFAULT_LEADER_NAME: '--',
  DEFAULT_PAGE_TITLE: 'Scoreboard',
  API: {
    TIMEOUT_MS: 10_000,
    LOCAL_EXPORTS_PATH: '/exports',
    DEFAULT_AZURE_STORAGE_URL: 'https://huronproblemcheckerjorge.blob.core.windows.net/exports',
  },
  DIMENSIONS: {
    PROBLEM_COL_DEFAULT_WIDTH: 280,
    PROBLEM_COL_MIN_WIDTH: 180,
    PROBLEM_COL_MAX_WIDTH: 700,
    STUDENT_COL_WIDTH_PX: 135,
  },
} as const;

export const SCOREBOARD_LABELS = {
  BRAND_LOGO: '📊',
  PROBLEM_HEADER: '(ID) Problem',
  DIFF_HEADER: 'Diff',
  TOTAL_SOLVED: 'Total Solved (ACs)',
  TOPIC_PREFIX: '// TOPIC:',
  NO_PROBLEMS: 'No problems recorded for this class',
  LOADING: 'Loading scoreboard data...',
  ERROR_FALLBACK: 'Failed to load scoreboard',
  MASTERY_BADGE: '✨ 100%',
  STATS: {
    STUDENTS_SUFFIX: 'Students',
    PROBLEMS_SUFFIX: 'Problems',
    LEADER_PREFIX: 'Leader:',
  },
  RESIZER: {
    TOOLTIP: 'Drag to resize column (double-click to reset)',
    ARIA_LABEL: 'Resize Problem Column',
  },
  SORT_LABELS: {
    TOP_ACS: '▲ Top ACs',
    ECHALE_GANAS: '▼ Echale Ganas',
  },
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
