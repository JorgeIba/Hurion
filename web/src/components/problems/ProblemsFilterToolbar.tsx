import {
  DifficultyFilter,
  CLASS_LABELS,
  PriorityFilter,
  ProblemSort,
  PRIORITY_ORDER,
} from '@/domain/class';
import { getProblemPriorityLabel } from '@/presentation/priority';
import type { PriorityPresentationTable } from '@/presentation/priorityConfig';

export interface ProblemsFilterToolbarProps {
  priorityPresentation: PriorityPresentationTable;
  difficultyFilter: DifficultyFilter;
  onSelectDifficulty: (filter: DifficultyFilter) => void;
  priorityFilter: PriorityFilter;
  onSelectPriority: (filter: PriorityFilter) => void;
  sortBy: ProblemSort;
  onSelectSort: (sort: ProblemSort) => void;
  hasMultipleTopics: boolean;
  isAllCollapsed: boolean;
  onToggleAllTopics: () => void;
  showingCount: number;
  totalCount: number;
}

/**
 * ProblemsFilterToolbar Component
 *
 * Renders the top filter and action toolbar for the Problems Catalog:
 * - Difficulty and priority filters
 * - Global Expand All / Collapse All topics toggle
 * - Live problem count badge (Showing X of Y)
 */
export function ProblemsFilterToolbar({
  priorityPresentation,
  difficultyFilter,
  onSelectDifficulty,
  priorityFilter,
  onSelectPriority,
  sortBy,
  onSelectSort,
  hasMultipleTopics,
  isAllCollapsed,
  onToggleAllTopics,
  showingCount,
  totalCount,
}: ProblemsFilterToolbarProps) {
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
              onSelectDifficulty(event.target.value as DifficultyFilter)
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
            onChange={(event) => onSelectPriority(event.target.value as PriorityFilter)}
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
        <span className="problem-control-label">Order within topics</span>
        <select
          value={sortBy}
          onChange={(event) => onSelectSort(event.target.value as ProblemSort)}
        >
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
        {hasMultipleTopics && (
          <button
            type="button"
            className="btn-toggle-all-topics"
            onClick={onToggleAllTopics}
            title={isAllCollapsed ? 'Expand all topics' : 'Collapse all topics'}
          >
            {isAllCollapsed ? 'Expand All ▾' : 'Collapse All ▴'}
          </button>
        )}

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
