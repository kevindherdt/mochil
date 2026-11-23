// playwright.config.ts — the L8 gate (plan docs/rewrite-plan.md §4.2): three engines, one phone viewport, and the
// layout laws that only a real browser can falsify. The Firefox dock incident is the reason this exists: `100dvh`
// and a fixed-looking shell behaved in Chromium and not in Firefox Android, and nothing in the suite could see it.
//
// Deliberately hermetic: it serves the built app from `vite preview` and fulfils `/api/**` from the repo's own
// contract fixtures, so the gate runs with no server, no database and no credentials — a CI job that needs the
// author's library would never run on a pull request.
import { defineConfig, devices } from "@playwright/test";

const PORT = 4173;

export default defineConfig({
  testDir: "./e2e",
  fullyParallel: true,
  forbidOnly: true,
  retries: 0,
  reporter: [["list"]],
  use: {
    baseURL: `http://127.0.0.1:${PORT}`,
    trace: "off",
  },
  projects: [
    { name: "chromium", use: { ...devices["Desktop Chrome"], viewport: { width: 390, height: 844 } } },
    { name: "firefox", use: { ...devices["Desktop Firefox"], viewport: { width: 390, height: 844 } } },
    { name: "webkit", use: { ...devices["Desktop Safari"], viewport: { width: 390, height: 844 } } },
  ],
  webServer: {
    // `--host 127.0.0.1`: vite preview otherwise binds ::1 only, and the baseURL below is IPv4.
    command: `npx vite preview --host 127.0.0.1 --port ${PORT} --strictPort`,
    url: `http://127.0.0.1:${PORT}/`,
    reuseExistingServer: !process.env.CI,
    timeout: 60_000,
  },
});
