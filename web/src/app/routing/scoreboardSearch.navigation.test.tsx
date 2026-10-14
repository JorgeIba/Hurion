// @vitest-environment jsdom
import { transferableAbortController } from 'node:util';
import { act, StrictMode } from 'react';
import { createRoot, type Root } from 'react-dom/client';
import { createMemoryRouter, useLocation, useNavigate } from 'react-router';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { RouterProvider } from 'react-router/dom';
import { ScoreboardPage } from '@/app/ScoreboardPage';
import { CLASS_CONFIG, CLASS_LABELS } from '@/domain/class/constants';

vi.mock('@/hooks/useScoreboardData', () => ({
  useScoreboardData: (classCode: string) => ({
    loading: false,
    error: null,
    data: {
      classCode,
      problems: [
        {
          id: 12,
          name: 'BFS',
          topic: 'grafos',
          url: 'https://example.com/bfs',
          createdAt: '2026-01-01',
        },
        { id: 120, name: 'DP', topic: 'dp', createdAt: '2026-01-01' },
      ],
      users:
        classCode === 'no-solves'
          ? [{ userId: 'jorge', displayName: 'Jorge', solved: [], solvedCount: 0 }]
          : [
              { userId: 'jorge', displayName: 'Jorge', solved: [12], solvedCount: 1 },
              { userId: 'luis', displayName: 'Luis', solved: [120], solvedCount: 1 },
            ],
    },
  }),
}));

let container: HTMLDivElement;
let root: Root;
let navigate: ReturnType<typeof useNavigate>;
let location: ReturnType<typeof useLocation>;

function NavigationProbe() {
  navigate = useNavigate();
  location = useLocation();
  return null;
}

let router: ReturnType<typeof createMemoryRouter>;
function mount(entries = ['/?class=basicos']) {
  router = createMemoryRouter(
    [
      {
        path: '*',
        element: (
          <>
            <ScoreboardPage />
            <NavigationProbe />
          </>
        ),
      },
    ],
    { initialEntries: entries }
  );
  act(() =>
    root.render(
      <StrictMode>
        <RouterProvider router={router} />
      </StrictMode>
    )
  );
}

function input() {
  const element = container.querySelector<HTMLInputElement>('.scoreboard-search input')!;
  if (element.hidden)
    act(() =>
      container.querySelector<HTMLButtonElement>('.scoreboard-search-trigger')!.click()
    );
  return element;
}

function type(value: string) {
  const element = input();
  act(() => {
    Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, 'value')!.set!.call(
      element,
      value
    );
    element.dispatchEvent(new InputEvent('input', { bubbles: true }));
  });
}

function students() {
  return Array.from(
    container.querySelectorAll('.student-name-text'),
    (element) => element.textContent
  );
}

function params() {
  return new URLSearchParams(location.search);
}

beforeEach(() => {
  vi.stubGlobal('IS_REACT_ACT_ENVIRONMENT', true);
  vi.stubGlobal('AbortController', transferableAbortController);
  vi.useFakeTimers();
  container = document.createElement('div');
  document.body.append(container);
  root = createRoot(container);
});
afterEach(() => {
  act(() => root.unmount());
  router.dispose();
  container.remove();
  vi.useRealTimers();
  vi.unstubAllGlobals();
});

