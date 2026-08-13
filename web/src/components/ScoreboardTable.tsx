import React, { useMemo } from 'react';
import { Problem, UserScore } from '@/domain/scoreboard/types';
import { getSolvedCount, groupProblemsByTopic } from '@/domain/scoreboard/utils';
import { SCOREBOARD_LABELS } from '@/domain/scoreboard/constants';

interface ScoreboardTableProps {
  problems: Problem[];
  users: UserScore[];
}

/**
 * ScoreboardTable Component
 * 
 * Renders the transposed matrix standings table:
 * - Columns: Student handles (Header Row 1) + Total AC summary counts (Header Row 2).
 * - Rows: Problem definitions grouped by section topics (e.g. "// TOPIC: ARRAYS & HASHING").
 * - Sticky Freeze: Columns 1 & 2 (Problem Name + Diff) and the double-row `thead` lock during scrolling.
 */
export function ScoreboardTable({ problems, users }: ScoreboardTableProps) {
  // 1. Topic Grouping: Preserves problem order while grouping by section topic banners.
  const topicGroups = useMemo(() => {
    return groupProblemsByTopic(problems);
  }, [problems]);

  // 2. Fast Status Lookup Optimization: Pre-computes Set<problemId> per student for O(1) cell checks.
  const userSolvedSets = useMemo(() => {
    return users.map((u) => new Set(u.solved || []));
  }, [users]);

  const totalCols = 2 + users.length;

  return (
    <main className="card-container">
      <div className="table-wrapper">
        <table className="scoreboard-table">
          <thead>
            {/* --- Header Row 1: Student Names, Rank Medals (👑, 🥈, 🥉), and Mastery Badges --- */}
            <tr id="table-header-row">
              <th className="col-problem-header" scope="col">{SCOREBOARD_LABELS.PROBLEM_HEADER}</th>
              <th className="col-diff-header" scope="col">{SCOREBOARD_LABELS.DIFF_HEADER}</th>
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

                return (
                  <th
                    key={u.userId || index}
                    className={`col-student-header ${rankClass}`}
                    scope="col"
                    title={u.userId ? `@${u.userId} (${getSolvedCount(u)} / ${problems.length} solved)` : ''}
                  >
                    <div className="student-header-inner">
                      {crownEmoji && <span className="student-crown">{crownEmoji}</span>}
                      <span>{u.displayName || u.userId}</span>
                      {isMastered && <span className="mastered-badge">{SCOREBOARD_LABELS.MASTERY_BADGE}</span>}
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
              <th className="col-diff-header total-ac-icon-cell" scope="col"></th>
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
              <tr>
                <td colSpan={totalCols} className="state-row">
                  {SCOREBOARD_LABELS.NO_PROBLEMS}
                </td>
              </tr>
            ) : (
              Array.from(topicGroups.entries()).map(([topicName, probList]) => (
                <React.Fragment key={topicName}>
                  {/* --- Full-Width Topic Section Header Banner --- */}
                  <tr className="topic-header-row">
                    <td colSpan={totalCols}>
                      {`${SCOREBOARD_LABELS.TOPIC_PREFIX} ${topicName.toUpperCase()}`}
                    </td>
                  </tr>

                  {/* --- Problem Matrix Rows --- */}
                  {probList.map((p) => {
                    // Normalize problem difficulty to assign color tint, dot, and ARIA label
                    const diffLower = (p.difficulty || '').toLowerCase();
                    let diffClass = 'problem-row-medium';
                    let badgeClass = 'badge-medium';
                    let diffDotClass = 'diff-medium';
                    let diffLabel: string = SCOREBOARD_LABELS.DIFFICULTY.MEDIUM;

                    if (diffLower.includes('easy')) {
                      diffClass = 'problem-row-easy';
                      badgeClass = 'badge-easy';
                      diffDotClass = 'diff-easy';
                      diffLabel = SCOREBOARD_LABELS.DIFFICULTY.EASY;
                    } else if (diffLower.includes('hard')) {
                      diffClass = 'problem-row-hard';
                      badgeClass = 'badge-hard';
                      diffDotClass = 'diff-hard';
                      diffLabel = SCOREBOARD_LABELS.DIFFICULTY.HARD;
                    } else if (diffLower.includes('not specified') || diffLower === '-' || !diffLower) {
                      diffClass = 'problem-row-unspecified';
                      badgeClass = 'badge-unspecified';
                      diffDotClass = 'diff-unspecified';
                      diffLabel = SCOREBOARD_LABELS.DIFFICULTY.UNSPECIFIED;
                    }

                    return (
                      <tr key={p.id} className={`problem-row ${diffClass}`}>
                        {/* Col 1: Problem Code Badge ([01], [02]) + Name */}
                        <td className="problem-title-cell">
                          <span className={`problem-code-badge ${badgeClass}`}>
                            {`[${String(p.id).padStart(2, '0')}]`}
                          </span>
                          {p.name || 'Problem'}
                        </td>

                        {/* Col 2: Difficulty Pure Color Dot Indicator */}
                        <td className="diff-cell">
                          <span
                            className={`diff-dot ${diffDotClass}`}
                            title={diffLabel}
                            aria-label={`Difficulty: ${diffLabel}`}
                          />
                        </td>

                        {/* Cols 3..N: Per-Student Solve Status (3D Glass Inset AC Pills) */}
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
