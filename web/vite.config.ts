import { defineConfig } from 'vite';
import { resolve } from 'path';

export default defineConfig({
  root: '.',
  build: {
    outDir: '../cmd/inventa/web-dist',
    rollupOptions: {
      input: {
        main: resolve(__dirname, 'index.html'),
        threed: resolve(__dirname, '3dindex.html'),
        vr: resolve(__dirname, 'vrindex.html'),
      },
    },
  },
});
