import { useState, useCallback, useEffect } from 'react';
import { ActiveTab } from '@/domain/scoreboard/types';
import { sanitizeTab, sanitizeClassCode } from '@/domain/scoreboard/utils';
import { SCOREBOARD_CONFIG } from '@/domain/scoreboard/constants';

/**
 * Generic React Hook: useQueryParam
 * 
 * Synchronizes any typed parameter with the browser URL query string:
 * - Reads lazily from `window.location.search` on mount.
 * - Handles browser Back / Forward navigation via `popstate` event.
 * - Non-destructively updates the URL via `window.history.replaceState`.
 * - Automatically deletes the query parameter if reset to its default value.
 * 
 * @param key The URL query parameter key (e.g. 'class', 'tab')
 * @param parse Pure domain parser / sanitizer to validate raw string input
 * @param defaultValue Fallback value if parameter is missing or invalid
 * @returns Tuple containing `[value, setValue]`
 */
export function useQueryParam<T>(
  key: string,
  parse: (raw: string | null) => T,
  defaultValue: T
): [T, (newValue: T) => void] {
  const [value, setValueState] = useState<T>(() => {
    if (typeof window === 'undefined') return defaultValue;
    const raw = new URLSearchParams(window.location.search).get(key);
    return parse(raw);
  });

  // Two-way synchronization on browser Back / Forward navigation (popstate)
  useEffect(() => {
    const handlePopState = () => {
      const raw = new URLSearchParams(window.location.search).get(key);
      setValueState(parse(raw));
    };
    window.addEventListener('popstate', handlePopState);
    return () => window.removeEventListener('popstate', handlePopState);
  }, [key, parse]);

  // Update React state and mirror changes to the URL without polluting history
  const setValue = useCallback(
    (newValue: T) => {
      setValueState(newValue);
      if (typeof window === 'undefined') return;

      const url = new URL(window.location.href);
      if (newValue === defaultValue || !newValue) {
        url.searchParams.delete(key);
      } else {
        url.searchParams.set(key, String(newValue));
      }
      window.history.replaceState(null, '', url.toString());
    },
    [key, defaultValue]
  );

  return [value, setValue];
}

/**
 * Custom React Hook: useTabParam
 * Synchronizes the active scoreboard tab ('standings' | 'problems') with the `?tab=` URL parameter.
 */
export function useTabParam(
  defaultTab: ActiveTab = SCOREBOARD_CONFIG.DEFAULT_TAB
): [ActiveTab, (newTab: ActiveTab) => void] {
  return useQueryParam<ActiveTab>('tab', sanitizeTab, defaultTab);
}

/**
 * Custom React Hook: useClassCodeParam
 * Synchronizes the active class code (e.g. "basicos", "avanzados") with the `?class=` URL parameter.
 */
export function useClassCodeParam(
  defaultCode: string = SCOREBOARD_CONFIG.DEFAULT_CLASS_CODE
): [string, (newCode: string) => void] {
  return useQueryParam<string>(
    'class',
    (raw) => sanitizeClassCode(raw) || defaultCode,
    defaultCode
  );
}
