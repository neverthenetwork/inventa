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
  resolve: {
    // Three.js is pulled by aframe (as super-three), 3d-force-graph, and
    // aframe-extras — dedupe ensures a single instance so instanceof checks pass.
    dedupe: ['three'],
  },
});
