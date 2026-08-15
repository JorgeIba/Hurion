import { useState, useEffect } from 'react';
import { ScoreboardData } from '@/domain/scoreboard/types';

/**
 * Custom React Hook: useScoreboardData
 * 
 * Fetches class scoreboard JSON data from the Go backend endpoint (`/exports/{classCode}_score.json`).
 * Manages loading states, HTTP 404/network error boundaries, and cancels pending network requests
 * via AbortController when the component unmounts or classCode changes.
 * 
 * @param classCode Unique identifier string for the class (e.g. "basicos")
 * @returns Object containing `data`, `loading`, and `error`
 */
export function useScoreboardData(classCode: string) {
  const [data, setData] = useState<ScoreboardData | null>(null);
  const [loading, setLoading] = useState<boolean>(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    // AbortController cancels network requests at the browser layer on unmount/re-fetch
    const controller = new AbortController();
    setLoading(true);
    setError(null);
    setData(null); // Reset previous class data to avoid stale UI state during transition/errors

    // Fetch JSON export file (falls back to local /exports if VITE_EXPORTS_BASE_URL is unset)
    const exportsBaseUrl = (import.meta.env.VITE_EXPORTS_BASE_URL || '/exports').replace(/\/+$/, '');
    fetch(`${exportsBaseUrl}/${encodeURIComponent(classCode)}_score.json`, {
      signal: controller.signal,
    })
      .then((res) => {
        if (!res.ok) {
          throw new Error(`Scoreboard for class "${classCode}" not found (${res.status})`);
        }
        return res.json();
      })
      .then((payload: ScoreboardData) => {
        if (!controller.signal.aborted) {
          setData(payload);
          setLoading(false);
        }
      })
      .catch((err: Error) => {
        // Ignore deliberate abort signals triggered during unmount/class change
        if (err.name !== 'AbortError') {
          setError(err.message);
          setData(null);
          setLoading(false);
        }
      });

    // Cleanup: cancel pending HTTP request instantly if classCode changes or component unmounts
    return () => {
      controller.abort();
    };
  }, [classCode]);

  return { data, loading, error };
}
