import { dirname, resolve } from 'path';
import { fileURLToPath } from 'url';
import { defineConfig } from 'vite';
import { svelte } from '@sveltejs/vite-plugin-svelte';

const __dirname = dirname(fileURLToPath(import.meta.url));
const openCoreRoot = resolve(__dirname, '../..');
const webSrc = resolve(openCoreRoot, 'apps/web/src');

export default defineConfig({
  root: __dirname,
  plugins: [svelte()],
  resolve: {
    alias: {
      $web: webSrc,
    },
    dedupe: ['svelte'],
  },
  server: {
    host: '127.0.0.1',
    port: 4750,
    strictPort: true,
    fs: {
      allow: [openCoreRoot],
    },
  },
  test: {
    environment: 'node',
    include: ['src/**/*.test.js'],
  },
});
