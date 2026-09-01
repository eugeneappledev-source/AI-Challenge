import { defineConfig, loadEnv } from "vite";
import react from "@vitejs/plugin-react";

export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, "../", "");
  const authorization = env.APP_ACCESS_TOKEN
    ? { Authorization: `Bearer ${env.APP_ACCESS_TOKEN}` }
    : undefined;

  return {
    plugins: [react()],
    envDir: "../",
    server: {
      proxy: {
        "/web-api": {
          target: "http://localhost:8080",
          changeOrigin: true,
          headers: authorization,
          rewrite: (path) => path.replace(/^\/web-api/, "/v1"),
        },
      },
    },
  };
});
