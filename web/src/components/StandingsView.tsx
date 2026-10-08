import { useState } from 'react';
import { useDebouncedCallback } from 'use-debounce';
import { ScoreboardSearchBar } from '@/components/ScoreboardSearchBar';
import { ScoreboardTable } from '@/components/ScoreboardTable';
import { CLASS_CONFIG } from '@/domain/class/constants';
import type { Problem, UserScore } from '@/domain/class/types';
import { useScoreboardSearch } from '@/hooks/useScoreboardSearch';
import type { PriorityPresentationTable } from '@/presentation/priorityConfig';

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
  const [appliedQuery, setAppliedQuery] = useState('');
  const applyQuery = useDebouncedCallback(
    setAppliedQuery,
    CLASS_CONFIG.SCOREBOARD_SEARCH.DEBOUNCE_MS
  );
  const { issues } = useScoreboardSearch(appliedQuery, problems, users);

  const changeQuery = (value: string, isComposing = false) => {
    setQuery(value);
    if (value === '' || isComposing) {
      applyQuery.cancel();
      if (value === '') setAppliedQuery('');
    } else {
      applyQuery(value);
    }
  };
  const submitQuery = () => {
    applyQuery.cancel();
    setAppliedQuery(query);
  };

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
      />
    </div>
  );
}
