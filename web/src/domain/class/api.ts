import axios from 'axios';
import { ClassData } from './types';
import { CLASS_CONFIG } from './constants';

/**
 * Resolves the appropriate Base URL for fetching class data based on the runtime environment:
 * 1. Discord Activity (`frame_id` present in URL): Uses relative `/exports` for Discord Cloud Proxy.
 * 2. Local Development (`localhost` or `127.0.0.1`): Uses relative `/exports` for the Go local server.
 * 3. Standalone Web Browser (Azure Static Web Apps): Uses direct Azure Blob Storage with CORS.
 */
function getBaseUrl(): string {
  if (typeof window === 'undefined') {
    return CLASS_CONFIG.API.LOCAL_EXPORTS_PATH;
  }

  const isDiscord = new URLSearchParams(window.location.search).has('frame_id');
  const isLocal = window.location.hostname === 'localhost' || window.location.hostname === '127.0.0.1';

  if (isDiscord || isLocal) {
    return CLASS_CONFIG.API.LOCAL_EXPORTS_PATH;
  }

  const configuredUrl =
    import.meta.env.VITE_EXPORTS_BASE_URL || CLASS_CONFIG.API.DEFAULT_AZURE_STORAGE_URL;
  return configuredUrl.replace(/\/+$/, '');
}

/**
 * Isolated Axios client instance configured with environment-aware Base URL.
 */
export const apiClient = axios.create({
  baseURL: getBaseUrl(),
  timeout: CLASS_CONFIG.API.TIMEOUT_MS,
  headers: {
    Accept: 'application/json',
  },
});

/**
 * Domain Service: classApi
 * Provides typed methods for retrieving class export data.
 */
export const classApi = {
  async fetchClass(classCode: string, signal?: AbortSignal): Promise<ClassData> {
    const response = await apiClient.get<ClassData>(`/${encodeURIComponent(classCode)}_score.json`, {
      signal,
    });
    return response.data;
  },
};
