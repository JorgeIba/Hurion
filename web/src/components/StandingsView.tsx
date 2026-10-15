import { ScoreboardSearchBar } from '@/components/ScoreboardSearchBar';
import { ScoreboardTable } from '@/components/ScoreboardTable';
import type { ProblemViewControls } from '@/components/ProblemToolbar';
import type { Problem, UserScore } from '@/domain/class/types';
import { useScoreboardSearch } from '@/hooks/useScoreboardSearch';
import type { PriorityPresentationTable } from '@/presentation/priorityConfig';

export interface StandingsDataProps extends ProblemViewControls {
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
  options,
  onOptionsChange,
  search: { query, appliedQuery, changeQuery, submitQuery },
}: StandingsViewProps) {
  const { issues, selection } = useScoreboardSearch(appliedQuery, problems, users);

  return (
    <div className="standings-view">
      <ScoreboardTable
        search={
          <ScoreboardSearchBar
            query={query}
            onQueryChange={changeQuery}
            onQuerySubmit={submitQuery}
            issues={query === appliedQuery ? issues : []}
          />
        }
        priorityPresentation={priorityPresentation}
        problems={problems}
        users={users}
        selection={selection}
        options={options}
        onOptionsChange={onOptionsChange}
      />
    </div>
  );
}
