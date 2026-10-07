import { useMemo } from 'react';
import type { Problem, UserScore } from '@/domain/class/types';
import { parseScoreboardSearch } from '@/domain/class/parser';
import { evaluateScoreboardFilter } from '@/domain/class/filters';

/** Resolves each query against full class data; incomplete prefixes do not filter. */
export function useScoreboardSearch(
  query: string,
  problems: readonly Problem[],
  users: readonly UserScore[]
) {
  return useMemo(() => {
    const { filters, issues } = parseScoreboardSearch(query);
    const isFiltering = filters.students !== undefined || filters.rows !== undefined;
    const selection = isFiltering
      ? evaluateScoreboardFilter({ problems, users, filters })
      : undefined;
    return { selection, issues, isFiltering };
  }, [query, problems, users]);
}
