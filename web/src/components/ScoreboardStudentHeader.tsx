import { CLASS_LABELS, getSolvedCount } from '@/domain/class';
import type { SortDirection, UserScore } from '@/domain/class';

interface ScoreboardStudentHeaderProps {
  user: UserScore;
  rankIndex: number;
  totalProblems: number;
  sortDirection: SortDirection;
  onSort: (userId: string) => void;
}

export function ScoreboardStudentHeader({
  user,
  rankIndex,
  totalProblems,
  sortDirection,
  onSort,
}: ScoreboardStudentHeaderProps) {
  let rankClass = '';
  let crownEmoji = '';
  if (rankIndex === 0) {
    rankClass = 'rank-1-header';
    crownEmoji = CLASS_LABELS.MEDALS.GOLD;
  } else if (rankIndex === 1) {
    rankClass = 'rank-2-header';
    crownEmoji = CLASS_LABELS.MEDALS.SILVER;
  } else if (rankIndex === 2) {
    rankClass = 'rank-3-header';
    crownEmoji = CLASS_LABELS.MEDALS.BRONZE;
  }

  const isMastered = totalProblems > 0 && getSolvedCount(user) >= totalProblems;

  const name = user.displayName || user.userId;
  const sortPresentation = {
    none: {
      className: '',
      ariaSort: 'none' as const,
      label: null,
      badgeClassName: '',
      title: `Click to sort scoreboard by ${name}'s solves`,
    },
    desc: {
      className: 'sorted-active-desc',
      ariaSort: 'descending' as const,
      label: CLASS_LABELS.SORT_LABELS.TOP_ACS,
      badgeClassName: 'sort-indicator-badge',
      title: `Sorted by ${name}'s Top Topics (Click for ${CLASS_LABELS.SORT_LABELS.ECHALE_GANAS})`,
    },
    asc: {
      className: 'sorted-active-asc',
      ariaSort: 'ascending' as const,
      label: CLASS_LABELS.SORT_LABELS.ECHALE_GANAS,
      badgeClassName: 'sort-indicator-badge sort-asc',
      title: `Sorted by ${name}'s ${CLASS_LABELS.SORT_LABELS.ECHALE_GANAS} (Click to Reset)`,
    },
  }[sortDirection ?? 'none'];

  return (
    <th
      className={`col-student-header ${rankClass} ${sortPresentation.className}`}
      scope="col"
      aria-sort={sortPresentation.ariaSort}
    >
      <button
        type="button"
        className="student-header-inner student-sort-button"
        onClick={() => onSort(user.userId)}
        title={sortPresentation.title}
      >
        {crownEmoji && <span className="student-crown">{crownEmoji}</span>}
        <span className="student-name-text">{name}</span>
        {isMastered && (
          <span className="mastered-badge">{CLASS_LABELS.MASTERY_BADGE}</span>
        )}
        {sortPresentation.label && (
          <span className={sortPresentation.badgeClassName}>
            {sortPresentation.label}
          </span>
        )}
      </button>
    </th>
  );
}
