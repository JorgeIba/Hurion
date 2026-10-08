// @vitest-environment jsdom
import { act } from 'react';
import { createRoot, type Root } from 'react-dom/client';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { StandingsView } from '@/components/StandingsView';
import { CLASS_CONFIG, CLASS_LABELS } from '@/domain/class/constants';
import type { Problem, UserScore } from '@/domain/class/types';
import { priorityPresentationConfig } from '@/presentation/priorityConfig';

const problems: Problem[] = [
  { id: 12, name: 'BFS', topic: 'grafos', createdAt: '2026-01-01' },
  { id: 120, name: 'Grafo implícito', topic: 'dp', createdAt: '2026-01-01' },
  { id: 3, name: 'Mochila', topic: 'dp', createdAt: '2026-01-01' },
];
const users: UserScore[] = [
  { userId: 'jorgeiba', displayName: 'Jorge', solved: [12, 3], solvedCount: 2 },
  { userId: 'luis', displayName: 'Luis', solved: [120], solvedCount: 1 },
  { userId: 'huron', displayName: 'Huron', solved: [], solvedCount: 0 },
];
let container: HTMLDivElement;
let root: Root;
let input: HTMLInputElement;

function mount(classProblems = problems, classUsers = users, classCode = 'basicos') {
  act(() =>
    root.render(
      <StandingsView
        key={classCode}
        priorityPresentation={priorityPresentationConfig.defaults}
        problems={classProblems}
        users={classUsers}
      />
    )
  );
  act(() =>
    container.querySelector<HTMLButtonElement>('.scoreboard-search-trigger')!.click()
  );
  input = container.querySelector('input')!;
}

function type(value: string) {
  act(() => {
    Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, 'value')!.set!.call(
      input,
      value
    );
    input.dispatchEvent(new InputEvent('input', { bubbles: true }));
  });
}

function apply(query: string) {
  type(query);
  act(() => vi.advanceTimersByTime(CLASS_CONFIG.SCOREBOARD_SEARCH.DEBOUNCE_MS));
}

function names() {
  return Array.from(
    container.querySelectorAll('.problem-name-text'),
    (element) => element.textContent
  );
}

function studentNames() {
  return Array.from(
    container.querySelectorAll('.student-name-text'),
    (element) => element.textContent
  );
}

beforeEach(() => {
  vi.stubGlobal('IS_REACT_ACT_ENVIRONMENT', true);
  vi.useFakeTimers();
  container = document.createElement('div');
  document.body.append(container);
  root = createRoot(container);
  mount();
});
afterEach(() => {
  act(() => root.unmount());
  container.remove();
  vi.useRealTimers();
  vi.unstubAllGlobals();
});

