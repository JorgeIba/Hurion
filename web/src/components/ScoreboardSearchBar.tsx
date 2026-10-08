import { useEffect, useId, useRef, useState } from 'react';
import { CLASS_LABELS } from '@/domain/class/constants';
import type { SearchIssue } from '@/domain/class/parser';

interface ScoreboardSearchBarProps {
  query: string;
  onQueryChange: (query: string, isComposing?: boolean) => void;
  onQuerySubmit: () => void;
  issues: readonly SearchIssue[];
}

export function ScoreboardSearchBar({
  query,
  onQueryChange,
  onQuerySubmit,
  issues,
}: ScoreboardSearchBarProps) {
  const id = useId();
  const inputRef = useRef<HTMLInputElement>(null);
  const triggerRef = useRef<HTMLButtonElement>(null);
  const helpRef = useRef<HTMLDivElement>(null);
  const [isOpen, setIsOpen] = useState(false);
  const [showHelp, setShowHelp] = useState(false);
  const expanded = isOpen || query.length > 0;
  useEffect(() => {
    if (isOpen) inputRef.current?.focus({ preventScroll: true });
  }, [isOpen]);
  useEffect(() => {
    if (!showHelp) return;
    const dismissOutside = (event: PointerEvent) => {
      if (event.target instanceof Node && !helpRef.current?.contains(event.target)) {
        setShowHelp(false);
      }
    };
    document.addEventListener('pointerdown', dismissOutside, true);
    return () => document.removeEventListener('pointerdown', dismissOutside, true);
  }, [showHelp]);
  const labels = CLASS_LABELS.SCOREBOARD_SEARCH;
  const clear = () => {
    onQueryChange('');
    inputRef.current?.focus({ preventScroll: true });
  };

  return (
    <div
      className="scoreboard-search"
      role="search"
      aria-label={labels.LABEL}
      onBlur={(event) => {
        if (!event.currentTarget.contains(event.relatedTarget)) {
          setIsOpen(false);
        }
      }}
    >
      <div className="scoreboard-search-actions">
        <div className="scoreboard-search-controls" data-expanded={expanded}>
          <button
            ref={triggerRef}
            type="button"
            className="scoreboard-search-trigger"
            aria-label={labels.LABEL}
            aria-expanded={expanded}
            aria-controls={id}
            onClick={() => {
              setIsOpen(true);
              inputRef.current?.focus({ preventScroll: true });
            }}
          >
            <svg
              aria-hidden="true"
              width="18"
              height="18"
              viewBox="0 0 24 24"
              fill="none"
            >
              <circle cx="10.5" cy="10.5" r="6.5" />
              <path d="m16 16 4 4" />
            </svg>
            {labels.TRIGGER}
          </button>
          <label htmlFor={id} className="scoreboard-search-accessible-text">
            {labels.LABEL}
          </label>
          <input
            ref={inputRef}
            id={id}
            type="search"
            value={query}
            hidden={!expanded}
            tabIndex={expanded ? 0 : -1}
            onFocus={() => setIsOpen(true)}
            onChange={(event) =>
              onQueryChange(
                event.target.value,
                (event.nativeEvent as InputEvent).isComposing
              )
            }
            onCompositionEnd={(event) => onQueryChange(event.currentTarget.value)}
            onKeyDown={(event) => {
              if (event.key === 'Enter' && !event.nativeEvent.isComposing) {
                event.preventDefault();
                onQuerySubmit();
              }
              if (event.key === 'Escape' && !event.nativeEvent.isComposing) {
                event.preventDefault();
                if (query.length > 0) {
                  clear();
                } else {
                  setIsOpen(false);
                  triggerRef.current?.focus();
                }
              }
            }}
            placeholder={labels.PLACEHOLDER}
            aria-describedby={`${id}-help${issues.length > 0 ? ` ${id}-issues` : ''}`}
            aria-invalid={issues.length > 0}
            autoComplete="off"
            spellCheck={false}
          />
          {query.length > 0 && (
            <button
              type="button"
              className="scoreboard-search-clear"
              aria-label={labels.CLEAR}
              title={labels.CLEAR}
              onClick={clear}
            >
              <svg
                aria-hidden="true"
                width="16"
                height="16"
                viewBox="0 0 24 24"
                fill="none"
              >
                <path d="m6 6 12 12M18 6 6 18" />
              </svg>
            </button>
          )}
        </div>
        <div
          ref={helpRef}
          className="scoreboard-search-help-anchor"
          onMouseEnter={() => setShowHelp(true)}
          onMouseLeave={(event) => {
            if (!event.currentTarget.contains(document.activeElement)) setShowHelp(false);
          }}
          onFocus={() => setShowHelp(true)}
          onBlur={(event) => {
            if (!event.currentTarget.contains(event.relatedTarget)) setShowHelp(false);
          }}
          onKeyDown={(event) => {
            if (event.key === 'Escape') {
              event.preventDefault();
              setShowHelp(false);
            }
          }}
        >
          <button
            type="button"
            className="scoreboard-search-help-trigger"
            aria-label={labels.HELP_LABEL}
            aria-describedby={`${id}-help`}
            onClick={() => setShowHelp(true)}
          >
            <svg
              aria-hidden="true"
              width="18"
              height="18"
              viewBox="0 0 24 24"
              fill="none"
            >
              <circle cx="12" cy="12" r="9" />
              <path d="M9.5 9a2.5 2.5 0 0 1 5 0c0 2-2.5 2-2.5 4M12 16h.01" />
            </svg>
          </button>
          <p
            id={`${id}-help`}
            role="tooltip"
            className="scoreboard-search-help"
            hidden={!showHelp}
          >
            {labels.HELP}
          </p>
        </div>
        <div id={`${id}-issues`} className="scoreboard-search-issues" role="status">
          {issues.length > 0 && (
            <p>
              {labels.INVALID_ID} {issues.map(({ token }) => token).join(', ')}.{' '}
              {labels.ID_HINT}
            </p>
          )}
        </div>
      </div>
    </div>
  );
}
