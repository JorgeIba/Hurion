import { useState, useEffect } from 'react';
import axios from 'axios';
import { ClassData, classApi, CLASS_LABELS } from '@/domain/class';

/**
 * Custom React Hook: useScoreboardData
 * 
 * Fetches class scoreboard data via the classApi domain service.
 * Manages loading states, error boundaries, and cancels pending network requests
 * via AbortController when the component unmounts or classCode changes.
 * 
 * @param classCode Unique identifier string for the class (e.g. "basicos")
 * @returns Object containing `data`, `loading`, and `error`
 */
export function useScoreboardData(classCode: string) {
  const [data, setData] = useState<ClassData | null>(null);
  const [loading, setLoading] = useState<boolean>(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const controller = new AbortController();
    setLoading(true);
    setError(null);
    setData(null); // Reset previous class data to avoid stale UI state during transition/errors

    classApi
      .fetchClass(classCode, controller.signal)
      .then((payload) => {
        setData(payload);
        setLoading(false);
      })
      .catch((err: unknown) => {
        // Ignore deliberate abort signals triggered during unmount/class change
        if (!axios.isCancel(err) && !(err instanceof Error && err.name === 'AbortError')) {
          let message: string = CLASS_LABELS.ERRORS.FALLBACK;

          if (axios.isAxiosError(err)) {
            if (err.response?.status === 404) {
              message = `${CLASS_LABELS.ERRORS.NOT_FOUND} "${classCode}"`;
            } else if (!err.response) {
              message = CLASS_LABELS.ERRORS.NETWORK_ERROR;
            } else if (err.response.status >= 500) {
              message = CLASS_LABELS.ERRORS.SERVER_ERROR;
            } else {
              message = CLASS_LABELS.ERRORS.FALLBACK;
            }
          } else if (err instanceof Error) {
            message = err.message;
          }

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
