import { fileURLToPath } from 'node:url';
import react from '@vitejs/plugin-react';
import { defineConfig } from 'vitest/config';

// During development the Vite server proxies /api to the Go backend, so the
// browser sees a single origin (same as in production) and no CORS is needed.
const apiTarget = process.env.API_PROXY_TARGET ?? 'http://localhost:8080';

const page = (path: string) => fileURLToPath(new URL(path, import.meta.url));

export default defineConfig({
  plugins: [react()],
  server: {
    port: 5173,
    proxy: {
      '/api': apiTarget,
      '/healthz': apiTarget,
    },
  },
  build: {
    // Two pages: the calculator, and the API docs (served at /docs/).
    rollupOptions: {
      input: {
        calculator: page('./index.html'),
        docs: page('./docs/index.html'),
      },
    },
    // Swagger UI is a single ~1.4 MB bundle, loaded only by the docs page.
    chunkSizeWarningLimit: 1600,
  },
  test: {
    environment: 'jsdom',
    setupFiles: ['./src/test/setup.ts'],
    coverage: {
      provider: 'v8',
      include: ['src/**/*.{ts,tsx}'],
      // Entry points are bootstrap wiring; types.ts and *.d.ts hold only types.
      exclude: [
        'src/main.tsx',
        'src/docs/main.ts',
        'src/**/*.d.ts',
        'src/api/types.ts',
        'src/test/**',
        'src/**/*.test.{ts,tsx}',
      ],
      reporter: ['text', 'html', 'json-summary'],
    },
  },
});
