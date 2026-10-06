import React, { useMemo, useState, useCallback, startTransition } from 'react';
import {
  Problem,
  UserScore,
  SortState,
  getSolvedCount,
  getProblemDifficultyMetadata,
  getNextSortState,
  sortTopicGroupsByStudent,
  CLASS_LABELS,
  CLASS_CONFIG,
} from '@/domain/class';
import { useColumnResize } from '@/hooks/useColumnResize';
import { useTopicCollapse } from '@/hooks/useTopicCollapse';
import { PriorityLabel } from './problems/PriorityLabel';
import { getPriorityPresentation } from '@/presentation/priority';
import type { PriorityPresentationTable } from '@/presentation/priorityConfig';

interface ScoreboardTableProps {
  priorityPresentation: PriorityPresentationTable;
  problems: Problem[];
  users: UserScore[];
}

/**
 * ScoreboardTable Component
 *
 * Renders the transposed matrix standings table:
 * - Columns: Student handles (Header Row 1) + Total AC summary counts (Header Row 2).
 * - Rows: Problem definitions grouped by section topics (e.g. "// TOPIC: MAX_FLOW_MIN_COST").
 * - Sticky Freeze: 2D header locking and Column 1 frozen on horizontal scroll.
 * - Interactive Column Resizing: Via `useColumnResize` Pointer Events hook with drag handle.
 * - Tri-State Adaptive Sorting: Via React 19 `startTransition` non-blocking re-ordering.
 * - Collapsible Topics: Via `useTopicCollapse` hook and `ScoreboardTopicRow` component.
 */