describe('search applied to scoreboard rows and columns', () => {
  it('keeps the full table until the pause, matches IDs exactly, and clears immediately', () => {
    type('#12');
    act(() => vi.advanceTimersByTime(CLASS_CONFIG.SCOREBOARD_SEARCH.DEBOUNCE_MS - 1));
    expect(names()).toHaveLength(3);
    act(() => vi.advanceTimersByTime(1));
    expect(names()).toEqual(['BFS']);
    expect(studentNames()).toEqual(['Jorge', 'Luis', 'Huron']);
    expect(
      container.querySelector('.scoreboard-selection-summary')?.textContent
    ).toContain('1 of 3 problems');
    act(() =>
      container.querySelector<HTMLButtonElement>('.scoreboard-search-clear')!.click()
    );
    expect(names()).toHaveLength(3);
    expect(container.querySelector('.scoreboard-selection-summary')).toBeNull();
  });

  it('combines the student restriction with topics OR text from another topic', () => {
    apply('@luis t:grafos + grafo');
    expect(studentNames()).toEqual(['Luis']);
    expect(names()).toEqual(['Grafo implícito', 'BFS']);
    const rows = container.querySelectorAll('.problem-row');
    expect(rows[0].querySelectorAll('.status-cell')).toHaveLength(1);
    expect(rows[0].querySelector('.status-ac-pill')).not.toBeNull();
    expect(rows[1].querySelector('.status-ac-pill')).toBeNull();
    expect(container.querySelector('.topic-header-row td')?.getAttribute('colspan')).toBe(
      '2'
    );
    expect(container.querySelectorAll('col')).toHaveLength(2);
  });

  it('preserves class rank, totals and mastery when selecting a single student and problem', () => {
    apply('@luis #120');
    const header = container.querySelector('.col-student-header')!;
    expect(header.classList.contains('rank-2-header')).toBe(true);
    expect(header.classList.contains('rank-1-header')).toBe(false);
    expect(header.querySelector('.student-crown')?.textContent).toBe(
      CLASS_LABELS.MEDALS.SILVER
    );
    expect(header.querySelector('.mastered-badge')).toBeNull();
    expect(container.querySelector('.total-ac-count-cell')?.textContent).toBe('1');
    expect(container.querySelector('.status-ac-pill')).not.toBeNull();
    apply('@jorge #12');
    expect(container.querySelector('.total-ac-count-cell')?.textContent).toBe('2');
    expect(container.querySelector('.mastered-badge')).toBeNull();
  });

  it('expands matches during search and restores prior collapse state on clear', () => {
    const topic = Array.from(
      container.querySelectorAll<HTMLButtonElement>('.topic-header-btn')
    ).find((button) => button.textContent?.toLowerCase().includes('grafos'))!;
    act(() => topic.click());
    expect(topic.getAttribute('aria-expanded')).toBe('false');
    expect(names()).not.toContain('BFS');
    apply('t:grafos');
    const matchingTopic =
      container.querySelector<HTMLButtonElement>('.topic-header-btn')!;
    expect(matchingTopic.disabled).toBe(true);
    expect(matchingTopic.getAttribute('aria-expanded')).toBe('true');
    expect(names()).toEqual(['BFS']);
    type('');
    const restoredTopic = Array.from(
      container.querySelectorAll<HTMLButtonElement>('.topic-header-btn')
    ).find((button) => button.textContent?.toLowerCase().includes('grafos'))!;
    expect(restoredTopic.disabled).toBe(false);
    expect(restoredTopic.getAttribute('aria-expanded')).toBe('false');
    expect(names()).not.toContain('BFS');
    act(() => restoredTopic.click());
    expect(names()).toContain('BFS');
  });

  it.each([
    ['@unknown', CLASS_LABELS.SCOREBOARD_VIEW.NO_MATCHING_STUDENTS],
    ['#999', CLASS_LABELS.SCOREBOARD_VIEW.NO_MATCHING_PROBLEMS],
    ['@unknown #999', CLASS_LABELS.SCOREBOARD_VIEW.NO_MATCHES],
  ])('shows a recoverable empty state for %s', (query, message) => {
    apply(query);
    expect(container.querySelector('table')).toBeNull();
    expect(container.querySelector('.state-row')?.textContent).toBe(message);
    expect(input.value).toBe(query);
    type('');
    expect(names()).toHaveLength(3);
  });

  it('keeps valid alternatives when another problem ID is invalid', () => {
    apply('#wrong');
    expect(names()).toHaveLength(0);
    expect(input.getAttribute('aria-invalid')).toBe('true');
    apply('#wrong + #3');
    expect(names()).toEqual(['Mochila']);
    expect(input.getAttribute('aria-invalid')).toBe('true');
    apply('@ t: #');
    expect(names()).toHaveLength(3);
    expect(
      container.querySelector<HTMLButtonElement>('.topic-header-btn')!.disabled
    ).toBe(false);
  });

  it('cycles sorting through a native button and exposes direction on the column header', () => {
    const header = container.querySelectorAll<HTMLElement>('.col-student-header')[1];
    const button = header.querySelector<HTMLButtonElement>('button')!;
    expect(button.type).toBe('button');
    expect(header.hasAttribute('tabindex')).toBe(false);
    expect(header.getAttribute('aria-sort')).toBe('none');
    act(() => button.click());
    expect(header.getAttribute('aria-sort')).toBe('descending');
    expect(names()).toEqual(['Grafo implícito', 'Mochila', 'BFS']);
    act(() => button.click());
    expect(header.getAttribute('aria-sort')).toBe('ascending');
    expect(names()).toEqual(['BFS', 'Mochila', 'Grafo implícito']);
    act(() => button.click());
    expect(header.getAttribute('aria-sort')).toBe('none');
    expect(names()).toEqual(['Mochila', 'Grafo implícito', 'BFS']);
  });

  it.each([
    [[], [], CLASS_LABELS.NO_PROBLEMS],
    [problems, [], CLASS_LABELS.SCOREBOARD_VIEW.NO_STUDENTS],
  ])(
    'distinguishes missing class data from empty search results',
    (classProblems, classUsers, message) => {
      mount(classProblems, classUsers);
      apply('@unknown #999');
      expect(container.querySelectorAll('#tabpanel-standings')).toHaveLength(1);
      expect(container.querySelector('table')).toBeNull();
      expect(container.querySelector('.state-row')?.textContent).toBe(message);
    }
  );

  it('preserves full-class sorting when the sorted student is hidden and allows resetting it', () => {
    // Luis has only solved a DP problem, so DP should lead when sorting by him.
    const headers = container.querySelectorAll<HTMLButtonElement>('.student-sort-button');
    act(() => headers[1].click());
    expect(names()[0]).toBe('Grafo implícito');
    apply('@jorge');
    expect(names()[0]).toBe('Grafo implícito');
    expect(container.querySelector('.scoreboard-sort-summary')?.textContent).toContain(
      'Sorted by Luis'
    );
    act(() =>
      container
        .querySelector<HTMLButtonElement>('.scoreboard-sort-summary button')!
        .click()
    );
    expect(names()[0]).toBe('Mochila');
    expect(container.querySelector('.scoreboard-sort-summary')).toBeNull();
  });

  it('evaluates consecutive queries against the full class and resets for another class', () => {
    const before = JSON.stringify({ problems, users });
    apply('@jorge #12');
    apply('@luis #120');
    expect(names()).toEqual(['Grafo implícito']);
    expect(studentNames()).toEqual(['Luis']);
    mount(problems, users, 'avanzados');
    expect(input.value).toBe('');
    expect(names()).toHaveLength(3);
    expect(studentNames()).toHaveLength(3);
    expect(JSON.stringify({ problems, users })).toBe(before);
  });
});
