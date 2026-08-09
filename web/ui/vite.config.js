import { defineConfig } from "vite";
import vue from "@vitejs/plugin-vue";
import { resolve } from "node:path";

export default defineConfig({
  plugins: [vue()],
  base: "/",
  build: {
    outDir: resolve(__dirname, "../static"),
    emptyOutDir: true,
    assetsDir: "assets",
  },
  server: {
    port: 5173,
    proxy: {
      "/api": {
        target: "http://127.0.0.1:3080",
        changeOrigin: true,
        ws: true,
      },
    },
  },
});
