import React, { startTransition, useCallback, useMemo, useState } from 'react';
import {
  CLASS_CONFIG,
  CLASS_LABELS,
  getNextSortState,
  getSolvedCount,
  filterProblemGroups,
  Problem,
  SortState,
  sortTopicGroupsByStudent,
  UserScore,
} from '@/domain/class';
import type { ScoreboardFilterResult } from '@/domain/class/filters';
import { useColumnResize } from '@/hooks/useColumnResize';
import { useTopicCollapse } from '@/hooks/useTopicCollapse';
import type { PriorityPresentationTable } from '@/presentation/priorityConfig';
import { ScoreboardProblemRow } from '@/components/ScoreboardProblemRow';
import { ScoreboardStudentHeader } from '@/components/ScoreboardStudentHeader';
import { ProblemToolbar, type ProblemViewControls } from '@/components/ProblemToolbar';

interface ScoreboardTableProps extends ProblemViewControls {
  priorityPresentation: PriorityPresentationTable;
  problems: Problem[];
  users: UserScore[];
  selection?: ScoreboardFilterResult;
}

/** Coordinates table selection, sorting, topic collapse and column sizing. */
export const ScoreboardTable = React.memo(function ScoreboardTable({
  priorityPresentation,
  problems,
  users,
  selection,
  options,
  onOptionsChange,
}: ScoreboardTableProps) {
  const {
    width: problemWidth,
    isResizing,
    handlePointerDown,
    handlePointerMove,
    handlePointerUp,
    handleDoubleClick,
    handleKeyDown,
  } = useColumnResize();

  const [sortState, setSortState] = useState<SortState>({
    userId: null,
    direction: null,
  });

  const handleStudentClick = useCallback((userId: string) => {
    startTransition(() => {
      setSortState((prev) => getNextSortState(prev, userId));
    });
  }, []);

  const { isCollapsed, toggleTopic, expandAll, collapseAll, isAllCollapsed } =
    useTopicCollapse();

  const topicGroups = useMemo(() => {
    return sortTopicGroupsByStudent(problems, users, sortState, options.sortBy);
  }, [problems, users, sortState, options.sortBy]);

  // Preserve each student's class rank and solve status when selecting columns.
  const columns = useMemo(() => {
    return users.map((user, rankIndex) => ({
      user,
      rankIndex,
      solved: new Set(user.solved || []),
    }));
  }, [users]);

  const visibleColumns = useMemo(
    () =>
      selection
        ? columns.filter(({ user }) => selection.userIds.has(user.userId))
        : columns,
    [columns, selection]
  );

  // Filter after sorting the full class so search does not redefine topic order.
  const visibleTopicGroups = useMemo(() => {
    return Array.from(
      filterProblemGroups(
        topicGroups,
        options.difficultyFilter,
        options.priorityFilter,
        selection?.problemIds
      )
    );
  }, [topicGroups, options.difficultyFilter, options.priorityFilter, selection]);
  const visibleProblemCount = visibleTopicGroups.reduce(
    (count, [, entries]) => count + entries.length,
    0
  );

  const sortedUser = sortState.direction
    ? users.find((user) => user.userId === sortState.userId)
    : undefined;
  const hiddenSortedUser =
    sortedUser && !visibleColumns.some(({ user }) => user.userId === sortedUser.userId)
      ? sortedUser
      : undefined;
  const labels = CLASS_LABELS.SCOREBOARD_VIEW;
  const expandTopics = selection !== undefined;
  const allTopics = visibleTopicGroups.map(([topic]) => topic);
  const allCollapsed = !expandTopics && isAllCollapsed(allTopics);
  const emptyMessage = getEmptyMessage({
    totalProblems: problems.length,
    totalStudents: users.length,
    visibleProblems: visibleProblemCount,
    visibleStudents: visibleColumns.length,
  });
  const totalCols = 1 + visibleColumns.length;
  const feedback = (
    <>
      {selection && (
        <p className="scoreboard-selection-summary" role="status">
          {labels.SHOWING} {visibleColumns.length} {labels.OF} {users.length}{' '}
          {labels.STUDENTS}
          {' · '}
          {visibleProblemCount} {labels.OF} {problems.length} {labels.PROBLEMS}
        </p>
      )}
      {hiddenSortedUser && (
        <div className="scoreboard-sort-summary">
          <span>
            {labels.SORTED_BY} {hiddenSortedUser.displayName || hiddenSortedUser.userId}
            {' · '}
            {sortState.direction === 'desc'
              ? CLASS_LABELS.SORT_LABELS.TOP_ACS
              : CLASS_LABELS.SORT_LABELS.ECHALE_GANAS}
          </span>
          <button
            type="button"
            className="btn-reset-filters"
            onClick={() => setSortState({ userId: null, direction: null })}
          >
            {labels.RESET_SORT}
          </button>
        </div>
      )}
    </>
  );

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
      <ProblemToolbar
        priorityPresentation={priorityPresentation}
        options={options}
        onOptionsChange={onOptionsChange}
        isAllCollapsed={allCollapsed}
        onToggleAllTopics={() =>
          allCollapsed ? expandAll(allTopics) : collapseAll(allTopics)
        }
        showingCount={visibleProblemCount}
        totalCount={problems.length}
        collapseDisabledReason={
          expandTopics ? labels.EXPANDED_WHILE_FILTERING : undefined
        }
        orderWithinSolveStatus={sortedUser !== undefined}
      />
      {feedback}
      {emptyMessage ? (
        <div className="state-row" role="status">
          <p>{emptyMessage}</p>
        </div>
      ) : (
        <div className="table-wrapper">
          <table className="scoreboard-table">
            <colgroup>
              <col
                className="col-problem-track"
                style={{ width: 'var(--col-problem-width)' }}
              />
              {visibleColumns.map(({ user }) => (
                <col
                  key={`col-student-${user.userId}`}
                  style={{
                    width: `${CLASS_CONFIG.DIMENSIONS.STUDENT_COL_WIDTH_PX}px`,
                  }}
                />
              ))}
            </colgroup>
            <thead>
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
                {visibleColumns.map(({ user, rankIndex }) => (
                  <ScoreboardStudentHeader
                    key={user.userId || rankIndex}
                    user={user}
                    rankIndex={rankIndex}
                    totalProblems={problems.length}
                    sortDirection={
                      sortState.userId === user.userId ? sortState.direction : null
                    }
                    onSort={handleStudentClick}
                  />
                ))}
              </tr>

              <tr id="table-sub-header-row" className="total-ac-row">
                <th className="col-problem-header total-ac-label-cell" scope="col">
                  {CLASS_LABELS.TOTAL_SOLVED}
                </th>
                {visibleColumns.map(({ user, rankIndex }) => (
                  <th
                    key={`ac-${user.userId || rankIndex}`}
                    className={`total-ac-count-cell ${rankIndex === 0 ? 'rank-1-ac' : ''}`}
                    scope="col"
                  >
                    {getSolvedCount(user)}
                  </th>
                ))}
              </tr>
            </thead>

            <tbody id="leaderboard">
              {visibleTopicGroups.map(([topicName, topicProblems]) => {
                // Search overrides presentation without overwriting saved collapse state.
                const collapsed = !expandTopics && isCollapsed(topicName);

                return (
                  <React.Fragment key={`topic-${topicName}`}>
                    <ScoreboardTopicRow
                      topicName={topicName}
                      problemCount={topicProblems.length}
                      colSpan={totalCols}
                      isCollapsed={collapsed}
                      onToggle={toggleTopic}
                      disableToggle={expandTopics}
                    />

                    {!collapsed &&
                      topicProblems.map((problem) => (
                        <ScoreboardProblemRow
                          key={`prob-${problem.id}`}
                          problem={problem}
                          columns={visibleColumns}
                          priorityPresentation={priorityPresentation}
                        />
                      ))}
                  </React.Fragment>
                );
              })}
            </tbody>
          </table>
        </div>
      )}
    </section>
  );
});

