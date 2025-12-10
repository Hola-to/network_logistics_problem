import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import path from "path";

export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "./src"),
      "@gen": path.resolve(__dirname, "./gen/ts"),
    },
  },
  optimizeDeps: {
    include: [
      "@bufbuild/protobuf",
      "@bufbuild/protobuf/codegenv2",
      "@bufbuild/protobuf/wkt",
      "@connectrpc/connect",
      "@connectrpc/connect-web",
    ],
  },
  server: {
    port: 3000,
    proxy: {
      "/logistics.gateway.v1.GatewayService": {
        target: "http://localhost:8080",
        changeOrigin: true,
      },
    },
  },
  build: {
    chunkSizeWarningLimit: 2000,
    rollupOptions: {
      output: {
        manualChunks(id) {
          if (id.includes("node_modules")) {
            return "vendor";
          }
        },
      },
    },
  },
});
