import { watchForUpdates } from "./app-update.js?v=1";

// Registers the service worker once the page has loaded so it never competes
// with first paint. Failure is harmless: the app works the same without it.
export function registerServiceWorker(win = window) {
  if (!("serviceWorker" in win.navigator)) return;
  win.addEventListener("load", () => {
    const worker = win.navigator.serviceWorker;
    worker.register("/sw.js", { scope: "/" }).catch(() => {});
    // A new worker takes over after its own activate, and the page it just replaced is still running
    // the old script. Reloading once here means an upgrade is felt immediately instead of on the next
    // launch, which is why a fixed version could look like it had no effect at all.
    worker.addEventListener?.("controllerchange", () => {
      if (win.__mochilReloadedForWorker) return;
      win.__mochilReloadedForWorker = true;
      win.location.reload();
    });
    watchForUpdates(win);
  }, { once: true });
}
