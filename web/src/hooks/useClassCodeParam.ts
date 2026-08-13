import { useState } from 'react';
import { sanitizeClassCode } from '@/domain/scoreboard/utils';

/**
 * Custom React Hook: useClassCodeParam
 * 
 * Manages the active class code for the scoreboard (e.g. "basicos" or "avanzados").
 * Reads `?class=...` from the browser URL on initial load and provides state to switch classes.
 * 
 * @param defaultCode Fallback class identifier if missing from URL (defaults to "basicos")
 * @returns Tuple containing `[classCode, setClassCode]`
 */
export function useClassCodeParam(defaultCode = 'basicos'): [string, (newCode: string) => void] {
  // Read initial class code from URL query parameter on load
  const params = new URLSearchParams(window.location.search);
  const rawCode = params.get('class');

  // Sanitize using unified domain helper function
  const sanitizedCode = sanitizeClassCode(rawCode);
  const initialCode = sanitizedCode || defaultCode;

  const [classCode, setClassCode] = useState<string>(initialCode);

  return [classCode, setClassCode];
}
