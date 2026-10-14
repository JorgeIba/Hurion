import { useMemo } from 'react';
import {
  Problem,
  UserScore,
  ProblemSolveStats,
  computeAllProblemSolveStats,
  filterProblemGroups,
  groupProblemsByTopic,
  CLASS_LABELS,
} from '@/domain/class';
import { ProblemToolbar, type ProblemViewControls } from '@/components/ProblemToolbar';
import { ProblemTopicSection } from '@/components/problems/ProblemTopicSection';

import { useTopicCollapse } from '@/hooks/useTopicCollapse';
import type { PriorityPresentationTable } from '@/presentation/priorityConfig';

export interface ProblemsViewProps extends ProblemViewControls {
  priorityPresentation: PriorityPresentationTable;
  problems: Problem[];
  users: UserScore[];
}

/**
 * ProblemsView Component (Orchestrator)
 *
 * Manages the state, filtering, and layout orchestration for the Problems Tab.
 */
export function ProblemsView({
  priorityPresentation,
  problems,
  users,
  options,
  onOptionsChange,
}: ProblemsViewProps) {
  const { difficultyFilter, priorityFilter, sortBy } = options;

  // Establish topic order before filtering out each topic's earliest problems.
  const topicGroups = useMemo(() => {
    return filterProblemGroups(
      groupProblemsByTopic(problems, sortBy),
      difficultyFilter,
      priorityFilter
    );
  }, [problems, difficultyFilter, priorityFilter, sortBy]);

  const showingCount = Array.from(topicGroups.values()).reduce(
    (count, entries) => count + entries.length,
    0
  );

  // 3. Batched single-pass solve statistics lookup map: O(U*S + P)
  const problemStatsMap: Map<number, ProblemSolveStats> = useMemo(() => {
    return computeAllProblemSolveStats(problems, users);
  }, [problems, users]);

  const allTopics = useMemo(() => Array.from(topicGroups.keys()), [topicGroups]);

  // 4. Shared Collapsible Topic Hook
  const { isCollapsed, toggleTopic, expandAll, collapseAll, isAllCollapsed } =
    useTopicCollapse();

  const allCollapsed = isAllCollapsed(allTopics);

  const handleToggleAll = () => {
    if (allCollapsed) {
      expandAll();
    } else {
      collapseAll(allTopics);
    }
  };

  return (
    <section
      className="card-container problems-view-container"
      role="tabpanel"
      id="tabpanel-problems"
      aria-labelledby="tab-problems"
    >
      {/* 1. Filter and Action Toolbar */}
      <ProblemToolbar
        priorityPresentation={priorityPresentation}
        options={options}
        onOptionsChange={onOptionsChange}
        isAllCollapsed={allCollapsed}
        onToggleAllTopics={handleToggleAll}
        showingCount={showingCount}
        totalCount={problems.length}
      />

      {/* 2. Scrollable Problem Groups Area */}
      <div className="problems-scroll-area">
        {problems.length === 0 ? (
          <div className="state-row">
            <p>{CLASS_LABELS.NO_PROBLEMS}</p>
          </div>
        ) : showingCount === 0 ? (
          <div className="state-row empty-search-state">
            <p>{CLASS_LABELS.PROBLEMS_VIEW.EMPTY_SEARCH}</p>
            <button
              type="button"
              className="btn-reset-filters"
              onClick={() => {
                onOptionsChange({
                  ...options,
                  difficultyFilter: 'all',
                  priorityFilter: 'all',
                });
              }}
            >
              {CLASS_LABELS.PROBLEMS_VIEW.CLEAR_SEARCH}
            </button>
          </div>
        ) : (
          Array.from(topicGroups.entries()).map(([topicName, topicProblems]) => (
            <ProblemTopicSection
              priorityPresentation={priorityPresentation}
              key={`topic-${topicName}`}
              topicName={topicName}
              problems={topicProblems}
              problemStatsMap={problemStatsMap}
              totalStudents={users.length}
              isCollapsed={isCollapsed(topicName)}
              onToggleCollapse={toggleTopic}
            />
          ))
        )}
      </div>
    </section>
  );
}
