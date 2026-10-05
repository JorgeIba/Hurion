import { renderToStaticMarkup } from 'react-dom/server';
import { describe, expect, it } from 'vitest';
import { getProblemPriorityLabel } from '@/domain/class/problems';
import { ProblemRow } from './ProblemRow';
import { ScoreboardTable } from '../ScoreboardTable';

describe('priority presentation', () => {
  it.each([
    ['essential', 'Essential'],
    ['recommended', 'Recommended'],
    ['supplementary', 'Supplementary'],
    ['not_required', 'Not required'],
    ['discouraged', 'Discouraged'],
    [null, 'No priority assigned'],
    [undefined, 'No priority assigned'],
    ['future_code', 'future_code'],
  ])('formats %s', (code, label) => {
    expect(getProblemPriorityLabel(code)).toBe(label);
  });

  it.each([
    ['recommended', 'Recommended'],
    [null, 'No priority assigned'],
  ])('shows %s in the catalog and scoreboard', (priority, label) => {
    const problem = { id: 1, name: 'Two Sum', difficulty: 'easy', priority };
    const catalog = renderToStaticMarkup(
      <ProblemRow problem={problem} totalStudents={1} />
    );
    const scoreboard = renderToStaticMarkup(
      <ScoreboardTable
        problems={[problem]}
        users={[{ userId: 'student', displayName: 'Student', solved: [1] }]}
      />
    );
    expect(catalog).toContain(`Priority: ${label}`);
    expect(scoreboard).toContain(`Priority: ${label}`);
    expect(scoreboard).toContain('aria-label="Solved"');
  });
});
