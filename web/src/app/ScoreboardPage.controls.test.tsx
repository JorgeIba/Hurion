// @vitest-environment jsdom
import { transferableAbortController } from 'node:util';
import { act, StrictMode } from 'react';
import { createRoot, type Root } from 'react-dom/client';
import { createMemoryRouter } from 'react-router';
import { RouterProvider } from 'react-router/dom';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { ScoreboardPage } from './ScoreboardPage';
import { apiClient } from '@/domain/class/api';
import type { ClassData } from '@/domain/class/types';

const classData: ClassData = {
  classCode: 'basicos',
  className: 'Básicos',
  problems: [
    {
      id: 1,
      name: 'Legacy',
      topic: 'dp',
      difficulty: 'hard',
      priority: 'discouraged',
      createdAt: '2026-01-01',
    },
    {
      id: 4,
      name: 'Starter',
      topic: 'dp',
      difficulty: 'easy',
      priority: 'recommended',
      createdAt: '2026-01-01',
    },
    {
      id: 5,
      name: 'Must',
      topic: 'dp',
      difficulty: 'hard',
      priority: 'essential',
      createdAt: '2026-01-02',
    },
    {
      id: 8,
      name: 'Core',
      topic: 'dp',
      difficulty: 'medium',
      priority: 'essential',
      createdAt: '2026-01-01',
    },
    {
      id: 2,
      name: 'BFS',
      topic: 'grafos',
      difficulty: 'easy',
      priority: 'recommended',
      createdAt: '2026-01-01',
    },
    {
      id: 3,
      name: 'Flow',
      topic: 'grafos',
      difficulty: 'hard',
      priority: 'essential',
      createdAt: '2026-01-01',
    },
    { id: 20, name: 'Unknown', topic: 'math', createdAt: '2026-01-01' },
  ],
  users: [
    { userId: 'jorge', displayName: 'Jorge', solved: [1, 8, 2, 3], solvedCount: 4 },
    { userId: 'luis', displayName: 'Luis', solved: [4], solvedCount: 1 },
  ],
};

let container: HTMLDivElement;
let root: Root;
let router: ReturnType<typeof createMemoryRouter>;
let previousAdapter: typeof apiClient.defaults.adapter;

async function mount(url = '/?tab=problems') {
  router = createMemoryRouter([{ path: '*', element: <ScoreboardPage /> }], {
    initialEntries: [url],
  });
  await act(async () => {
    root.render(
      <StrictMode>
        <RouterProvider router={router} />
      </StrictMode>
    );
  });
}

function panel() {
  const active = container.querySelector<HTMLElement>(
    '.tabpanel-wrapper:not([hidden]) [role="tabpanel"]'
  );
  expect(active).not.toBeNull();
  return active!;
}

function control(label: string) {
  const element = Array.from(panel().querySelectorAll('label'))
    .find((candidate) => candidate.querySelector('span')?.textContent === label)
    ?.querySelector('select');
  expect(element, `Missing control: ${label}`).toBeDefined();
  return element!;
}

async function select(label: string, value: string) {
  const element = control(label);
  await act(async () => {
    element.value = value;
    element.dispatchEvent(new Event('change', { bubbles: true }));
  });
}

async function switchTab(tab: 'standings' | 'problems') {
  await act(async () => {
    container.querySelector<HTMLButtonElement>(`#tab-${tab}`)!.click();
  });
}

function names() {
  return Array.from(
    panel().querySelectorAll('.problem-name-text, .problem-list-title-text')
  )
    .filter((element) => !element.closest('[hidden]'))
    .map((element) => element.textContent);
}

async function clickStudent(name: string) {
  const button = Array.from(
    panel().querySelectorAll<HTMLButtonElement>('.student-sort-button')
  ).find((element) => element.querySelector('.student-name-text')?.textContent === name);
  expect(button).toBeDefined();
  await act(async () => button!.click());
}

async function toggleAllTopics() {
  const button = panel().querySelector<HTMLButtonElement>('.btn-toggle-all-topics');
  expect(button).not.toBeNull();
  await act(async () => button!.click());
}

async function search(query: string) {
  const input = container.querySelector<HTMLInputElement>('.scoreboard-search input')!;
  if (input.hidden) {
    await act(async () =>
      container.querySelector<HTMLButtonElement>('.scoreboard-search-trigger')!.click()
    );
  }
  await act(async () => {
    Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, 'value')!.set!.call(
      input,
      query
    );
    input.dispatchEvent(new InputEvent('input', { bubbles: true }));
  });
  await act(async () => {
    input.dispatchEvent(new KeyboardEvent('keydown', { bubbles: true, key: 'Enter' }));
  });
}

beforeEach(() => {
  vi.stubGlobal('IS_REACT_ACT_ENVIRONMENT', true);
  vi.stubGlobal('AbortController', transferableAbortController);
  previousAdapter = apiClient.defaults.adapter;
  // Replace only the HTTP boundary; exercise real fetching, routing and views.
  apiClient.defaults.adapter = async (config) => ({
    data: classData,
    status: 200,
    statusText: 'OK',
    headers: {},
    config,
  });
  container = document.createElement('div');
  document.body.append(container);
  root = createRoot(container);
});

afterEach(() => {
  act(() => root.unmount());
  router.dispose();
  container.remove();
  apiClient.defaults.adapter = previousAdapter;
  vi.unstubAllGlobals();
});