function getEmptyMessage({
  totalProblems,
  totalStudents,
  visibleProblems,
  visibleStudents,
}: {
  totalProblems: number;
  totalStudents: number;
  visibleProblems: number;
  visibleStudents: number;
}): string | null {
  const labels = CLASS_LABELS.SCOREBOARD_VIEW;
  if (totalProblems === 0) return CLASS_LABELS.NO_PROBLEMS;
  if (totalStudents === 0) return labels.NO_STUDENTS;
  if (visibleStudents === 0 && visibleProblems === 0) return labels.NO_MATCHES;
  if (visibleStudents === 0) return labels.NO_MATCHING_STUDENTS;
  if (visibleProblems === 0) return labels.NO_MATCHING_PROBLEMS;
  return null;
}

interface ScoreboardTopicRowProps {
  topicName: string;
  problemCount: number;
  colSpan: number;
  isCollapsed: boolean;
  onToggle: (topicName: string) => void;
  disableToggle: boolean;
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
    disableToggle,
  }: ScoreboardTopicRowProps) => {
    return (
      <tr className="topic-header-row">
        <td colSpan={colSpan}>
          <button
            type="button"
            disabled={disableToggle}
            className="topic-header-btn"
            onClick={() => onToggle(topicName)}
            aria-expanded={!isCollapsed}
            title={
              disableToggle
                ? CLASS_LABELS.SCOREBOARD_VIEW.EXPANDED_WHILE_FILTERING
                : isCollapsed
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
