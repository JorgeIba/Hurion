import {
  DifficultyFilter,
  CLASS_LABELS,
  PriorityFilter,
  ProblemSort,
  ProblemViewOptions,
  PRIORITY_ORDER,
} from '@/domain/class';
import { getProblemPriorityLabel } from '@/presentation/priority';
import type { PriorityPresentationTable } from '@/presentation/priorityConfig';

export interface ProblemViewControls {
  options: ProblemViewOptions;
  onOptionsChange: (options: ProblemViewOptions) => void;
}

interface ProblemToolbarProps extends ProblemViewControls {
  priorityPresentation: PriorityPresentationTable;
  isAllCollapsed: boolean;
  onToggleAllTopics: () => void;
  showingCount: number;
  totalCount: number;
  collapseDisabledReason?: string;
  orderWithinSolveStatus?: boolean;
}

/**
 * Shared problem controls for the catalog and standings.
 */
export function ProblemToolbar({
  priorityPresentation,
  options,
  onOptionsChange,
  isAllCollapsed,
  onToggleAllTopics,
  showingCount,
  totalCount,
  collapseDisabledReason,
  orderWithinSolveStatus = false,
}: ProblemToolbarProps) {
  const { difficultyFilter, priorityFilter, sortBy } = options;
  const toggleDisabledReason =
    showingCount === 0 ? 'No topics to expand or collapse' : collapseDisabledReason;
  return (
    <div className="problems-toolbar">
      <div className="problem-filter-controls">
        <label
          className={`problem-select-control problem-filter-select ${difficultyFilter !== 'all' ? 'is-active' : ''}`}
        >
          <span className="problem-control-label">Difficulty</span>
          <select
            value={difficultyFilter}
            onChange={(event) =>
              onOptionsChange({
                ...options,
                difficultyFilter: event.target.value as DifficultyFilter,
              })
            }
          >
            <option value="all">All difficulties</option>
            <option value="easy">{CLASS_LABELS.DIFFICULTY.EASY}</option>
            <option value="medium">{CLASS_LABELS.DIFFICULTY.MEDIUM}</option>
            <option value="hard">{CLASS_LABELS.DIFFICULTY.HARD}</option>
          </select>
        </label>

        <label
          className={`problem-select-control problem-filter-select ${priorityFilter !== 'all' ? 'is-active' : ''}`}
        >
          <span className="problem-control-label">Priority</span>
          <select
            value={priorityFilter}
            onChange={(event) =>
              onOptionsChange({
                ...options,
                priorityFilter: event.target.value as PriorityFilter,
              })
            }
          >
            <option value="all">All priorities</option>
            {PRIORITY_ORDER.map((code) => (
              <option key={code ?? 'unset'} value={code ?? 'unset'}>
                {getProblemPriorityLabel(code, priorityPresentation)}
              </option>
            ))}
          </select>
        </label>
      </div>
      <label className="problem-select-control problem-order-select">
        <span className="problem-control-label">
          {orderWithinSolveStatus
            ? 'Order within solved/pending groups'
            : 'Order within topics'}
        </span>
        <select
          value={sortBy}
          onChange={(event) =>
            onOptionsChange({ ...options, sortBy: event.target.value as ProblemSort })
          }
        >
          <option value="id">ID · Lowest first</option>
          <option value="difficulty-easy-first">Difficulty · Easy first</option>
          <option value="difficulty-hard-first">Difficulty · Hard first</option>
          <option value="priority-essential-first">
            Priority · Most recommended first
          </option>
          <option value="priority-discouraged-first">
            Priority · Least recommended first
          </option>
        </select>
      </label>

      {/* Right Actions: Expand/Collapse All + Count Badge */}
      <div className="toolbar-right-actions">
        <button
          type="button"
          className="btn-toggle-all-topics"
          onClick={onToggleAllTopics}
          disabled={toggleDisabledReason !== undefined}
          title={
            toggleDisabledReason ??
            (isAllCollapsed ? 'Expand all topics' : 'Collapse all topics')
          }
        >
          {isAllCollapsed ? 'Expand All ▾' : 'Collapse All ▴'}
        </button>

        <div
          className="problems-count-badge"
          role="status"
          aria-live="polite"
          aria-atomic="true"
        >
          {CLASS_LABELS.PROBLEMS_VIEW.STATS_SHOWING}{' '}
          <span className="count-highlight">{showingCount}</span>{' '}
          {CLASS_LABELS.PROBLEMS_VIEW.STATS_OF} {totalCount}{' '}
          {CLASS_LABELS.PROBLEMS_VIEW.STATS_PROBLEMS}
        </div>
      </div>
    </div>
  );
}
