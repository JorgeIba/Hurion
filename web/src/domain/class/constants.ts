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
  SCOREBOARD_SEARCH: {
    DEBOUNCE_MS: 200,
  },
  API: {
    TIMEOUT_MS: 10_000,
    LOCAL_EXPORTS_PATH: '/exports',
    DEFAULT_AZURE_STORAGE_URL:
      'https://huronproblemcheckerjorge.blob.core.windows.net/exports',
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
  TOPIC_PREFIX: '//',
  NO_PROBLEMS: 'No problems recorded for this class',
  SCOREBOARD_SEARCH: {
    TRIGGER: 'Search',
    LABEL: 'Search standings',
    PLACEHOLDER: '@jorge t:grafos + #12',
    CLEAR: 'Clear search',
    HELP_LABEL: 'Search syntax help',
    HELP: 'Use @name for students, t:topic for topics, #12 for an exact problem ID, or a problem name. Add + for another alternative.',
    INVALID_ID: 'Invalid problem ID:',
    ID_HINT:
      'Use # followed by digits, for example #12. This alternative matches no problems until corrected.',
  },
  LOADING: 'Loading scoreboard data...',
  SCOREBOARD_VIEW: {
    NO_STUDENTS: 'No students recorded for this class',
    NO_MATCHES: 'No students or problems match the current filters.',
    NO_MATCHING_STUDENTS: 'No students match the current filters.',
    NO_MATCHING_PROBLEMS: 'No problems match the current filters.',
    SHOWING: 'Showing',
    OF: 'of',
    STUDENTS: 'students',
    PROBLEMS: 'problems',
    SORTED_BY: 'Sorted by',
    RESET_SORT: 'Reset sorting',
    EXPANDED_WHILE_FILTERING: 'Topics stay expanded while filtering',
  },
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
