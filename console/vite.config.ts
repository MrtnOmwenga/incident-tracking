import { defineConfig } from 'vitest/config';
import vue from '@vitejs/plugin-vue';

// The console is served by Lighthouse at /console/ and embedded in its binary.
export default defineConfig({
  plugins: [vue()],
  base: '/console/',
  build: {
    outDir: '../internal/web/console',
    emptyOutDir: false, // keep the committed placeholder
    modulePreload: { polyfill: false }, // no inline script: the CSP forbids it
  },
  server: {
    proxy: { '/api': 'http://localhost:8080', '/auth': 'http://localhost:8080', '/static': 'http://localhost:8080' },
  },
  test: { environment: 'jsdom', globals: true, include: ['src/**/*.test.ts'] },
});
