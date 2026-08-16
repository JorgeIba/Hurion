import { useState, useEffect } from 'react';
import axios from 'axios';
import { ScoreboardData } from '@/domain/scoreboard/types';
import { scoreboardApi } from '@/domain/scoreboard/api';

/**
 * Custom React Hook: useScoreboardData
 * 
 * Fetches class scoreboard data via the scoreboardApi domain service.
 * Manages loading states, error boundaries, and cancels pending network requests
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
    const controller = new AbortController();
    setLoading(true);
    setError(null);
    setData(null); // Reset previous class data to avoid stale UI state during transition/errors

    scoreboardApi
      .fetchClass(classCode, controller.signal)
      .then((payload) => {
        setData(payload);
        setLoading(false);
      })
      .catch((err: unknown) => {
        // Ignore deliberate abort signals triggered during unmount/class change
        if (!axios.isCancel(err) && !(err instanceof Error && err.name === 'AbortError')) {
          const message = err instanceof Error ? err.message : 'Error loading scoreboard';
          setError(message);
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
