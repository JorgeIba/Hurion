import { CLASS_CONFIG, getProblemDifficultyMetadata } from '@/domain/class';
import type { Problem, UserScore } from '@/domain/class';
import { getPriorityPresentation } from '@/presentation/priority';
import type { PriorityPresentationTable } from '@/presentation/priorityConfig';
import { PriorityLabel } from '@/components/problems/PriorityLabel';

interface ScoreboardProblemRowProps {
  problem: Problem;
  columns: readonly { user: UserScore; solved: ReadonlySet<number> }[];
  priorityPresentation: PriorityPresentationTable;
}

export function ScoreboardProblemRow({
  problem,
  columns,
  priorityPresentation,
}: ScoreboardProblemRowProps) {
  const {
    rowClassName,
    badgeClassName,
    dotClassName,
    label: diffLabel,
  } = getProblemDifficultyMetadata(problem.difficulty);

  const problemTitle = problem.name || CLASS_CONFIG.DEFAULT_PROBLEM_NAME;
  const cleanUrl = problem.url?.trim();

  const identity = (
    <>
      <span className={`problem-code-badge ${badgeClassName}`}>
        {`[${String(problem.id).padStart(2, '0')}]`}
      </span>
      <span
        className={`diff-dot ${dotClassName}`}
        title={`Difficulty: ${diffLabel}`}
        aria-label={`Difficulty: ${diffLabel}`}
      />
      <span className="problem-name-text" title={cleanUrl ? undefined : problemTitle}>
        {problemTitle}
      </span>
    </>
  );

  return (
    <tr className={`problem-row ${rowClassName}`}>
      <th scope="row" className="problem-title-cell">
        {cleanUrl ? (
          <a
            href={cleanUrl}
            target="_blank"
            rel="noopener noreferrer"
            className="problem-link"
            title={`Open ${problemTitle} in new tab`}
          >
            {identity}
            <span className="external-link-icon" aria-hidden="true">
              ↗
            </span>
          </a>
        ) : (
          <div className="problem-title-inner">{identity}</div>
        )}
        <PriorityLabel
          presentation={getPriorityPresentation(problem.priority, priorityPresentation)}
        />
      </th>

      {columns.map(({ user, solved }) => {
        const isSolved = solved.has(problem.id);
        return (
          <td
            key={`status-${problem.id}-${user.userId}`}
            className="status-cell"
            aria-label={isSolved ? 'Solved' : 'Not solved'}
          >
            {isSolved && (
              <span className="status-ac-pill" title="Solved">
                ✓
              </span>
            )}
            {!isSolved && (
              <span className="status-not-solved" aria-hidden="true">
                —
              </span>
            )}
          </td>
        );
      })}
    </tr>
  );
}
