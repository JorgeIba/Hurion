// @vitest-environment node
import { readFileSync } from 'node:fs';
import { mkdtemp, readFile, rm, writeFile } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import path from 'node:path';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { build, createServer } from 'vite';
import { CLASS_CONFIG } from '../src/domain/class/constants';
import { llmsGuide, renderLlms } from './llms';

const template = readFileSync(new URL('../llms.txt.template', import.meta.url), 'utf8');
const testUrl = 'https://template-test.invalid/exports';
const temporaryRoots: string[] = [];

async function createFixture(): Promise<string> {
  const root = await mkdtemp(path.join(tmpdir(), 'hurion-llms-'));
  temporaryRoots.push(root);
  await writeFile(path.join(root, 'llms.txt.template'), template);
  await writeFile(path.join(root, 'index.html'), '<html><body>Test</body></html>');
  return root;
}

afterEach(async () => {
  vi.unstubAllEnvs();
  await Promise.all(
    temporaryRoots.splice(0).map((root) => rm(root, { recursive: true }))
  );
});

describe('AI agent guide', () => {
  it('renders every discovery and schema URL with no unresolved placeholder', () => {
    const guide = renderLlms(template, `${testUrl}///`);
    for (const filename of [
      'classes.json',
      '<classCode>_score.json',
      'problem-add-batch.json',
      'problem-edit-batch.json',
      'ac-add-batch.json',
    ]) {
      expect(guide).toContain(`${testUrl}/${filename}`);
    }
    expect(guide).not.toContain('${VITE_EXPORTS_BASE_URL}');
    expect(guide).not.toContain(CLASS_CONFIG.API.DEFAULT_AZURE_STORAGE_URL);
  });

  it('emits llms.txt during a build using the URL from the Vite environment', async () => {
    vi.stubEnv('VITE_EXPORTS_BASE_URL', undefined);
    const root = await createFixture();
    await writeFile(
      path.join(root, '.env.production'),
      `VITE_EXPORTS_BASE_URL=${testUrl}/`
    );
    await build({ root, configFile: false, plugins: [llmsGuide()], logLevel: 'silent' });

    const guide = await readFile(path.join(root, 'dist/llms.txt'), 'utf8');
    expect(guide).toBe(renderLlms(template, testUrl));
  });

  it('uses the frontend default when no URL is configured for the build', async () => {
    vi.stubEnv('VITE_EXPORTS_BASE_URL', undefined);
    const root = await createFixture();
    await build({ root, configFile: false, plugins: [llmsGuide()], logLevel: 'silent' });

    const guide = await readFile(path.join(root, 'dist/llms.txt'), 'utf8');
    expect(guide).toBe(renderLlms(template, CLASS_CONFIG.API.DEFAULT_AZURE_STORAGE_URL));
  });

  it('serves the local exports URL during development even with a production URL set', async () => {
    vi.stubEnv('VITE_EXPORTS_BASE_URL', testUrl);
    const root = await createFixture();
    const server = await createServer({
      root,
      configFile: false,
      plugins: [llmsGuide()],
      logLevel: 'silent',
      server: { host: '127.0.0.1', port: 0 },
    });
    try {
      await server.listen();
      const address = server.httpServer!.address();
      if (!address || typeof address === 'string') throw new Error('No dev server port');
      const response = await fetch(`http://127.0.0.1:${address.port}/llms.txt`);
      expect(response.status).toBe(200);
      expect(response.headers.get('content-type')).toBe('text/plain; charset=utf-8');
      expect(await response.text()).toBe(renderLlms(template, '/exports'));
    } finally {
      await server.close();
    }
  });
});
