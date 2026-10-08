import type { Location, Path } from 'react-router';
/** URL parameter names; each consumer defines how its value is interpreted. */
export const URL_PARAMS = {
  CLASS: 'class',
  TAB: 'tab',
  QUERY: 'q',
} as const;

/** Opens solved students for one problem while preserving the current URL context. */
export function getSolvedStudentsLocation(
  location: Pick<Location, 'pathname' | 'search' | 'hash'>,
  problemId: number
): Path {
  const params = new URLSearchParams(location.search);
  params.set(URL_PARAMS.TAB, 'standings');
  params.set(URL_PARAMS.QUERY, `solved:${problemId}`);
  return { pathname: location.pathname, search: `?${params}`, hash: location.hash };
}
