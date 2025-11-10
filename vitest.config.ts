import { defineConfig, mergeConfig } from "vitest/config";

import viteConfig from "./vite.config";

export default mergeConfig(
  viteConfig,
  defineConfig({
    test: {
      globals: true,
      environment: "jsdom",
      setupFiles: ["./src/test/setup.ts"],
      include: ["src/**/*.{test,spec}.{ts,tsx}"],
      coverage: {
        provider: "v8",
        reporter: ["text", "json", "html"],
        exclude: [
          "node_modules/",
          "dist/",
          "src/test/",
          "**/*.config.ts",
          "**/*.d.ts",
          "src/main.tsx",
          "src/lib/api/v1.d.ts",
        ],
      },
      exclude: ["node_modules", "dist", ".git", ".cache", ".direnv", ".tmp"],
    },
  })
);