export function ScoreboardTable({
  priorityPresentation,
  problems,
  users,
}: ScoreboardTableProps) {
  // 1. Column Resizing State (Pointer Events & Keyboard A11y)
  const {
    width: problemWidth,
    isResizing,
    handlePointerDown,
    handlePointerMove,
    handlePointerUp,
    handleDoubleClick,
    handleKeyDown,
  } = useColumnResize();

  // 2. Tri-State Student-Targeted Topic Sorting
  const [sortState, setSortState] = useState<SortState>({
    userId: null,
    direction: null,
  });

  const handleStudentClick = useCallback((userId: string) => {
    // React 19 Non-blocking transition keeps interaction responsive while re-ordering 200+ rows
    startTransition(() => {
      setSortState((prev) => getNextSortState(prev, userId));
    });
  }, []);

  // 3. Collapsible Topics State (Accordions in Standings table)
  const { isCollapsed, toggleTopic } = useTopicCollapse();

  // 3. Topic Grouping & Adaptive Sorting (Pure Domain Layer)
  const topicGroups = useMemo(() => {
    return sortTopicGroupsByStudent(problems, users, sortState);
  }, [problems, users, sortState]);

  // 4. Pre-computed Set lookups for O(1) status cell rendering
  const userSolvedSets = useMemo(() => {
    return users.map((u) => new Set(u.solved || []));
  }, [users]);

  // 5. Empty State Guard (Matches ProblemsView presentation)
  if (problems.length === 0) {
    return (
      <section
        className="card-container"
        role="tabpanel"
        id="tabpanel-standings"
        aria-labelledby="tab-standings"
      >
        <div className="state-row">
          <p>{CLASS_LABELS.NO_PROBLEMS}</p>
        </div>
      </section>
    );
  }

  const totalCols = 1 + users.length;

  return (
    <section
      className="card-container"
      role="tabpanel"
      id="tabpanel-standings"
      aria-labelledby="tab-standings"
      style={{
        ['--col-problem-width' as string]: `${problemWidth}px`,
      }}
    >
      <div className="table-wrapper">
        <table className="scoreboard-table">
          <colgroup>
            <col
              className="col-problem-track"
              style={{ width: 'var(--col-problem-width)' }}
            />
            {users.map((u, i) => (
              <col
                key={`col-student-${u.userId || i}`}
                style={{
                  width: `${CLASS_CONFIG.DIMENSIONS.STUDENT_COL_WIDTH_PX}px`,
                }}
              />
            ))}
          </colgroup>
          <thead>
            {/* --- Header Row 1: Student Names, Rank Medals (👑, 🥈, 🥉), and Mastery Badges --- */}
            <tr id="table-header-row">
              <th className="col-problem-header" scope="col">
                <span>{CLASS_LABELS.PROBLEM_HEADER}</span>
                <div
                  className={`column-resizer ${isResizing ? 'is-resizing' : ''}`}
                  role="separator"
                  tabIndex={0}
                  aria-orientation="vertical"
                  aria-valuenow={problemWidth}
                  aria-valuemin={CLASS_CONFIG.DIMENSIONS.PROBLEM_COL_MIN_WIDTH}
                  aria-valuemax={CLASS_CONFIG.DIMENSIONS.PROBLEM_COL_MAX_WIDTH}
                  onPointerDown={handlePointerDown}
                  onPointerMove={handlePointerMove}
                  onPointerUp={handlePointerUp}
                  onPointerCancel={handlePointerUp}
                  onLostPointerCapture={handlePointerUp}
                  onDoubleClick={handleDoubleClick}
                  onKeyDown={handleKeyDown}
                  title={CLASS_LABELS.RESIZER.TOOLTIP}
                  aria-label={CLASS_LABELS.RESIZER.ARIA_LABEL}
                />
              </th>
              {users.map((u, index) => {
                let rankClass = '';
                let crownEmoji = '';
                if (index === 0) {
                  rankClass = 'rank-1-header';
                  crownEmoji = CLASS_LABELS.MEDALS.GOLD;
                } else if (index === 1) {
                  rankClass = 'rank-2-header';
                  crownEmoji = CLASS_LABELS.MEDALS.SILVER;
                } else if (index === 2) {
                  rankClass = 'rank-3-header';
                  crownEmoji = CLASS_LABELS.MEDALS.BRONZE;
                }

                const isMastered =
                  problems.length > 0 && getSolvedCount(u) >= problems.length;

                const isSelected = sortState.userId === u.userId;
                const sortDir = isSelected ? sortState.direction : null;
                const sortClass = isSelected
                  ? sortDir === 'desc'
                    ? 'sorted-active-desc'
                    : 'sorted-active-asc'
                  : '';

                const ariaSortValue = isSelected
                  ? sortDir === 'desc'
                    ? 'descending'
                    : 'ascending'
                  : 'none';

                return (
                  <th
                    key={u.userId || index}
                    className={`col-student-header ${rankClass} ${sortClass}`}
                    scope="col"
                    aria-sort={ariaSortValue}
                    tabIndex={0}
                    onClick={() => handleStudentClick(u.userId)}
                    onKeyDown={(e) => {
                      if (e.key === 'Enter' || e.key === ' ') {
                        e.preventDefault();
                        handleStudentClick(u.userId);
                      }
                    }}
                    title={
                      isSelected
                        ? sortDir === 'desc'
                          ? `Sorted by ${u.displayName || u.userId}'s Top Topics (Click for ${CLASS_LABELS.SORT_LABELS.ECHALE_GANAS})`
                          : `Sorted by ${u.displayName || u.userId}'s ${CLASS_LABELS.SORT_LABELS.ECHALE_GANAS} (Click to Reset)`
                        : `Click to sort scoreboard by ${u.displayName || u.userId}'s solves`
                    }
                  >
                    <div className="student-header-inner">
                      {crownEmoji && <span className="student-crown">{crownEmoji}</span>}
                      <span className="student-name-text">
                        {u.displayName || u.userId}
                      </span>
                      {isMastered && (
                        <span className="mastered-badge">
                          {CLASS_LABELS.MASTERY_BADGE}
                        </span>
                      )}
                      {sortDir === 'desc' && (
                        <span className="sort-indicator-badge">
                          {CLASS_LABELS.SORT_LABELS.TOP_ACS}
                        </span>
                      )}
                      {sortDir === 'asc' && (
                        <span className="sort-indicator-badge sort-asc">
                          {CLASS_LABELS.SORT_LABELS.ECHALE_GANAS}
                        </span>
                      )}
                    </div>
                  </th>
                );
              })}
            </tr>

            {/* --- Header Row 2: Total Solved (ACs) Sub-Header Row --- */}
            <tr id="table-sub-header-row" className="total-ac-row">
              <th className="col-problem-header total-ac-label-cell" scope="col">
                {CLASS_LABELS.TOTAL_SOLVED}
              </th>
              {users.map((u, index) => (
                <th
                  key={`ac-${u.userId || index}`}
                  className={`total-ac-count-cell ${index === 0 ? 'rank-1-ac' : ''}`}
                  scope="col"
                >
                  {getSolvedCount(u)}
                </th>
              ))}
            </tr>
          </thead>

          <tbody id="leaderboard">
            {Array.from(topicGroups.entries()).map(([topicName, topicProblems]) => {
              const collapsed = isCollapsed(topicName);

              return (
                <React.Fragment key={`topic-${topicName}`}>
                  <ScoreboardTopicRow
                    topicName={topicName}
                    problemCount={topicProblems.length}
                    colSpan={totalCols}
                    isCollapsed={collapsed}
                    onToggle={toggleTopic}
                  />

                  {/* Problem Rows inside this Topic */}
                  {!collapsed &&
                    topicProblems.map((p) => {
                      const {
                        rowClassName,
                        badgeClassName,
                        dotClassName,
                        label: diffLabel,
                      } = getProblemDifficultyMetadata(p.difficulty);

                      const problemTitle = p.name || CLASS_CONFIG.DEFAULT_PROBLEM_NAME;
                      const cleanUrl = p.url?.trim();

                      return (
                        <tr
                          key={`prob-${p.id}`}
                          className={`problem-row ${rowClassName}`}
                        >
                          {/* Col 1: Problem Title, Numerical ID, and Unified Difficulty Dot */}
                          <th scope="row" className="problem-title-cell">
                            {cleanUrl ? (
                              <a
                                href={cleanUrl}
                                target="_blank"
                                rel="noopener noreferrer"
                                className="problem-link"
                                title={`Open ${problemTitle} in new tab`}
                              >
                                <span className={`problem-code-badge ${badgeClassName}`}>
                                  {`[${String(p.id).padStart(2, '0')}]`}
                                </span>
                                <span
                                  className={`diff-dot ${dotClassName}`}
                                  title={`Difficulty: ${diffLabel}`}
                                  aria-label={`Difficulty: ${diffLabel}`}
                                />
                                <span className="problem-name-text">{problemTitle}</span>
                                <span className="external-link-icon" aria-hidden="true">
                                  ↗
                                </span>
                              </a>
                            ) : (
                              <div className="problem-title-inner">
                                <span className={`problem-code-badge ${badgeClassName}`}>
                                  {`[${String(p.id).padStart(2, '0')}]`}
                                </span>
                                <span
                                  className={`diff-dot ${dotClassName}`}
                                  title={`Difficulty: ${diffLabel}`}
                                  aria-label={`Difficulty: ${diffLabel}`}
                                />
                                <span className="problem-name-text" title={problemTitle}>
                                  {problemTitle}
                                </span>
                              </div>
                            )}
                            <PriorityLabel
                              presentation={getPriorityPresentation(
                                p.priority,
                                priorityPresentation
                              )}
                            />
                          </th>

                          {/* Cols 2..N: Per-Student Solve Status (3D Glass Inset AC Pills) */}
                          {userSolvedSets.map((solvedSet, uIndex) => {
                            const isSolved = solvedSet.has(p.id);
                            return (
                              <td
                                key={`status-${p.id}-${users[uIndex]?.userId || uIndex}`}
                                className="status-cell"
                              >
                                {isSolved && (
                                  <span
                                    className="status-ac-pill"
                                    aria-label="Solved"
                                    title="Solved"
                                  >
                                    ✓
                                  </span>
                                )}
                              </td>
                            );
                          })}
                        </tr>
                      );
                    })}
                </React.Fragment>
              );
            })}
          </tbody>
        </table>
      </div>
    </section>
  );
}

