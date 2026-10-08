import { ScoreboardSearchBar } from '@/components/ScoreboardSearchBar';
import { ScoreboardTable } from '@/components/ScoreboardTable';
import type { Problem, UserScore } from '@/domain/class/types';
import { useScoreboardSearch } from '@/hooks/useScoreboardSearch';
import type { PriorityPresentationTable } from '@/presentation/priorityConfig';

export interface StandingsDataProps {
  priorityPresentation: PriorityPresentationTable;
  problems: Problem[];
  users: UserScore[];
}

interface StandingsViewProps extends StandingsDataProps {
  search: {
    query: string;
    appliedQuery: string;
    changeQuery: (value: string, isComposing?: boolean) => void;
    submitQuery: () => void;
  };
}

export function StandingsView({
  priorityPresentation,
  problems,
  users,
  search: { query, appliedQuery, changeQuery, submitQuery },
}: StandingsViewProps) {
  const { issues, selection } = useScoreboardSearch(appliedQuery, problems, users);

  return (
    <div className="standings-view">
      <ScoreboardSearchBar
        query={query}
        onQueryChange={changeQuery}
        onQuerySubmit={submitQuery}
        issues={query === appliedQuery ? issues : []}
      />
      <ScoreboardTable
        priorityPresentation={priorityPresentation}
        problems={problems}
        users={users}
        selection={selection}
      />
    </div>
  );
}
