import { useEffect, useId, useLayoutEffect, useRef, useState } from 'react';
import { useLocation, useNavigationType } from 'react-router';
import { useDebouncedCallback } from 'use-debounce';
import { CLASS_CONFIG } from '@/domain/class/constants';
import { URL_PARAMS } from '@/app/routing/navigation';
import { useQueryParam } from '@/app/routing/useQueryParam';

interface PendingSearch {
  value: string;
  navigationVersion: number;
}

/** The URL owns applied search; the input keeps a draft until its debounce expires. */
export function useScoreboardSearchQuery() {
  const location = useLocation();
  const navigationType = useNavigationType();
  const [appliedQuery, writeQuery] = useQueryParam(
    URL_PARAMS.QUERY,
    (raw) => raw ?? '',
    ''
  );

  const [query, setQuery] = useState(appliedQuery);
  const navigationVersion = useRef(0);
  const ownWrite = useRef<string | null>(null);
  const ownerId = useId();
  const writeCount = useRef(0);

  const commitQuery = (request: PendingSearch) => {
    // Never let work scheduled on an earlier page overwrite a navigation.
    if (request.navigationVersion !== navigationVersion.current) return;
    const params = new URLSearchParams(location.search);
    const raw = params.get(URL_PARAMS.QUERY);
    if (request.value === (raw ?? '') && !(request.value === '' && raw !== null)) return;

    const token = `${ownerId}-${++writeCount.current}`;
    ownWrite.current = token;
    writeQuery(request.value, {
      replace: true,
      state: { ...location.state, scoreboardSearchWrite: token },
    });
  };

  const applyQuery = useDebouncedCallback(
    commitQuery,
    CLASS_CONFIG.SCOREBOARD_SEARCH.DEBOUNCE_MS
  );

  useLayoutEffect(() => {
    const confirmation = ownWrite.current;
    ownWrite.current = null;
    if (
      confirmation &&
      navigationType === 'REPLACE' &&
      confirmation === location.state?.scoreboardSearchWrite
    ) {
      // Our own update preserves the draft and any newer scheduled search.
      return;
    }
    // External navigation invalidates all work from the previous page, even if q is equal.
    navigationVersion.current += 1;
    applyQuery.cancel();
    setQuery(appliedQuery);
  }, [location, navigationType, appliedQuery, applyQuery]);

  useEffect(() => () => applyQuery.cancel(), [applyQuery]);

  const changeQuery = (value: string, isComposing = false) => {
    setQuery(value);
    applyQuery.cancel();
    if (isComposing) return;
    const request = { value, navigationVersion: navigationVersion.current };
    if (value === '') commitQuery(request);
    else applyQuery(request);
  };
  const submitQuery = () => {
    applyQuery.cancel();
    commitQuery({ value: query, navigationVersion: navigationVersion.current });
  };

  return { query, appliedQuery, changeQuery, submitQuery };
}
