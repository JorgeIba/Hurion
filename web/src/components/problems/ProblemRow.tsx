import { memo } from 'react';
import {
  Problem,
  ProblemSolveStats,
  getProblemDifficultyMetadata,
  CLASS_CONFIG,
} from '@/domain/class';
import { PriorityLabel } from './PriorityLabel';
import { getPriorityPresentation } from '@/presentation/priority';
import type { PriorityPresentationTable } from '@/presentation/priorityConfig';

export interface ProblemRowProps {
  priorityPresentation: PriorityPresentationTable;
  problem: Problem;
  stats?: ProblemSolveStats;
  totalStudents: number;
}

/**
 * ProblemRow Component
 *
 * Renders a single high-density problem row:
 * - Col 1: Monospace problem ID badge with difficulty color theme ([#01] / [#128])
 * - Col 2: Problem title with external link icon
 * - Col 3: GPU-composited class solve progress meter
 */
export const ProblemRow = memo(function ProblemRow({
  priorityPresentation,
  problem,
  stats,
  totalStudents,
}: ProblemRowProps) {
  const { badgeClassName, label: diffLabel } = getProblemDifficultyMetadata(
    problem.difficulty
  );

  const solvedCount = stats?.solvedCount ?? 0;
  const effectiveTotalStudents = stats?.totalStudents ?? totalStudents;
  const solveRate = effectiveTotalStudents > 0 ? solvedCount / effectiveTotalStudents : 0;
  const solvePercent = Math.round(solveRate * 100);

  const problemTitle = problem.name || CLASS_CONFIG.DEFAULT_PROBLEM_NAME;
  const cleanUrl = problem.url?.trim();
  const formattedId = `[#${String(problem.id).padStart(2, '0')}]`;

  return (
    <div className="problem-list-row" role="listitem">
      {/* Col 1: Difficulty-Colorized Problem ID Badge */}
      <div className="problem-list-id">
        <span
          className={`problem-code-badge ${badgeClassName}`}
          title={`Difficulty: ${diffLabel}`}
          aria-label={`Difficulty: ${diffLabel}`}
        >
          {formattedId}
        </span>
      </div>

      <div className="problem-list-details">
        {cleanUrl ? (
          <a
            href={cleanUrl}
            target="_blank"
            rel="noopener noreferrer"
            className="problem-list-title-link"
            title={`Open ${problemTitle} in new tab`}
          >
            <span className="problem-list-title-text">{problemTitle}</span>
            <span className="external-link-icon" aria-hidden="true">
              ↗
            </span>
          </a>
        ) : (
          <div className="problem-list-title-plain">
            <span className="problem-list-title-text">{problemTitle}</span>
          </div>
        )}
        <PriorityLabel
          presentation={getPriorityPresentation(problem.priority, priorityPresentation)}
        />
      </div>

      {/* Col 3: Class Solves Micro-Progress Pill */}
      <div
        className="problem-list-solves-badge"
        role="meter"
        aria-valuenow={solvedCount}
        aria-valuemin={0}
        aria-valuemax={effectiveTotalStudents}
        aria-label={`${solvedCount} of ${effectiveTotalStudents} students solved`}
        title={`${solvedCount} of ${effectiveTotalStudents} students solved (${solvePercent}%)`}
      >
        <div
          className="solves-progress-fill"
          style={{ transform: `scaleX(${solveRate})` }}
        />
        <span className="solves-badge-text">
          <strong>{solvedCount}</strong>/{effectiveTotalStudents}
          <span className="solves-badge-percent"> ({solvePercent}%)</span>
        </span>
      </div>
    </div>
  );
});
