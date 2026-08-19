import React, { useMemo, useState, useCallback, startTransition } from 'react';
import { Problem, UserScore, SortState } from '@/domain/scoreboard/types';
import { getSolvedCount, getProblemDifficultyMetadata } from '@/domain/scoreboard/utils';
import { getNextSortState, sortTopicGroupsByStudent } from '@/domain/scoreboard/sorting';
import { SCOREBOARD_LABELS, SCOREBOARD_CONFIG } from '@/domain/scoreboard/constants';
import { useColumnResize } from '@/hooks/useColumnResize';

interface ScoreboardTableProps {
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
 */
export function ScoreboardTable({ problems, users }: ScoreboardTableProps) {
  // 1. Column Resizing State (Pointer Events API)
  const {
    width: problemWidth,
    isResizing,
    handlePointerDown,
    handlePointerMove,
    handlePointerUp,
    handleDoubleClick,
  } = useColumnResize();

  // 2. Tri-State Student-Targeted Topic Sorting
  const [sortState, setSortState] = useState<SortState>({ userId: null, direction: null });

  const handleStudentClick = useCallback((userId: string) => {
    // React 19 Non-blocking transition keeps interaction responsive while re-ordering 200+ rows
    startTransition(() => {
      setSortState((prev) => getNextSortState(prev, userId));
    });
  }, []);

  // 3. Topic Grouping & Adaptive Sorting (Pure Domain Layer)
  const topicGroups = useMemo(() => {
    return sortTopicGroupsByStudent(problems, users, sortState);
  }, [problems, users, sortState]);

  // 4. Fast Status Lookup Optimization: Pre-computes Set<problemId> per student for O(1) cell checks.
  const userSolvedSets = useMemo(() => {
    return users.map((u) => new Set(u.solved || []));
  }, [users]);

  const totalCols = 1 + users.length;

  return (
    <main
      className="card-container"
      style={{
        ['--col-problem-width' as string]: `${problemWidth}px`,
      }}
    >
      <div className="table-wrapper">
        <table className="scoreboard-table">
          <colgroup>
            <col className="col-problem-track" style={{ width: 'var(--col-problem-width)' }} />
            {users.map((u, i) => (
              <col
                key={`col-student-${u.userId || i}`}
                style={{ width: `${SCOREBOARD_CONFIG.DIMENSIONS.STUDENT_COL_WIDTH_PX}px` }}
              />
            ))}
          </colgroup>
          <thead>
            {/* --- Header Row 1: Student Names, Rank Medals (👑, 🥈, 🥉), and Mastery Badges --- */}
            <tr id="table-header-row">
              <th className="col-problem-header" scope="col">
                <span>{SCOREBOARD_LABELS.PROBLEM_HEADER}</span>
                <div
                  className={`column-resizer ${isResizing ? 'is-resizing' : ''}`}
                  role="separator"
                  aria-orientation="vertical"
                  aria-valuenow={problemWidth}
                  aria-valuemin={SCOREBOARD_CONFIG.DIMENSIONS.PROBLEM_COL_MIN_WIDTH}
                  aria-valuemax={SCOREBOARD_CONFIG.DIMENSIONS.PROBLEM_COL_MAX_WIDTH}
                  onPointerDown={handlePointerDown}
                  onPointerMove={handlePointerMove}
                  onPointerUp={handlePointerUp}
                  onDoubleClick={handleDoubleClick}
                  title={SCOREBOARD_LABELS.RESIZER.TOOLTIP}
                  aria-label={SCOREBOARD_LABELS.RESIZER.ARIA_LABEL}
                />
              </th>
              {users.map((u, index) => {
                let rankClass = '';
                let crownEmoji = '';
                if (index === 0) {
                  rankClass = 'rank-1-header';
                  crownEmoji = SCOREBOARD_LABELS.MEDALS.GOLD;
                } else if (index === 1) {
                  rankClass = 'rank-2-header';
                  crownEmoji = SCOREBOARD_LABELS.MEDALS.SILVER;
                } else if (index === 2) {
                  rankClass = 'rank-3-header';
                  crownEmoji = SCOREBOARD_LABELS.MEDALS.BRONZE;
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
                          ? `Sorted by ${u.displayName || u.userId}'s Top Topics (Click for ${SCOREBOARD_LABELS.SORT_LABELS.ECHALE_GANAS})`
                          : `Sorted by ${u.displayName || u.userId}'s ${SCOREBOARD_LABELS.SORT_LABELS.ECHALE_GANAS} (Click to Reset)`
                        : `Click to sort scoreboard by ${u.displayName || u.userId}'s solves`
                    }
                  >
                    <div className="student-header-inner">
                      {crownEmoji && <span className="student-crown">{crownEmoji}</span>}
                      <span className="student-name-text">{u.displayName || u.userId}</span>
                      {isMastered && <span className="mastered-badge">{SCOREBOARD_LABELS.MASTERY_BADGE}</span>}
                      {sortDir === 'desc' && (
                        <span className="sort-indicator-badge">{SCOREBOARD_LABELS.SORT_LABELS.TOP_ACS}</span>
                      )}
                      {sortDir === 'asc' && (
                        <span className="sort-indicator-badge sort-asc">{SCOREBOARD_LABELS.SORT_LABELS.ECHALE_GANAS}</span>
                      )}
                    </div>
                  </th>
                );
              })}
            </tr>

            {/* --- Header Row 2: Total Solved (ACs) Sub-Header Row --- */}
            <tr id="table-sub-header-row" className="total-ac-row">
              <th className="col-problem-header total-ac-label-cell" scope="col">
                {SCOREBOARD_LABELS.TOTAL_SOLVED}
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
            {problems.length === 0 ? (
              <tr className="state-row">
                <td colSpan={totalCols}>{SCOREBOARD_LABELS.NO_PROBLEMS}</td>
              </tr>
            ) : (
              Array.from(topicGroups.entries()).map(([topicName, topicProblems]) => (
                <React.Fragment key={`topic-${topicName}`}>
                  {/* Topic Section Header Banner */}
                  <tr className="topic-header-row">
                    <td colSpan={totalCols}>
                      <div className="topic-header-sticky">
                        {SCOREBOARD_LABELS.TOPIC_PREFIX} {topicName.toUpperCase()}
                      </div>
                    </td>
                  </tr>

                  {/* Problem Rows inside this Topic */}
                  {topicProblems.map((p) => {
                    const {
                      rowClass,
                      badgeClass,
                      dotClass,
                      label: diffLabel,
                    } = getProblemDifficultyMetadata(p.difficulty);

                    const problemTitle = p.name || SCOREBOARD_CONFIG.DEFAULT_PROBLEM_NAME;

                    return (
                      <tr key={`prob-${p.id}`} className={`problem-row ${rowClass}`}>
                        {/* Col 1: Problem Title, Numerical ID, and Unified Difficulty Dot */}
                        <td className="problem-title-cell">
                          {p.url ? (
                            <a
                              href={p.url}
                              target="_blank"
                              rel="noopener noreferrer"
                              className="problem-link"
                              title={`Open ${problemTitle} in new tab`}
                            >
                              <span className={`problem-code-badge ${badgeClass}`}>
                                {`[${String(p.id).padStart(2, '0')}]`}
                              </span>
                              <span
                                className={`diff-dot ${dotClass}`}
                                title={`Difficulty: ${diffLabel}`}
                                aria-label={`Difficulty: ${diffLabel}`}
                              />
                              <span className="problem-name-text">
                                {problemTitle}
                              </span>
                              <span className="external-link-icon" aria-hidden="true">↗</span>
                            </a>
                          ) : (
                            <div className="problem-title-inner">
                              <span className={`problem-code-badge ${badgeClass}`}>
                                {`[${String(p.id).padStart(2, '0')}]`}
                              </span>
                              <span
                                className={`diff-dot ${dotClass}`}
                                title={`Difficulty: ${diffLabel}`}
                                aria-label={`Difficulty: ${diffLabel}`}
                              />
                              <span className="problem-name-text" title={problemTitle}>
                                {problemTitle}
                              </span>
                            </div>
                          )}
                        </td>

                        {/* Cols 2..N: Per-Student Solve Status (3D Glass Inset AC Pills) */}
                        {userSolvedSets.map((solvedSet, uIndex) => {
                          const isSolved = solvedSet.has(p.id);
                          return (
                            <td
                              key={`status-${p.id}-${users[uIndex]?.userId || uIndex}`}
                              className="status-cell"
                            >
                              {isSolved && <span className="status-ac-pill">✓</span>}
                            </td>
                          );
                        })}
                      </tr>
                    );
                  })}
                </React.Fragment>
              ))
            )}
          </tbody>
        </table>
      </div>
    </main>
  );
}
