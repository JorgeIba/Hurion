import { defineConfig } from 'vitest/config';
import react from '@vitejs/plugin-react';
import path from 'path';
import { llmsGuide } from './build/llms';

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [react(), llmsGuide()],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
  build: {
    outDir: 'dist',
    emptyOutDir: true,
  },
  server: {
    port: 3000,
    proxy: {
      '/exports': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
    },
  },
  test: {
    poolOptions: {
      threads: {
        maxThreads: 2,
        minThreads: 1,
      },
    },
  },
});