describe('shared problem controls', () => {
  it('keeps the chosen priority order when moving between Problems and Standings', async () => {
    await mount();
    await select('Order within topics', 'priority-essential-first');
    const expected = ['Must', 'Core', 'Starter', 'Legacy', 'Flow', 'BFS', 'Unknown'];
    expect(names()).toEqual(expected);
    await switchTab('standings');
    expect(names()).toEqual(expected);
    expect(control('Order within topics').value).toBe('priority-essential-first');
    await switchTab('problems');
    expect(names()).toEqual(expected);
  });

  it('shares combined difficulty and priority filters without changing topic order or class totals', async () => {
    await mount('/');
    expect(control('Order within topics').value).toBe('id');
    await select('Difficulty', 'hard');
    await select('Priority', 'essential');
    // DP stays first even though its first remaining problem has ID 5, after grafos' 3.
    expect(names()).toEqual(['Must', 'Flow']);
    expect(panel().querySelector('.problems-count-badge')?.textContent).toContain(
      '2 of 7'
    );
    expect(
      Array.from(
        panel().querySelectorAll('.total-ac-count-cell'),
        (element) => element.textContent
      )
    ).toEqual(['4', '1']);
    await switchTab('problems');
    expect(names()).toEqual(['Must', 'Flow']);
    expect(control('Difficulty').value).toBe('hard');
    expect(control('Priority').value).toBe('essential');
    expect(panel().querySelector('[role="meter"]')?.getAttribute('aria-valuemax')).toBe(
      '2'
    );
    await select('Difficulty', 'easy');
    expect(names()).toEqual([]);
    await act(async () => {
      panel().querySelector<HTMLButtonElement>('.btn-reset-filters')!.click();
    });
    expect(names()).toHaveLength(7);
    await switchTab('standings');
    expect(names()).toHaveLength(7);
    expect(control('Difficulty').value).toBe('all');
    expect(control('Priority').value).toBe('all');
  });

  it('uses priority within solved and pending groups and restores normal order on the third student click', async () => {
    await mount('/');
    await select('Order within topics', 'priority-essential-first');
    await clickStudent('Jorge');
    // His fully solved grafos topic leads; within DP, solved Core precedes Legacy.
    expect(names()).toEqual([
      'Flow',
      'BFS',
      'Core',
      'Legacy',
      'Must',
      'Starter',
      'Unknown',
    ]);
    expect(control('Order within solved/pending groups').value).toBe(
      'priority-essential-first'
    );
    await clickStudent('Jorge');
    expect(names()).toEqual([
      'Must',
      'Starter',
      'Core',
      'Legacy',
      'Unknown',
      'Flow',
      'BFS',
    ]);
    await clickStudent('Jorge');
    expect(names()).toEqual([
      'Must',
      'Core',
      'Starter',
      'Legacy',
      'Flow',
      'BFS',
      'Unknown',
    ]);
    expect(control('Order within topics').value).toBe('priority-essential-first');
  });

  it('changes the difficulty tie-breaker across tabs without discarding the selected student', async () => {
    await mount('/');
    await clickStudent('Jorge');
    await switchTab('problems');
    await select('Order within topics', 'difficulty-hard-first');
    expect(names()).toEqual([
      'Legacy',
      'Must',
      'Core',
      'Starter',
      'Flow',
      'BFS',
      'Unknown',
    ]);
    await switchTab('standings');
    expect(names()).toEqual([
      'Flow',
      'BFS',
      'Legacy',
      'Core',
      'Must',
      'Starter',
      'Unknown',
    ]);
    expect(control('Order within solved/pending groups').value).toBe(
      'difficulty-hard-first'
    );
  });

  it('expands and collapses all topics independently in each tab', async () => {
    await mount('/');
    await toggleAllTopics();
    expect(names()).toEqual([]);
    await switchTab('problems');
    expect(names()).toHaveLength(7);
    await toggleAllTopics();
    expect(names()).toEqual([]);
    await switchTab('standings');
    expect(names()).toEqual([]);
    await toggleAllTopics();
    expect(names()).toHaveLength(7);
    await switchTab('problems');
    expect(names()).toEqual([]);
    await toggleAllTopics();
    expect(names()).toHaveLength(7);
  });

  it('intersects shared filters with standings search without filtering Problems or redefining student rank', async () => {
    await mount('/');
    await select('Priority', 'essential');
    await search('@luis #3');
    expect(names()).toEqual(['Flow']);
    expect(
      panel().querySelector('.col-student-header')?.classList.contains('rank-2-header')
    ).toBe(true);
    expect(panel().querySelector('.total-ac-count-cell')?.textContent).toBe('1');
    await switchTab('problems');
    expect(names()).toEqual(['Must', 'Core', 'Flow']);
    await switchTab('standings');
    expect(names()).toEqual(['Flow']);
    await search('');
    expect(names()).toEqual(['Must', 'Core', 'Flow']);
    expect(control('Priority').value).toBe('essential');
  });

  it('temporarily expands search matches and restores the saved collapse state when search clears', async () => {
    await mount('/');
    await toggleAllTopics();
    await search('@luis');
    expect(names()).toHaveLength(7);
    expect(
      panel().querySelector<HTMLButtonElement>('.btn-toggle-all-topics')?.disabled
    ).toBe(true);
    await search('');
    expect(names()).toEqual([]);
    await toggleAllTopics();
    expect(names()).toHaveLength(7);
  });

  it('resets shared controls and student sorting when changing class', async () => {
    await mount('/');
    await select('Priority', 'essential');
    await select('Order within topics', 'difficulty-hard-first');
    await clickStudent('Jorge');
    await toggleAllTopics();
    await act(async () => {
      await router.navigate('/?class=avanzados');
    });
    expect(control('Priority').value).toBe('all');
    expect(control('Order within topics').value).toBe('id');
    expect(names()).toEqual([
      'Legacy',
      'Starter',
      'Must',
      'Core',
      'BFS',
      'Flow',
      'Unknown',
    ]);
    expect(panel().querySelector('[aria-sort="descending"]')).toBeNull();
  });
});
