import { Problem, ProblemSolveStats, CLASS_LABELS } from '@/domain/class';
import { ProblemRow } from '@/components/problems/ProblemRow';

export interface ProblemTopicSectionProps {
  topicName: string;
  problems: Problem[];
  problemStatsMap: Map<number, ProblemSolveStats>;
  totalStudents: number;
  isCollapsed: boolean;
  onToggleCollapse: (topicName: string) => void;
}

/**
 * ProblemTopicSection Component
 * 
 * Renders an accessible, collapsible section for problems sharing a topic:
 * - Semantic <h3> wrapper with WAI-ARIA 1.2 accordion header button
 * - Animated chevron toggle indicator
 * - Enclosed problem list container with row mappings
 */
export function ProblemTopicSection({
  topicName,
  problems,
  problemStatsMap,
  totalStudents,
  isCollapsed,
  onToggleCollapse,
}: ProblemTopicSectionProps) {
  const topicDomId = `topic-list-${topicName.replace(/[^a-zA-Z0-9_-]/g, '_')}`;

  return (
    <section className="problem-topic-section">
      {/* Semantic Heading & Interactive Collapsible Topic Header */}
      <h3 className="problem-topic-header-heading">
        <button
          type="button"
          className="problem-topic-header-btn"
          onClick={() => onToggleCollapse(topicName)}
          aria-expanded={!isCollapsed}
          aria-controls={topicDomId}
        >
          <div className="topic-header-left">
            <span className={`topic-chevron ${isCollapsed ? 'is-collapsed' : ''}`} aria-hidden="true">
              ▾
            </span>
            <span className="topic-title-code">
              {topicName.toUpperCase()}
            </span>
          </div>

          <div className="topic-header-meta">
            <span className="topic-prob-count">
              {problems.length} {CLASS_LABELS.PROBLEMS_VIEW.TOPIC_COUNT_SUFFIX}
            </span>
          </div>
        </button>
      </h3>

      {/* High-Density Problem List Rows */}
      <div id={topicDomId} className="problem-list" role="list" hidden={isCollapsed}>
        {problems.map((p) => (
          <ProblemRow
            key={`prob-row-${p.id}`}
            problem={p}
            stats={problemStatsMap.get(p.id)}
            totalStudents={totalStudents}
          />
        ))}
      </div>
    </section>
  );
}
