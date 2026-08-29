import {
  DifficultyFilter,
  getProblemDifficultyMetadata,
  CLASS_LABELS,
} from '@/domain/class';

export interface ProblemsFilterToolbarProps {
  difficultyFilter: DifficultyFilter;
  onSelectDifficulty: (filter: DifficultyFilter) => void;
  hasMultipleTopics: boolean;
  isAllCollapsed: boolean;
  onToggleAllTopics: () => void;
  showingCount: number;
  totalCount: number;
}

const DIFFICULTY_BUTTONS: {
  id: DifficultyFilter;
  label: string;
  dotClassName?: string;
}[] = [
  { id: 'all', label: CLASS_LABELS.PROBLEMS_VIEW.FILTER_ALL },
  {
    id: 'easy',
    label: CLASS_LABELS.DIFFICULTY.EASY,
    dotClassName: getProblemDifficultyMetadata('easy').dotClassName,
  },
  {
    id: 'medium',
    label: CLASS_LABELS.DIFFICULTY.MEDIUM,
    dotClassName: getProblemDifficultyMetadata('medium').dotClassName,
  },
  {
    id: 'hard',
    label: CLASS_LABELS.DIFFICULTY.HARD,
    dotClassName: getProblemDifficultyMetadata('hard').dotClassName,
  },
];

/**
 * ProblemsFilterToolbar Component
 *
 * Renders the top filter and action toolbar for the Problems Catalog:
 * - Difficulty filter pill group (All, Easy, Medium, Hard)
 * - Global Expand All / Collapse All topics toggle
 * - Live problem count badge (Showing X of Y)
 */
export function ProblemsFilterToolbar({
  difficultyFilter,
  onSelectDifficulty,
  hasMultipleTopics,
  isAllCollapsed,
  onToggleAllTopics,
  showingCount,
  totalCount,
}: ProblemsFilterToolbarProps) {
  return (
    <div className="problems-toolbar">
      {/* Difficulty Filter Pills */}
      <div
        className="difficulty-filter-group"
        role="group"
        aria-label="Filter by difficulty"
      >
        {DIFFICULTY_BUTTONS.map((btn) => {
          const isSelected = difficultyFilter === btn.id;
          return (
            <button
              key={btn.id}
              type="button"
              className={`diff-filter-pill ${isSelected ? 'is-selected' : ''}`}
              onClick={() => onSelectDifficulty(btn.id)}
              aria-pressed={isSelected}
            >
              {btn.dotClassName && (
                <span className={`diff-dot ${btn.dotClassName}`} aria-hidden="true" />
              )}
              <span>{btn.label}</span>
            </button>
          );
        })}
      </div>

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

        <div className="problems-count-badge">
          {CLASS_LABELS.PROBLEMS_VIEW.STATS_SHOWING}{' '}
          <span className="count-highlight">{showingCount}</span>{' '}
          {CLASS_LABELS.PROBLEMS_VIEW.STATS_OF} {totalCount}{' '}
          {CLASS_LABELS.PROBLEMS_VIEW.STATS_PROBLEMS}
        </div>
      </div>
    </div>
  );
}
