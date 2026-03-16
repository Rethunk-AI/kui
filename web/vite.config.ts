import { defineConfig } from "vite";

// noVNC's browser.js uses top-level await in CJS; replace with sync fallback for build
function novncTopLevelAwaitFix() {
  return {
    name: "novnc-top-level-await-fix",
    transform(code: string, id: string) {
      if (id.includes("@novnc/novnc") && id.endsWith("browser.js")) {
        return code.replace(
          /exports\.supportsWebCodecsH264Decode\s*=\s*supportsWebCodecsH264Decode\s*=\s*await\s+_checkWebCodecsH264DecodeSupport\(\)/,
          "exports.supportsWebCodecsH264Decode = supportsWebCodecsH264Decode = false"
        );
      }
    },
  };
}

export default defineConfig({
  root: ".",
  publicDir: "public",
  plugins: [novncTopLevelAwaitFix()],
  build: {
    outDir: "dist",
    emptyOutDir: true,
    target: "esnext",
    rollupOptions: {
      output: {
        manualChunks(id) {
          if (id.includes("node_modules/winbox") || id.includes("node_modules/@novnc") || id.includes("node_modules/xterm")) {
            return "console-vendors";
          }
        },
      },
    },
  },
  optimizeDeps: {
    esbuildOptions: {
      target: "esnext",
    },
  },
  server: {
    port: 5173,
    host: true,
    proxy: {
      "/api": {
        target: "http://127.0.0.1:8080",
        changeOrigin: true,
        ws: true,
      },
      "/api/events": {
        target: "http://127.0.0.1:8080",
        changeOrigin: true,
        ws: true,
      },
    },
  },
});
