// @vitest-environment jsdom
import { act } from 'react';
import { createRoot, type Root } from 'react-dom/client';
import {
  afterEach,
  beforeEach,
  describe,
  expect,
  it,
  type MockInstance,
  vi,
} from 'vitest';
import { StandingsView } from '@/components/StandingsView';
import * as search from '@/hooks/useScoreboardSearch';
import { priorityPresentationConfig } from '@/presentation/priorityConfig';

vi.mock('@/components/ScoreboardTable', () => ({
  ScoreboardTable: () => <div>Scoreboard</div>,
}));

let container: HTMLDivElement;
let root: Root;
let input: HTMLInputElement;
let resolve: MockInstance<typeof search.useScoreboardSearch>;

function type(value: string, isComposing = false) {
  act(() => {
    // Use the native setter so React observes an actual input event.
    Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, 'value')?.set?.call(
      input,
      value
    );
    input.dispatchEvent(new InputEvent('input', { bubbles: true, isComposing }));
  });
}

function advance(ms: number) {
  act(() => vi.advanceTimersByTime(ms));
}

function key(key: string, isComposing = false) {
  act(() => {
    input.dispatchEvent(
      new KeyboardEvent('keydown', { bubbles: true, key, isComposing })
    );
  });
}

function appliedQuery() {
  return resolve.mock.calls.at(-1)?.[0];
}

beforeEach(() => {
  vi.stubGlobal('IS_REACT_ACT_ENVIRONMENT', true);
  vi.useFakeTimers();
  resolve = vi.spyOn(search, 'useScoreboardSearch');
  container = document.createElement('div');
  document.body.append(container);
  root = createRoot(container);
  act(() =>
    root.render(
      <StandingsView
        priorityPresentation={priorityPresentationConfig.defaults}
        problems={[]}
        users={[]}
      />
    )
  );
  act(() =>
    container.querySelector<HTMLButtonElement>('.scoreboard-search-trigger')?.click()
  );
  input = container.querySelector('input')!;
});

afterEach(() => {
  act(() => root.unmount());
  container.remove();
  vi.useRealTimers();
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
});

describe('scoreboard search interaction', () => {
  it('updates text immediately and applies only the last query after a full pause', () => {
    type('jor');
    expect(input.value).toBe('jor');
    advance(150);
    type('jorge');
    advance(199);
    expect(appliedQuery()).toBe('');
    advance(1);
    expect(appliedQuery()).toBe('jorge');
    expect(resolve.mock.calls.map(([query]) => query)).not.toContain('jor');
  });

  it('applies Enter immediately and cancels the older pending query', () => {
    type('#wrong');
    key('Enter');
    expect(appliedQuery()).toBe('#wrong');
    expect(input.getAttribute('aria-invalid')).toBe('true');
    type('luis');
    advance(200);
    expect(appliedQuery()).toBe('luis');
  });

  it.each(['button', 'Escape', 'delete'])(
    'clears immediately with %s and cancels pending work',
    (method) => {
      type('jorge');
      key('Enter');
      type('#pending');
      if (method === 'button') {
        act(() =>
          container.querySelector<HTMLButtonElement>('.scoreboard-search-clear')?.click()
        );
      } else if (method === 'Escape') {
        key('Escape');
      } else {
        type('');
      }
      expect(input.value).toBe('');
      expect(appliedQuery()).toBe('');
      advance(300);
      expect(appliedQuery()).toBe('');
      expect(document.activeElement).toBe(input);
    }
  );

  it('waits for composition to finish and ignores its Enter', () => {
    type('jor');
    type('jorge', true);
    key('Enter', true);
    advance(300);
    expect(appliedQuery()).toBe('');
    act(() =>
      input.dispatchEvent(new CompositionEvent('compositionend', { bubbles: true }))
    );
    advance(199);
    expect(appliedQuery()).toBe('');
    advance(1);
    expect(appliedQuery()).toBe('jorge');
  });

  it('hides stale errors while editing and keeps the feedback region mounted', () => {
    const feedback = container.querySelector('[role="status"]');
    type('#wrong');
    advance(200);
    expect(feedback?.textContent).toContain('#wrong');
    type('jorge');
    expect(input.getAttribute('aria-invalid')).toBe('false');
    expect(feedback?.textContent).toBe('');
    expect(container.querySelector('[role="status"]')).toBe(feedback);
  });

  it('opens the tooltip with focus or touch and dismisses it with Escape or blur', () => {
    const help = container.querySelector<HTMLParagraphElement>(
      '.scoreboard-search-help'
    )!;
    const button = container.querySelector<HTMLButtonElement>(
      '.scoreboard-search-help-trigger'
    )!;
    expect(help.hidden).toBe(true);
    act(() => button.focus());
    expect(help.hidden).toBe(false);
    expect(button.getAttribute('aria-describedby')).toBe(help.id);
    expect(help.getAttribute('role')).toBe('tooltip');
    act(() =>
      button.dispatchEvent(new KeyboardEvent('keydown', { bubbles: true, key: 'Escape' }))
    );
    expect(help.hidden).toBe(true);
    expect(document.activeElement).toBe(button);
    act(() => button.click());
    expect(help.hidden).toBe(false);
    act(() => input.focus());
    expect(help.hidden).toBe(true);
  });

  it('dismisses a tooltip opened without button focus when pressing outside', () => {
    const help = container.querySelector<HTMLParagraphElement>('[role="tooltip"]')!;
    const button = container.querySelector<HTMLButtonElement>(
      '.scoreboard-search-help-trigger'
    )!;
    act(() => button.click());
    expect(document.activeElement).not.toBe(button);
    expect(help.hidden).toBe(false);
    act(() => help.dispatchEvent(new MouseEvent('pointerdown', { bubbles: true })));
    expect(help.hidden).toBe(false);
    act(() => button.dispatchEvent(new MouseEvent('pointerdown', { bubbles: true })));
    expect(help.hidden).toBe(false);
    act(() => container.dispatchEvent(new MouseEvent('pointerdown', { bubbles: true })));
    expect(help.hidden).toBe(true);
  });

  it('keeps the tooltip open when moving the pointer from its trigger to its text', () => {
    const anchor = container.querySelector('.scoreboard-search-help-anchor')!;
    const help = container.querySelector<HTMLParagraphElement>('[role="tooltip"]')!;
    const button = anchor.querySelector('button')!;
    act(() => button.dispatchEvent(new MouseEvent('mouseover', { bubbles: true })));
    expect(help.hidden).toBe(false);
    act(() =>
      button.dispatchEvent(
        new MouseEvent('mouseout', { bubbles: true, relatedTarget: help })
      )
    );
    expect(help.hidden).toBe(false);
    act(() =>
      anchor.dispatchEvent(
        new MouseEvent('mouseout', { bubbles: true, relatedTarget: document.body })
      )
    );
    expect(help.hidden).toBe(true);
  });

  it('clears on the first Escape and closes on the second, restoring trigger focus', () => {
    type('jorge');
    key('Escape');
    expect(input.hidden).toBe(false);
    key('Escape');
    expect(input.hidden).toBe(true);
    expect(document.activeElement).toBe(
      container.querySelector('.scoreboard-search-trigger')
    );
  });
});
