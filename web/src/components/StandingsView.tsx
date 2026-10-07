import { useState } from 'react';
import type { Problem, UserScore } from '@/domain/class/types';
import type { PriorityPresentationTable } from '@/presentation/priorityConfig';
import { useScoreboardSearch } from '@/hooks/useScoreboardSearch';
import { ScoreboardSearchBar } from '@/components/ScoreboardSearchBar';
import { ScoreboardTable } from '@/components/ScoreboardTable';

interface StandingsViewProps {
  priorityPresentation: PriorityPresentationTable;
  problems: Problem[];
  users: UserScore[];
}

export function StandingsView({
  priorityPresentation,
  problems,
  users,
}: StandingsViewProps) {
  const [query, setQuery] = useState('');
  const { issues } = useScoreboardSearch(query, problems, users);
  return (
    <div className="standings-view">
      <ScoreboardSearchBar query={query} onQueryChange={setQuery} issues={issues} />
      <ScoreboardTable
        priorityPresentation={priorityPresentation}
        problems={problems}
        users={users}
      />
    </div>
  );
}
