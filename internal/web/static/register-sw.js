import { watchForUpdates } from "./app-update.js?v=1";

// Registers the service worker once the page has loaded so it never competes
// with first paint. Failure is harmless: the app works the same without it.
export function registerServiceWorker(win = window) {
  if (!("serviceWorker" in win.navigator)) return;
  win.addEventListener("load", () => {
    win.navigator.serviceWorker.register("/sw.js", { scope: "/" }).catch(() => {});
    watchForUpdates(win);
  }, { once: true });
}
