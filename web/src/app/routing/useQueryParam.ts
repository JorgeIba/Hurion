import {
  useLocation,
  useNavigate,
  useSearchParams,
  type NavigateOptions,
} from 'react-router';
import { ActiveTab, sanitizeTab, sanitizeClassCode, CLASS_CONFIG } from '@/domain/class';
import { URL_PARAMS } from '@/app/routing/navigation';

/** Reads validated URL state and navigates without dropping other parameters. */
export function useQueryParam<T>(
  key: string,
  parse: (raw: string | null) => T,
  defaultValue: T
): [T, (newValue: T, options?: NavigateOptions) => void] {
  const [params] = useSearchParams();
  const location = useLocation();
  const navigate = useNavigate();
  const setValue = (newValue: T, options?: NavigateOptions) => {
    const next = new URLSearchParams(params);
    if (newValue === defaultValue) next.delete(key);
    else next.set(key, String(newValue));
    if (next.toString() !== params.toString()) {
      navigate(
        {
          pathname: location.pathname,
          search: next.toString(),
          hash: location.hash,
        },
        options
      );
    }
  };
  return [parse(params.get(key)), setValue];
}

/**
 * Custom React Hook: useTabParam
 * Synchronizes the active scoreboard tab ('standings' | 'problems') with the `?tab=` URL parameter.
 */
export function useTabParam(
  defaultTab: ActiveTab = CLASS_CONFIG.DEFAULT_TAB
): [ActiveTab, (newTab: ActiveTab) => void] {
  return useQueryParam<ActiveTab>(
    URL_PARAMS.TAB,
    (raw) => sanitizeTab(raw, defaultTab),
    defaultTab
  );
}

/**
 * Custom React Hook: useClassCodeParam
 * Synchronizes the active class code (e.g. "basicos", "avanzados") with the `?class=` URL parameter.
 */
export function useClassCodeParam(
  defaultCode: string = CLASS_CONFIG.DEFAULT_CLASS_CODE
): [string, (newCode: string) => void] {
  return useQueryParam<string>(
    URL_PARAMS.CLASS,
    (raw) => sanitizeClassCode(raw) || defaultCode,
    defaultCode
  );
}
