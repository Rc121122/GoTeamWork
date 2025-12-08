import { defineConfig } from 'vite';

const port = Number(process.env.WAILS_VITE_PORT || 5173);

export default defineConfig({
  server: {
    port,
  },
  build: {
    outDir: 'dist',
  },
});
