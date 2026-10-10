import { fileURLToPath, URL } from 'node:url'

import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'
import { defineConfig } from 'vitest/config'

export default defineConfig({
  plugins: [react(), tailwindcss()],
  server: {
    watch: {
      usePolling: true,
      interval: 100,
    },
  },
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
      react: fileURLToPath(new URL('./node_modules/react', import.meta.url)),
      'react-dom': fileURLToPath(new URL('./node_modules/react-dom', import.meta.url)),
    },
    dedupe: ['react', 'react-dom'],
  },
  test: {
    server: {
      deps: {
        inline: ['@testing-library/react', 'react-dom'],
      },
    },
    projects: [
      {
        extends: true,
        test: {
          name: 'dom',
          environment: 'jsdom',
          globals: true,
          setupFiles: './src/test/setup.ts',
          include: ['src/**/*.{test,spec}.{jsx,tsx}', 'src/lib/auth/msal-auth-client.test.ts'],
        },
      },
      {
        extends: true,
        test: {
          name: 'unit',
          environment: 'node',
          setupFiles: [],
          include: ['src/**/*.{test,spec}.{js,ts}'],
          exclude: ['src/lib/auth/msal-auth-client.test.ts'],
        },
      },
    ],
  },
})
