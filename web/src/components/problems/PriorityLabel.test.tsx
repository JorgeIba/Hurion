import { MemoryRouter } from 'react-router';
import { renderToStaticMarkup } from 'react-dom/server';
import { describe, expect, it } from 'vitest';
import {
  getProblemPriorityLabel,
  resolvePriorityPresentation,
} from '@/presentation/priority';
import { ProblemRow } from './ProblemRow';
import { PriorityLabel } from './PriorityLabel';
import { ScoreboardTable } from '../ScoreboardTable';
import { ProblemsView } from '../ProblemsView';
import { DEFAULT_PROBLEM_VIEW_OPTIONS } from '@/domain/class/constants';

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
    expect(
      getProblemPriorityLabel(code, resolvePriorityPresentation('another_class'))
    ).toBe(label);
  });

  it.each([
    ['essential', '⭐⭐⭐ Essential'],
    ['recommended', '⭐⭐ To Qualify to PDA'],
    ['supplementary', '⭐ For PDA Champ'],
    ['not_required', 'No Priority'],
    ['discouraged', '🌑 No lo hagan es basura'],
    [null, 'No Priority'],
    [undefined, 'No Priority'],
    ['unset', 'unset'],
    ['future_code', 'future_code'],
    ['toString', 'toString'],
  ])('overrides only presentation for %s in avanzados', (code, label) => {
    expect(getProblemPriorityLabel(code, resolvePriorityPresentation('avanzados'))).toBe(
      label
    );
  });

  it('renders a description without displaying an empty symbol', () => {
    const html = renderToStaticMarkup(
      <PriorityLabel
        presentation={{ label: 'Essential', symbol: '', description: 'Core practice' }}
      />
    );

    expect(html).toBe(
      '<span class="problem-priority-label" title="Core practice">Essential</span>'
    );
  });

  it.each([
    ['recommended', 'Recommended'],
    [null, 'No priority assigned'],
  ])('shows %s in the catalog and scoreboard', (priority, label) => {
    const problem = {
      createdAt: '2026-10-04T00:00:00Z',
      id: 1,
      name: 'Two Sum',
      difficulty: 'easy',
      priority,
    };
    const priorityPresentation = resolvePriorityPresentation();
    const catalog = renderToStaticMarkup(
      <MemoryRouter>
        <ProblemRow
          priorityPresentation={priorityPresentation}
          problem={problem}
          totalStudents={1}
        />
      </MemoryRouter>
    );
    const scoreboard = renderToStaticMarkup(
      <ScoreboardTable
        options={DEFAULT_PROBLEM_VIEW_OPTIONS}
        onOptionsChange={() => {}}
        priorityPresentation={priorityPresentation}
        problems={[problem]}
        users={[{ userId: 'student', displayName: 'Student', solved: [1] }]}
      />
    );
    expect(catalog).toContain(`class="problem-priority-label">${label}</span>`);
    expect(scoreboard).toContain(`class="problem-priority-label">${label}</span>`);
    expect(scoreboard).toContain('aria-label="Solved"');
  });

  it('uses class presentation in both views and controls without changing codes or ACs', () => {
    const problems = [
      {
        createdAt: '2026-10-04T00:00:00Z',
        id: 1,
        name: 'Two Sum',
        priority: 'essential',
      },
      {
        createdAt: '2026-10-04T00:00:00Z',
        id: 2,
        name: 'Another problem',
        priority: null,
      },
      {
        createdAt: '2026-10-04T00:00:00Z',
        id: 3,
        name: 'Recommended problem',
        priority: 'recommended',
      },
      {
        createdAt: '2026-10-04T00:00:00Z',
        id: 4,
        name: 'Supplementary problem',
        priority: 'supplementary',
      },
      {
        createdAt: '2026-10-04T00:00:00Z',
        id: 5,
        name: 'Optional problem',
        priority: 'not_required',
      },
      {
        createdAt: '2026-10-04T00:00:00Z',
        id: 6,
        name: 'Discouraged problem',
        priority: 'discouraged',
      },
    ];
    const users = [{ userId: 'student', displayName: 'Student', solved: [1] }];
    const priorityPresentation = resolvePriorityPresentation('avanzados');
    const catalog = renderToStaticMarkup(
      <MemoryRouter>
        <ProblemsView
          options={DEFAULT_PROBLEM_VIEW_OPTIONS}
          onOptionsChange={() => {}}
          priorityPresentation={priorityPresentation}
          problems={problems}
          users={users}
        />
      </MemoryRouter>
    );
    const scoreboard = renderToStaticMarkup(
      <ScoreboardTable
        options={DEFAULT_PROBLEM_VIEW_OPTIONS}
        onOptionsChange={() => {}}
        priorityPresentation={priorityPresentation}
        problems={problems}
        users={users}
      />
    );
    for (const html of [catalog, scoreboard]) {
      expect(html).toContain('aria-hidden="true">⭐⭐⭐ </span>Essential');
      expect(html).toContain('aria-hidden="true">⭐⭐ </span>To Qualify to PDA');
      expect(html).toContain('aria-hidden="true">⭐ </span>For PDA Champ');
      expect(html).toContain('aria-hidden="true">🌑 </span>No lo hagan es basura');
      expect(html).toContain('class="problem-priority-label">No Priority</span>');
      expect(html).not.toContain('Priority:');
    }
    expect(catalog).toContain('<option value="essential">⭐⭐⭐ Essential</option>');
    expect(catalog).toContain(
      '<option value="recommended">⭐⭐ To Qualify to PDA</option>'
    );
    expect(catalog).toContain('<option value="supplementary">⭐ For PDA Champ</option>');
    expect(catalog).toContain('<option value="not_required">No Priority</option>');
    expect(catalog).toContain('<option value="unset">No Priority</option>');
    expect(catalog).toContain(
      '<option value="priority-essential-first">Priority · Most recommended first</option>'
    );
    expect(catalog).toContain(
      '<option value="priority-discouraged-first">Priority · Least recommended first</option>'
    );
    expect(scoreboard).toContain('aria-label="Solved"');
    expect(problems.map((problem) => problem.priority)).toEqual([
      'essential',
      null,
      'recommended',
      'supplementary',
      'not_required',
      'discouraged',
    ]);
    expect(users[0].solved).toEqual([1]);
  });
});