describe('scoreboard URL navigation', () => {
  it('opens solved students from progress and restores Problems through history', async () => {
    mount(['/?class=basicos&tab=problems&q=%40luis&extra=keep#anchor']);
    const title = container.querySelector<HTMLAnchorElement>('.problem-list-title-link')!;
    expect(title.href).toBe('https://example.com/bfs');
    expect(title.target).toBe('_blank');
    const progress =
      title
        .closest('.problem-list-row')
        ?.querySelector<HTMLAnchorElement>('.problem-list-solves-link') ??
      container.querySelector<HTMLAnchorElement>('.problem-list-solves-link')!;
    const destination = new URL(progress.href);
    expect(destination.searchParams.get('tab')).toBe('standings');
    expect(destination.searchParams.get('q')).toBe('solved:12');
    expect(destination.searchParams.get('class')).toBe('basicos');
    expect(destination.searchParams.get('extra')).toBe('keep');
    expect(destination.hash).toBe('#anchor');
    expect(progress.getAttribute('aria-label')).toContain('BFS');
    expect(progress.querySelector('[role="meter"]')?.getAttribute('aria-valuenow')).toBe(
      '1'
    );
    await act(async () => {
      progress.click();
    });
    expect(params().get('tab')).toBe('standings');
    expect(input().value).toBe('solved:12');
    expect(students()).toEqual(['Jorge']);
    expect(container.querySelectorAll('.problem-name-text')).toHaveLength(2);
    expect(document.activeElement?.id).toBe('tab-standings');
    await act(async () => {
      await navigate(-1);
    });
    expect(params().get('tab')).toBe('problems');
    expect(params().get('q')).toBe('@luis');
    expect(container.querySelector('#tab-problems')?.getAttribute('aria-selected')).toBe(
      'true'
    );
    await act(async () => {
      await navigate(1);
    });
    expect(input().value).toBe('solved:12');
    expect(students()).toEqual(['Jorge']);
  });

  it('opens an empty result when no students have solved the problem', async () => {
    mount(['/?class=no-solves&tab=problems']);
    const title = container.querySelector<HTMLAnchorElement>('.problem-list-title-link')!;
    const progress =
      title
        .closest('.problem-list-row')
        ?.querySelector<HTMLAnchorElement>('.problem-list-solves-link') ??
      container.querySelector<HTMLAnchorElement>('.problem-list-solves-link')!;
    expect(progress.querySelector('[role="meter"]')?.getAttribute('aria-valuenow')).toBe(
      '0'
    );
    expect(progress.querySelector('[role="meter"]')?.getAttribute('aria-valuemax')).toBe(
      '1'
    );
    await act(async () => {
      progress.click();
    });
    expect(input().value).toBe('solved:12');
    expect(students()).toEqual([]);
    expect(
      container.querySelector('.scoreboard-selection-summary')?.textContent
    ).toContain('2 of 2 problems');
    expect(container.querySelector('#tabpanel-standings .state-row')?.textContent).toBe(
      CLASS_LABELS.SCOREBOARD_VIEW.NO_MATCHING_STUDENTS
    );
  });

  it.each(['delete', 'button', 'Escape'])(
    'preserves a clear made with %s while a URL replacement is in flight',
    async (method) => {
      mount();
      type('solved:12');
      const element = input();
      await act(async () => {
        vi.advanceTimersByTime(CLASS_CONFIG.SCOREBOARD_SEARCH.DEBOUNCE_MS);
        if (method === 'delete') {
          Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, 'value')!.set!.call(
            element,
            ''
          );
          element.dispatchEvent(new InputEvent('input', { bubbles: true }));
        } else if (method === 'button') {
          container.querySelector<HTMLButtonElement>('.scoreboard-search-clear')!.click();
        } else {
          element.dispatchEvent(
            new KeyboardEvent('keydown', { bubbles: true, key: 'Escape' })
          );
        }
      });
      expect(element.value).toBe('');
      expect(params().has('q')).toBe(false);
      expect(students()).toEqual(['Jorge', 'Luis']);
      await act(async () => {
        vi.advanceTimersByTime(300);
      });
      expect(params().has('q')).toBe(false);
    }
  );

  it('discards a queued clear when the user types a newer search', async () => {
    mount();
    type('solved:12');
    const element = input();
    await act(async () => {
      vi.advanceTimersByTime(CLASS_CONFIG.SCOREBOARD_SEARCH.DEBOUNCE_MS);
      const setValue = Object.getOwnPropertyDescriptor(
        HTMLInputElement.prototype,
        'value'
      )!.set!;
      setValue.call(element, '');
      element.dispatchEvent(new InputEvent('input', { bubbles: true }));
      setValue.call(element, 'solved:120');
      element.dispatchEvent(new InputEvent('input', { bubbles: true }));
    });
    expect(element.value).toBe('solved:120');
    expect(params().get('q')).toBe('solved:12');
    await act(async () => {
      vi.advanceTimersByTime(CLASS_CONFIG.SCOREBOARD_SEARCH.DEBOUNCE_MS);
    });
    expect(params().get('q')).toBe('solved:120');
    expect(students()).toEqual(['Luis']);
  });

  it('discards a queued clear when the user navigates elsewhere', async () => {
    mount();
    type('solved:12');
    const element = input();
    await act(async () => {
      vi.advanceTimersByTime(CLASS_CONFIG.SCOREBOARD_SEARCH.DEBOUNCE_MS);
      Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, 'value')!.set!.call(
        element,
        ''
      );
      element.dispatchEvent(new InputEvent('input', { bubbles: true }));
      await navigate('/?q=solved%3A120&extra=destination');
    });
    await act(async () => {
      vi.advanceTimersByTime(300);
    });
    expect(element.value).toBe('solved:120');
    expect(params().get('q')).toBe('solved:120');
    expect(params().get('extra')).toBe('destination');
    expect(students()).toEqual(['Luis']);
  });

  it('applies Enter immediately and restores applied queries through history', async () => {
    mount(['/?q=solved%3A12']);
    type('solved:120');
    await act(async () => {
      input().dispatchEvent(
        new KeyboardEvent('keydown', { bubbles: true, key: 'Enter' })
      );
    });
    expect(params().get('q')).toBe('solved:120');
    expect(students()).toEqual(['Luis']);
    await act(async () => {
      await navigate('/?q=%40jorge');
    });
    expect(input().value).toBe('@jorge');
    expect(students()).toEqual(['Jorge']);
    await act(async () => {
      await navigate(-1);
    });
    expect(input().value).toBe('solved:120');
    await act(async () => {
      await navigate(1);
    });
    expect(input().value).toBe('@jorge');
  });

  it('keeps an explicitly selected problems tab when the URL contains q', () => {
    mount(['/?tab=problems&q=solved%3A12']);
    expect(container.querySelector('#tab-problems')?.getAttribute('aria-selected')).toBe(
      'true'
    );
    expect(input().value).toBe('solved:12');
  });

  it('does not write a pending query after unmounting', async () => {
    mount();
    type('solved:12');
    act(() => root.render(null));
    await act(async () => {
      vi.advanceTimersByTime(CLASS_CONFIG.SCOREBOARD_SEARCH.DEBOUNCE_MS);
    });
    expect(params().has('q')).toBe(false);
  });

  it('restores URL search when a class navigation remounts standings', async () => {
    mount(['/?class=basicos&q=solved%3A12']);
    type('solved:120');
    await act(async () => {
      await navigate('/?class=avanzados&q=solved%3A12');
    });
    await act(async () => {
      vi.advanceTimersByTime(CLASS_CONFIG.SCOREBOARD_SEARCH.DEBOUNCE_MS);
    });
    expect(input().value).toBe('solved:12');
    expect(params().get('class')).toBe('avanzados');
    expect(params().get('q')).toBe('solved:12');
  });

  it('decodes an initial query and applies it immediately', async () => {
    mount(['/?class=basicos&q=solved%3A12']);
    expect(input().value).toBe('solved:12');
    expect(students()).toEqual(['Jorge']);
    expect(container.querySelectorAll('.problem-name-text')).toHaveLength(2);
  });

  it('preserves invalid URL text and shows the parser feedback', async () => {
    mount(['/?q=solved%3Awrong']);
    expect(input().value).toBe('solved:wrong');
    expect(input().getAttribute('aria-invalid')).toBe('true');
    expect(container.querySelector('.scoreboard-search-issues')?.textContent).toContain(
      CLASS_LABELS.SCOREBOARD_SEARCH.SOLVED_ID_HINT
    );
  });

  it('debounces URL edits, replaces history, and deletes q immediately on clear', async () => {
    mount(['/?tab=problems&class=basicos', '/?class=basicos&extra=keep#anchor']);
    type('solved:12');
    await act(async () => {
      await vi.advanceTimersByTime(199);
    });
    expect(params().has('q')).toBe(false);
    await act(async () => {
      await vi.advanceTimersByTime(1);
    });
    expect(params().get('q')).toBe('solved:12');
    expect(params().get('extra')).toBe('keep');
    expect(location.hash).toBe('#anchor');
    type('');
    await act(async () => {});
    expect(params().has('q')).toBe(false);
    expect(students()).toHaveLength(2);
    await act(async () => {
      await navigate(-1);
    });
    expect(params().get('tab')).toBe('problems');
  });

  it('cancels an old timer when switching tabs and does not force standings from q', async () => {
    mount(['/?q=solved%3A12#anchor']);
    type('solved:120');
    await act(async () => {
      await container.querySelector<HTMLButtonElement>('#tab-problems')!.click();
    });
    await act(async () => {
      await vi.advanceTimersByTime(300);
    });
    expect(params().get('tab')).toBe('problems');
    expect(location.hash).toBe('#anchor');
    expect(params().get('q')).toBe('solved:12');
    await act(async () => {
      await navigate(-1);
    });
    expect(input().value).toBe('solved:12');
  });

  it('cancels pending work on POP even when the destination has the same q', async () => {
    mount(['/?q=solved%3A12&extra=first', '/?q=solved%3A12&extra=second']);
    type('solved:120');
    await act(async () => {
      await navigate(-1);
    });
    await act(async () => {
      await vi.advanceTimersByTime(300);
    });
    expect(input().value).toBe('solved:12');
    expect(params().get('q')).toBe('solved:12');
    expect(params().get('extra')).toBe('first');
  });

  it('preserves a newer draft while acknowledging its own URL replacement', async () => {
    mount();
    type('solved:12');
    const element = input();
    await act(async () => {
      vi.advanceTimersByTime(CLASS_CONFIG.SCOREBOARD_SEARCH.DEBOUNCE_MS);
      Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, 'value')!.set!.call(
        element,
        'solved:120'
      );
      element.dispatchEvent(new InputEvent('input', { bubbles: true }));
    });
    expect(element.value).toBe('solved:120');
    expect(params().get('q')).toBe('solved:12');
    await act(async () => {
      await vi.advanceTimersByTime(CLASS_CONFIG.SCOREBOARD_SEARCH.DEBOUNCE_MS);
    });
    expect(params().get('q')).toBe('solved:120');
    expect(students()).toEqual(['Luis']);
  });

  it('treats an external REPLACE as navigation and cancels the pending draft', async () => {
    mount(['/?q=solved%3A12']);
    type('solved:120');
    await act(async () => {
      await navigate('/?q=solved%3A12&extra=replaced', { replace: true });
    });
    await act(async () => {
      await vi.advanceTimersByTime(300);
    });
    expect(input().value).toBe('solved:12');
    expect(params().get('extra')).toBe('replaced');
  });
});
