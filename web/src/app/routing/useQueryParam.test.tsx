// @vitest-environment jsdom
import { transferableAbortController } from 'node:util';
import { act, StrictMode } from 'react';
import { createRoot, type Root } from 'react-dom/client';
import { createMemoryRouter, useLocation, useNavigate } from 'react-router';
import { RouterProvider } from 'react-router/dom';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import {
  useClassCodeParam,
  useQueryParam,
  useTabParam,
} from '@/app/routing/useQueryParam';
import type { ActiveTab } from '@/domain/class';
import { URL_PARAMS } from '@/app/routing/navigation';

let container: HTMLDivElement;
let root: Root;
let location: ReturnType<typeof useLocation>;
let navigate: ReturnType<typeof useNavigate>;
let tab: ReturnType<typeof useTabParam>;
let classCode: ReturnType<typeof useClassCodeParam>;
let query: ReturnType<typeof useQueryParam<string>>;

function Probe({ defaultTab }: { defaultTab: ActiveTab }) {
  location = useLocation();
  navigate = useNavigate();
  tab = useTabParam(defaultTab);
  classCode = useClassCodeParam();
  // Exercise raw parameter transport without connecting it to the search UI.
  query = useQueryParam(URL_PARAMS.QUERY, (raw) => raw ?? '', '');
  return null;
}

function mount(url = '/', defaultTab: ActiveTab = 'standings') {
  const router = createMemoryRouter(
    [{ path: '*', element: <Probe defaultTab={defaultTab} /> }],
    { initialEntries: [url] }
  );
  act(() =>
    root.render(
      <StrictMode>
        <RouterProvider router={router} />
      </StrictMode>
    )
  );
}

beforeEach(() => {
  vi.stubGlobal('IS_REACT_ACT_ENVIRONMENT', true);
  // The data router uses Node's Request, which requires a compatible AbortSignal.
  vi.stubGlobal('AbortController', transferableAbortController);
  container = document.createElement('div');
  document.body.append(container);
  root = createRoot(container);
});

afterEach(() => {
  act(() => root.unmount());
  container.remove();
  vi.unstubAllGlobals();
});

describe('URL parameter navigation', () => {
  it.each(['/', '/?tab=invalid&class=%2F%2F'])(
    'defaults missing or invalid values in %s',
    (url) => {
      mount(url, 'problems');
      expect(tab[0]).toBe('problems');
      expect(classCode[0]).toBe('basicos');
      expect(query[0]).toBe('');
      // Reading applies defaults without rewriting the incoming URL.
      expect(location.search).toBe(new URL(url, 'http://localhost').search);
    }
  );

  it('accepts an explicit tab and sanitizes the class code', () => {
    mount('/?tab=%20STANDINGS%20&class=%20avanzados_2024%2F%20', 'problems');
    expect(tab[0]).toBe('standings');
    expect(classCode[0]).toBe('avanzados_2024');
  });

  it('preserves other parameters, pathname and hash while navigating tabs', async () => {
    mount('/score?class=avanzados&q=solved%3A304+%2B+solved%3A305&extra=keep#anchor');
    await act(async () => {
      await tab[1]('problems');
    });
    const params = new URLSearchParams(location.search);
    expect(params.get(URL_PARAMS.TAB)).toBe('problems');
    expect(params.get(URL_PARAMS.CLASS)).toBe('avanzados');
    expect(query[0]).toBe('solved:304 + solved:305');
    expect(params.get('extra')).toBe('keep');
    expect(location.pathname).toBe('/score');
    expect(location.hash).toBe('#anchor');
    await act(async () => {
      await navigate(-1);
    });
    expect(tab[0]).toBe('standings');
    await act(async () => {
      await navigate(1);
    });
    expect(tab[0]).toBe('problems');
  });

  it('removes only the parameter reset to its default', async () => {
    mount('/?class=avanzados&tab=problems&q=%40jorge&extra=keep#anchor');
    await act(async () => {
      await classCode[1]('basicos');
    });
    const params = new URLSearchParams(location.search);
    expect(params.has(URL_PARAMS.CLASS)).toBe(false);
    expect(classCode[0]).toBe('basicos');
    expect(params.get(URL_PARAMS.TAB)).toBe('problems');
    expect(params.get(URL_PARAMS.QUERY)).toBe('@jorge');
    expect(params.get('extra')).toBe('keep');
    expect(location.hash).toBe('#anchor');
  });

  it('round-trips parameter text with reserved characters through history', async () => {
    mount('/?class=basicos#anchor');
    const value = '@jorge + t:grafos & #304';
    await act(async () => {
      await query[1](value);
    });
    expect(query[0]).toBe(value);
    expect(new URLSearchParams(location.search).get(URL_PARAMS.QUERY)).toBe(value);
    await act(async () => {
      await navigate(-1);
    });
    expect(query[0]).toBe('');
    await act(async () => {
      await navigate(1);
    });
    expect(query[0]).toBe(value);
  });

  it('does not add a history entry when the serialized parameter is unchanged', async () => {
    mount('/?tab=problems');
    const key = location.key;
    await act(async () => {
      await tab[1]('problems');
    });
    expect(location.key).toBe(key);
  });
});
