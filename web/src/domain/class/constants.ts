/**
 * Single Source of Truth for Class Configuration & UI Constants.
 * Centralizes UI string constants, dimensions, API fallbacks, and labels
 * to eliminate scattered hardcoded values and prepare for internationalization (i18n).
 */

export const CLASS_CONFIG = {
  DEFAULT_CLASS_CODE: 'basicos',
  DEFAULT_TAB: 'standings',
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
    PROBLEM_COL_MOBILE_DEFAULT_WIDTH: 170,
    PROBLEM_COL_MOBILE_MIN_WIDTH: 130,
    STUDENT_COL_WIDTH_PX: 135,
  },
} as const;

export const CLASS_LABELS = {
  BRAND_LOGO: '📊',
  PROBLEM_HEADER: '(ID) Problem',
  DIFF_HEADER: 'Diff',
  TOTAL_SOLVED: 'Total Solved (ACs)',
  TOPIC_PREFIX: '// TOPIC:',
  NO_PROBLEMS: 'No problems recorded for this class',
  LOADING: 'Loading scoreboard data...',
  ERROR_FALLBACK: 'Failed to load scoreboard',
  ERRORS: {
    NOT_FOUND: 'Scoreboard not found for class:',
    NETWORK_ERROR: 'Unable to connect to server. Please check your connection.',
    SERVER_ERROR: 'Server error while loading scoreboard. Please try again later.',
    FALLBACK: 'Failed to load scoreboard data',
  },
  MASTERY_BADGE: '✨ 100%',
  TABS: {
    STANDINGS: 'Standings',
    PROBLEMS: 'Problems',
  },
  STATS: {
    STUDENTS_SUFFIX: 'Students',
    PROBLEMS_SUFFIX: 'Problems',
    LEADER_PREFIX: 'Leader:',
  },
  PROBLEMS_VIEW: {
    SEARCH_PLACEHOLDER: 'Search problems by name, ID, or topic...',
    FILTER_ALL: 'All Difficulties',
    EMPTY_SEARCH: 'No problems match your current search or filter.',
    CLEAR_SEARCH: 'Clear filters',
    OPEN_PROBLEM: 'Solve',
    CLASS_SOLVED_LABEL: 'solved',
    NO_URL: 'No link',
    TOPIC_COUNT_SUFFIX: 'problems',
    STATS_SHOWING: 'Showing',
    STATS_OF: 'of',
    STATS_PROBLEMS: 'problems',
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
