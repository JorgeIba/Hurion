import { useState, useMemo } from 'react';
import {
  Problem,
  UserScore,
  DifficultyFilter,
  PriorityFilter,
  ProblemSort,
  compareProblems,
  ProblemSolveStats,
  computeAllProblemSolveStats,
  filterProblems,
  groupProblemsByTopic,
  CLASS_LABELS,
} from '@/domain/class';
import { ProblemsFilterToolbar } from '@/components/problems/ProblemsFilterToolbar';
import { ProblemTopicSection } from '@/components/problems/ProblemTopicSection';

import { useTopicCollapse } from '@/hooks/useTopicCollapse';

export interface ProblemsViewProps {
  problems: Problem[];
  users: UserScore[];
}

/**
 * ProblemsView Component (Orchestrator)
 *
 * Manages the state, filtering, and layout orchestration for the Problems Tab.
 */
export function ProblemsView({ problems, users }: ProblemsViewProps) {
  const [difficultyFilter, setDifficultyFilter] = useState<DifficultyFilter>('all');
  const [priorityFilter, setPriorityFilter] = useState<PriorityFilter>('all');
  const [sortBy, setSortBy] = useState<ProblemSort>('difficulty-easy-first');

  // Establish topic order before filtering out each topic's earliest problems.
  const topicGroups = useMemo(() => {
    const groups = groupProblemsByTopic(problems);
    for (const [topic, entries] of groups) {
      const visibleProblems = filterProblems(
        entries,
        '',
        difficultyFilter,
        priorityFilter
      );
      if (visibleProblems.length === 0) {
        groups.delete(topic);
        continue;
      }
      groups.set(
        topic,
        visibleProblems.sort((a, b) => compareProblems(a, b, sortBy))
      );
    }
    return groups;
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
      <ProblemsFilterToolbar
        difficultyFilter={difficultyFilter}
        onSelectDifficulty={setDifficultyFilter}
        priorityFilter={priorityFilter}
        onSelectPriority={setPriorityFilter}
        sortBy={sortBy}
        onSelectSort={setSortBy}
        hasMultipleTopics={allTopics.length > 1}
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
                setDifficultyFilter('all');
                setPriorityFilter('all');
              }}
            >
              {CLASS_LABELS.PROBLEMS_VIEW.CLEAR_SEARCH}
            </button>
          </div>
        ) : (
          Array.from(topicGroups.entries()).map(([topicName, topicProblems]) => (
            <ProblemTopicSection
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
