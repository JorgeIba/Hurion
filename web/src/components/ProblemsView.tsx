import { useState, useMemo } from 'react';
import { Problem, UserScore, DifficultyFilter, ProblemSolveStats } from '@/domain/scoreboard/types';
import {
  computeAllProblemSolveStats,
  filterProblems,
  groupProblemsByTopic,
} from '@/domain/scoreboard/utils';
import { SCOREBOARD_LABELS } from '@/domain/scoreboard/constants';
import { ProblemsFilterToolbar } from '@/components/problems/ProblemsFilterToolbar';
import { ProblemTopicSection } from '@/components/problems/ProblemTopicSection';

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
  const [collapsedTopics, setCollapsedTopics] = useState<Set<string>>(new Set());

  // 1. Filter problems by active difficulty selection
  const filteredProblems = useMemo(() => {
    return filterProblems(problems, '', difficultyFilter);
  }, [problems, difficultyFilter]);

  // 2. Group filtered problems by their topic categories
  const topicGroups = useMemo(() => {
    return groupProblemsByTopic(filteredProblems);
  }, [filteredProblems]);

  // 3. Batched single-pass solve statistics lookup map: O(U*S + P)
  const problemStatsMap: Map<number, ProblemSolveStats> = useMemo(() => {
    return computeAllProblemSolveStats(problems, users);
  }, [problems, users]);

  const allTopics = Array.from(topicGroups.keys());
  const isAllCollapsed = allTopics.length > 0 && allTopics.every((t) => collapsedTopics.has(t));

  const toggleTopicCollapse = (topicName: string) => {
    setCollapsedTopics((prev) => {
      const next = new Set(prev);
      if (next.has(topicName)) {
        next.delete(topicName);
      } else {
        next.add(topicName);
      }
      return next;
    });
  };

  const toggleAllTopics = () => {
    setCollapsedTopics((prev) => {
      if (isAllCollapsed) {
        const next = new Set(prev);
        allTopics.forEach((t) => next.delete(t));
        return next;
      }
      return new Set([...prev, ...allTopics]);
    });
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
        hasMultipleTopics={allTopics.length > 1}
        isAllCollapsed={isAllCollapsed}
        onToggleAllTopics={toggleAllTopics}
        showingCount={filteredProblems.length}
        totalCount={problems.length}
      />

      {/* 2. Scrollable Problem Groups Area */}
      <div className="problems-scroll-area">
        {problems.length === 0 ? (
          <div className="state-row">
            <p>{SCOREBOARD_LABELS.NO_PROBLEMS}</p>
          </div>
        ) : filteredProblems.length === 0 ? (
          <div className="state-row empty-search-state">
            <p>{SCOREBOARD_LABELS.PROBLEMS_VIEW.EMPTY_SEARCH}</p>
            <button
              type="button"
              className="btn-reset-filters"
              onClick={() => setDifficultyFilter('all')}
            >
              {SCOREBOARD_LABELS.PROBLEMS_VIEW.CLEAR_SEARCH}
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
              isCollapsed={collapsedTopics.has(topicName)}
              onToggleCollapse={toggleTopicCollapse}
            />
          ))
        )}
      </div>
    </section>
  );
}
