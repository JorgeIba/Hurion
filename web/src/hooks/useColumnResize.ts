import { useState, useCallback, useRef } from 'react';
import { SCOREBOARD_CONFIG } from '@/domain/scoreboard/constants';

interface ColumnResizeOptions {
  defaultWidth?: number;
  minWidth?: number;
  maxWidth?: number;
}

/**
 * Resolves default and minimum column widths based on current viewport size.
 * Allows mobile screens (<= 640px) to start with a compact layout while
 * preserving user drag interactivity.
 */
function getResponsiveDefaults(): { defaultWidth: number; minWidth: number } {
  if (typeof window !== 'undefined' && window.innerWidth <= 640) {
    return {
      defaultWidth: SCOREBOARD_CONFIG.DIMENSIONS.PROBLEM_COL_MOBILE_DEFAULT_WIDTH,
      minWidth: SCOREBOARD_CONFIG.DIMENSIONS.PROBLEM_COL_MOBILE_MIN_WIDTH,
    };
  }
  return {
    defaultWidth: SCOREBOARD_CONFIG.DIMENSIONS.PROBLEM_COL_DEFAULT_WIDTH,
    minWidth: SCOREBOARD_CONFIG.DIMENSIONS.PROBLEM_COL_MIN_WIDTH,
  };
}

/**
 * Custom React Hook: useColumnResize
 * 
 * Encapsulates pointer-based drag resizing using the standard W3C Pointer Events API.
 * Uses `setPointerCapture` to seamlessly track mouse, touch (mobile/iPad), and stylus inputs
 * without needing global `window` event listeners or risking memory leaks.
 */
export function useColumnResize({
  defaultWidth,
  minWidth,
  maxWidth = SCOREBOARD_CONFIG.DIMENSIONS.PROBLEM_COL_MAX_WIDTH,
}: ColumnResizeOptions = {}) {
  const responsiveDefaults = getResponsiveDefaults();
  const resolvedDefaultWidth = defaultWidth ?? responsiveDefaults.defaultWidth;
  const resolvedMinWidth = minWidth ?? responsiveDefaults.minWidth;

  const [width, setWidth] = useState<number>(resolvedDefaultWidth);
  const [isResizing, setIsResizing] = useState<boolean>(false);
  const resizeStartRef = useRef<{ startX: number; startWidth: number } | null>(null);

  const handlePointerDown = useCallback(
    (e: React.PointerEvent<HTMLDivElement>) => {
      // Respond to primary pointer click/touch contact
      if (e.button !== 0 && e.pointerType === 'mouse') return;

      e.preventDefault();
      e.currentTarget.setPointerCapture(e.pointerId);
      setIsResizing(true);
      resizeStartRef.current = { startX: e.clientX, startWidth: width };
    },
    [width]
  );

  const handlePointerMove = useCallback(
    (e: React.PointerEvent<HTMLDivElement>) => {
      if (!isResizing || !resizeStartRef.current) return;

      const delta = e.clientX - resizeStartRef.current.startX;
      const newWidth = Math.max(
        resolvedMinWidth,
        Math.min(maxWidth, resizeStartRef.current.startWidth + delta)
      );
      setWidth(newWidth);
    },
    [isResizing, resolvedMinWidth, maxWidth]
  );

  const handlePointerUp = useCallback(
    (e: React.PointerEvent<HTMLDivElement>) => {
      if (!isResizing) return;

      if (e.currentTarget.hasPointerCapture(e.pointerId)) {
        e.currentTarget.releasePointerCapture(e.pointerId);
      }
      setIsResizing(false);
      resizeStartRef.current = null;
    },
    [isResizing]
  );

  const handleDoubleClick = useCallback(() => {
    setWidth(resolvedDefaultWidth);
  }, [resolvedDefaultWidth]);

  return {
    width,
    isResizing,
    handlePointerDown,
    handlePointerMove,
    handlePointerUp,
    handleDoubleClick,
  };
}
