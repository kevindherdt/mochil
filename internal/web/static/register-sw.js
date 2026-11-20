import { watchForUpdates } from "./app-update.js?v=1";

// Registers the service worker once the page has loaded so it never competes with first paint. Failure is
// harmless: the app works the same without it.
//
// It never reloads the page by itself. A worker taking over fires controllerchange, and answering that by
// reloading is a page that paints, then paints again — whether it is a first install claiming the page or a
// new build replacing a live worker. A new build is offered instead: watchForUpdates puts "A new version is
// ready - Reload" on screen, and the reader decides when to lose what they were doing.
export function registerServiceWorker(win = window) {
  if (!("serviceWorker" in win.navigator)) return;
  win.addEventListener("load", () => {
    win.navigator.serviceWorker.register("/sw.js", { scope: "/" }).catch(() => {});
    watchForUpdates(win);
  }, { once: true });
}
