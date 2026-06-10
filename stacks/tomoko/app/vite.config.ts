import { defineConfig } from 'vite';
import { svelte } from '@sveltejs/vite-plugin-svelte';

export default defineConfig({
  plugins: [svelte()],
  server: {
    // Local dev against a locally-running sophon (see stacks/sophon/app).
    proxy: {
      '/api': 'http://localhost:8099',
    },
  },
});
