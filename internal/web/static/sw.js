// Mochil service worker: makes the app shell open instantly and explains
// itself when the server can't be reached. It never caches API responses, book
// files or anything personal.
const SHELL = "mochil-shell-v1";
const ASSETS = "mochil-assets-v1";

const offlinePage = `<!doctype html><html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><meta name="color-scheme" content="dark light"><title>Offline · Mochil</title><style>
:root{color-scheme:dark light;--bg:#1f1b18;--text:#f4eee6;--muted:#b9aea2;--accent:#f09a72}
@media(prefers-color-scheme:light){:root{--bg:#f5efe6;--text:#241f1a;--muted:#6f655b;--accent:#c85d3a}}
body{margin:0;min-height:100vh;display:grid;place-items:center;background:var(--bg);color:var(--text);font:16px/1.5 system-ui,sans-serif;text-align:center;padding:24px}
h1{font:600 26px/1.2 Georgia,serif;margin:0 0 8px}p{margin:0 0 20px;color:var(--muted)}
button{min-height:44px;padding:0 22px;border:0;border-radius:10px;background:var(--accent);color:#1a1411;font:inherit;font-weight:650;cursor:pointer}
</style></head><body><main><h1>You're offline</h1><p>Mochil can't reach your server right now.</p><button onclick="location.reload()">Try again</button></main></body></html>`;

self.addEventListener("install", (event) => {
  // Fonts are small and needed by every screen, including the offline ones.
  event.waitUntil(caches.open(ASSETS).then((cache) => cache.addAll(["/assets/fonts/newsreader.woff2", "/assets/fonts/source-sans.woff2"])).catch(() => {}).then(() => self.skipWaiting()));
});

self.addEventListener("activate", (event) => {
  event.waitUntil((async () => {
    // Only Mochil' own caches are ever removed; another app on this origin keeps its own.
    for (const name of await caches.keys()) {
      if (name.startsWith("mochil-") && name !== SHELL && name !== ASSETS) await caches.delete(name);
    }
    await self.clients.claim();
  })());
});

// Built bundles carry a content hash in their name, so a cached copy is never stale. Fonts and icons keep their URL when
// their bytes change, so they are served from the cache and refreshed behind the scenes.
const hashed = (path) => path.startsWith("/assets/dist/");
const refreshed = (path) => path.startsWith("/assets/fonts/") || path.startsWith("/assets/icons/");
// Old builds pile up as bundles are replaced; the oldest entries go once the cache holds more than this.
const MAXIMUM_ASSETS = 300;

async function remember(cache, request, response) {
  try {
    await cache.put(request, response);
    const keys = await cache.keys();
    for (const key of keys.slice(0, Math.max(0, keys.length - MAXIMUM_ASSETS))) await cache.delete(key);
  } catch {
    // A full or blocked cache only means the next load goes to the network.
  }
}

self.addEventListener("fetch", (event) => {
  const { request } = event;
  if (request.method !== "GET") return;
  const url = new URL(request.url);
  if (url.origin !== self.location.origin) return;

  if (hashed(url.pathname) || refreshed(url.pathname)) {
    event.respondWith((async () => {
      const cache = await caches.open(ASSETS);
      const hit = await cache.match(request);
      const refresh = async () => {
        const response = await fetch(request);
        if (response.ok) await remember(cache, request, response.clone());
        return response;
      };
      if (hit && hashed(url.pathname)) return hit;
      if (hit) {
        event.waitUntil(refresh().catch(() => {}));
        return hit;
      }
      return refresh();
    })());
    return;
  }

  if (request.mode === "navigate") {
    event.respondWith((async () => {
      try {
        const response = await fetch(request);
        if (url.pathname === "/" && response.ok && response.type === "basic") {
          (await caches.open(SHELL)).put("/", response.clone());
        }
        return response;
      } catch {
        const shell = url.pathname === "/" ? await (await caches.open(SHELL)).match("/") : undefined;
        return shell || new Response(offlinePage, { status: 503, headers: { "Content-Type": "text/html; charset=utf-8" } });
      }
    })());
  }
});
