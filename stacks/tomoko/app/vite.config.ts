import { defineConfig } from 'vite';
import { svelte } from '@sveltejs/vite-plugin-svelte';
import tailwindcss from '@tailwindcss/vite';

export default defineConfig({
  plugins: [svelte(), tailwindcss()],
  server: {
    // Local dev against a locally-running sophon (see stacks/sophon/app).
    proxy: {
      '/api': 'http://localhost:8099',
    },
  },
  preview: {
    proxy: {
      '/api': 'http://localhost:8099',
    },
  },
});
