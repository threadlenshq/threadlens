import { dirname, resolve } from 'path';
import { fileURLToPath } from 'url';
import { defineConfig } from 'vite';
import { svelte } from '@sveltejs/vite-plugin-svelte';
import { configDefaults } from 'vitest/config';


const __dirname = dirname(fileURLToPath(import.meta.url));
const repoRoot = resolve(__dirname, '../..');
const apiProxyTarget = process.env.VITE_API_PROXY_TARGET || 'http://localhost:4747';

export default defineConfig({
  root: __dirname,
  plugins: [svelte({
    dynamicCompileOptions() {
      // Force client compilation in Vitest so Svelte component tests render correctly.
      return process.env.VITEST ? { generate: 'client' } : {};
    },
  })],
  test: {
    root: repoRoot,
    environmentMatchGlobs: [['test/frontend/**/*.test.js', 'jsdom']],
    exclude: [...configDefaults.exclude, '.worktrees/**', 'db/**'],
    alias: [
      {
        find: /^svelte$/,
        replacement: resolve(__dirname, 'node_modules/svelte/src/index-client.js'),
      },
      {
        find: /^svelte\/internal\/client$/,
        replacement: resolve(__dirname, 'node_modules/svelte/src/internal/client/index.js'),
      },
      {
        find: /^svelte\/internal\/server$/,
        replacement: resolve(__dirname, 'node_modules/svelte/src/internal/server/index.js'),
      },
      {
        find: /^svelte\/internal\/disclose-version$/,
        replacement: resolve(__dirname, 'node_modules/svelte/src/internal/disclose-version.js'),
      },
      {
        find: /^svelte\/internal\/flags\/legacy$/,
        replacement: resolve(__dirname, 'node_modules/svelte/src/internal/flags/legacy.js'),
      },
    ],
  },
  server: {
    port: 4748,
    proxy: {
      '/api': { target: apiProxyTarget, changeOrigin: true },
    },
    watch: {
      ignored: ['**/*.db', '**/*.db-shm', '**/*.db-wal'],
    },
  },
});
