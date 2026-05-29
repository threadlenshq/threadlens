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
  plugins: [svelte()],
  test: {
    root: repoRoot,
    environmentMatchGlobs: [['test/frontend/**/*.test.js', 'jsdom']],
    exclude: [...configDefaults.exclude, '.worktrees/**'],
    alias: [
      {
        find: /^svelte$/,
        replacement: resolve(__dirname, '../../node_modules/svelte/src/index-client.js'),
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