interface ScoreboardTopicRowProps {
  topicName: string;
  problemCount: number;
  colSpan: number;
  isCollapsed: boolean;
  onToggle: (topicName: string) => void;
}

/**
 * ScoreboardTopicRow Sub-Component
 *
 * Renders the collapsible topic section banner inside the Scoreboard standings table.
 * Preserves semantic table row markup with an accessible button trigger.
 */
const ScoreboardTopicRow = React.memo(
  ({
    topicName,
    problemCount,
    colSpan,
    isCollapsed,
    onToggle,
  }: ScoreboardTopicRowProps) => {
    return (
      <tr className="topic-header-row">
        <td colSpan={colSpan}>
          <button
            type="button"
            className="topic-header-btn"
            onClick={() => onToggle(topicName)}
            aria-expanded={!isCollapsed}
            title={
              isCollapsed
                ? `Expand ${CLASS_LABELS.TOPIC_PREFIX} ${topicName}`
                : `Collapse ${CLASS_LABELS.TOPIC_PREFIX} ${topicName}`
            }
          >
            <div className="topic-header-sticky">
              <span
                className={`topic-chevron ${isCollapsed ? 'is-collapsed' : ''}`}
                aria-hidden="true"
              >
                ▾
              </span>
              <span className="topic-code-comment">
                {`${CLASS_LABELS.TOPIC_PREFIX} ${topicName}`}
              </span>
              <span className="topic-count-pill">{problemCount}</span>
            </div>
          </button>
        </td>
      </tr>
    );
  }
);

ScoreboardTopicRow.displayName = 'ScoreboardTopicRow';
