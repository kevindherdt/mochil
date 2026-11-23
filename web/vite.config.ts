import react from "@vitejs/plugin-react";
import { defineConfig } from "vitest/config";

// Phase 1 (plan docs/rewrite-plan.md §8): builds to internal/web/static/next/ (gitignored), the Go server's
// The app is served from the root path since cutover (the legacy shell moved aside), so base is "/".
// `defineConfig` comes from vitest/config (a superset of vite's) so the same file configures `vite build`/`vite
// dev` and `vitest run` without a second config file to keep in sync.
export default defineConfig({
  plugins: [react()],
  base: "/",
  build: {
    outDir: "../internal/web/static/next",
    emptyOutDir: true,
  },
  test: {
    environment: "jsdom",
    include: ["tests/**/*.test.{ts,tsx}"],
  },
});
