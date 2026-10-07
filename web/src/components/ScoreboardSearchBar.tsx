import { useId, useRef } from 'react';
import { CLASS_LABELS } from '@/domain/class/constants';
import type { SearchIssue } from '@/domain/class/parser';

interface ScoreboardSearchBarProps {
  query: string;
  onQueryChange: (query: string) => void;
  issues: readonly SearchIssue[];
}

export function ScoreboardSearchBar({
  query,
  onQueryChange,
  issues,
}: ScoreboardSearchBarProps) {
  const id = useId();
  const inputRef = useRef<HTMLInputElement>(null);
  const labels = CLASS_LABELS.SCOREBOARD_SEARCH;
  const clear = () => {
    onQueryChange('');
    inputRef.current?.focus();
  };

  return (
    <div className="scoreboard-search" role="search" aria-label={labels.LABEL}>
      <label htmlFor={id}>{labels.LABEL}</label>
      <div className="scoreboard-search-controls">
        <input
          ref={inputRef}
          id={id}
          type="search"
          value={query}
          onChange={(event) => onQueryChange(event.target.value)}
          onKeyDown={(event) => {
            if (event.key === 'Escape' && !event.nativeEvent.isComposing) {
              event.preventDefault();
              clear();
            }
          }}
          placeholder={labels.PLACEHOLDER}
          aria-describedby={`${id}-help ${id}-issues`}
          aria-invalid={issues.length > 0}
          autoComplete="off"
          spellCheck={false}
        />
        <button
          type="button"
          className="btn-reset-filters"
          disabled={query.length === 0}
          onClick={clear}
        >
          {labels.CLEAR}
        </button>
      </div>
      <p id={`${id}-help`} className="scoreboard-search-help">
        {labels.HELP}
      </p>
      <div id={`${id}-issues`} className="scoreboard-search-issues" role="status">
        {issues.length > 0 && (
          <p>
            {labels.INVALID_ID} {issues.map(({ token }) => token).join(', ')}.{' '}
            {labels.ID_HINT}
          </p>
        )}
      </div>
    </div>
  );
}
