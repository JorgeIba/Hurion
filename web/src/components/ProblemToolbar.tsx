import { useEffect, useId, useRef, useState, type ReactNode } from 'react';
import {
  DifficultyFilter,
  CLASS_LABELS,
  DEFAULT_PROBLEM_VIEW_OPTIONS,
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
  search?: ReactNode;
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
  search,
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
  const toggleDisabledReason =
    showingCount === 0 ? 'No topics to expand or collapse' : collapseDisabledReason;
  return (
    <div className="problems-toolbar">
      {search}
      <ProblemControls
        options={options}
        onOptionsChange={onOptionsChange}
        priorityPresentation={priorityPresentation}
        orderWithinSolveStatus={orderWithinSolveStatus}
      />

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
          <span className="toolbar-full-label">
            {isAllCollapsed ? 'Expand All ▾' : 'Collapse All ▴'}
          </span>
          <span className="toolbar-compact-label">
            {isAllCollapsed ? 'Expand' : 'Collapse'}
          </span>
        </button>

        <div
          className="problems-count-badge"
          role="status"
          aria-live="polite"
          aria-atomic="true"
        >
          <span className="toolbar-full-label">
            {CLASS_LABELS.PROBLEMS_VIEW.STATS_SHOWING}{' '}
            <span className="count-highlight">{showingCount}</span>{' '}
            {CLASS_LABELS.PROBLEMS_VIEW.STATS_OF} {totalCount}{' '}
            {CLASS_LABELS.PROBLEMS_VIEW.STATS_PROBLEMS}
          </span>
          <span className="toolbar-compact-label">
            <span aria-hidden="true">
              {showingCount}/{totalCount}
            </span>
            <span className="visually-hidden">
              {showingCount} of {totalCount} problems
            </span>
          </span>
        </div>
      </div>
    </div>
  );
}

interface ProblemControlsProps extends ProblemViewControls {
  priorityPresentation: PriorityPresentationTable;
  orderWithinSolveStatus: boolean;
}

/** Owns the controls' disclosure; CSS selects the inline or compact layout. */
function ProblemControls({
  options,
  onOptionsChange,
  priorityPresentation,
  orderWithinSolveStatus,
}: ProblemControlsProps) {
  const { difficultyFilter, priorityFilter, sortBy } = options;
  const [controlsOpen, setControlsOpen] = useState(false);
  const controlsRef = useRef<HTMLDivElement>(null);
  const controlsButtonRef = useRef<HTMLButtonElement>(null);
  const controlsId = useId();
  const activeCount =
    Number(difficultyFilter !== 'all') +
    Number(priorityFilter !== 'all') +
    Number(sortBy !== DEFAULT_PROBLEM_VIEW_OPTIONS.sortBy);

  useEffect(() => {
    if (!controlsOpen) return;
    const dismissOutside = (event: PointerEvent) => {
      if (event.target instanceof Node && !controlsRef.current?.contains(event.target)) {
        setControlsOpen(false);
      }
    };
    document.addEventListener('pointerdown', dismissOutside, true);
    return () => document.removeEventListener('pointerdown', dismissOutside, true);
  }, [controlsOpen]);
  return (
    <div
      className="problem-controls"
      ref={controlsRef}
      onBlur={(event) => {
        if (event.relatedTarget && !event.currentTarget.contains(event.relatedTarget))
          setControlsOpen(false);
      }}
      onKeyDown={(event) => {
        if (event.key === 'Escape' && controlsOpen) {
          event.preventDefault();
          setControlsOpen(false);
          controlsButtonRef.current?.focus();
        }
      }}
    >
      <button
        ref={controlsButtonRef}
        className="problem-controls-trigger"
        type="button"
        aria-label={
          activeCount > 0 ? `Controls, ${activeCount} active settings` : undefined
        }
        aria-expanded={controlsOpen}
        aria-controls={controlsId}
        onClick={() => setControlsOpen((open) => !open)}
      >
        Controls
        {activeCount > 0 && (
          <span className="problem-controls-count" aria-hidden="true">
            {activeCount}
          </span>
        )}
      </button>
      <div id={controlsId} className="problem-controls-panel" data-open={controlsOpen}>
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
            <option value="id-highest-first">ID · Highest first</option>
            <option value="id-lowest-first">ID · Lowest first</option>
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
      </div>
    </div>
  );
}
