import { readFile } from 'node:fs/promises';
import path from 'node:path';
import type { Plugin } from 'vite';
import { CLASS_CONFIG } from '../src/domain/class/constants';

export function renderLlms(template: string, exportsUrl: string): string {
  const baseUrl = exportsUrl.replace(/\/+$/, '');
  return template.replaceAll('${VITE_EXPORTS_BASE_URL}', baseUrl);
}

export function llmsGuide(): Plugin {
  let templatePath: string;
  let exportsUrl: string;

  const render = async () => renderLlms(await readFile(templatePath, 'utf8'), exportsUrl);

  return {
    name: 'llms-guide',
    configResolved(config) {
      templatePath = path.resolve(config.root, 'llms.txt.template');
      exportsUrl =
        config.command === 'serve'
          ? CLASS_CONFIG.API.LOCAL_EXPORTS_PATH
          : config.env.VITE_EXPORTS_BASE_URL ||
            CLASS_CONFIG.API.DEFAULT_AZURE_STORAGE_URL;
    },
    async generateBundle() {
      this.emitFile({ type: 'asset', fileName: 'llms.txt', source: await render() });
    },
    configureServer(server) {
      server.middlewares.use(async (request, response, next) => {
        if (
          request.url?.split('?')[0] !== '/llms.txt' ||
          (request.method !== 'GET' && request.method !== 'HEAD')
        ) {
          next();
          return;
        }
        try {
          const content = await render();
          response.setHeader('Content-Type', 'text/plain; charset=utf-8');
          response.end(request.method === 'HEAD' ? undefined : content);
        } catch (error) {
          next(error);
        }
      });
    },
  };
}
