// Mochil service worker: retires the cache the old shell left behind.
//
// This worker used to keep the legacy shell offline — index.html and the 245 modules behind it. That shell is gone: the
// app is served from / by the new client, which does not register a worker yet. An installed worker would otherwise
// keep answering navigations with a cached page pointing at files that no longer exist, so this version deletes its
// caches, unregisters itself and hands the network back. It intercepts nothing and caches nothing.
self.addEventListener("install", () => {
  self.skipWaiting();
});

self.addEventListener("activate", (event) => {
  event.waitUntil(
    (async () => {
      for (const name of await caches.keys()) {
        // Only Mochil's own caches: another app on this origin keeps its own.
        if (name.startsWith("mochil-")) await caches.delete(name);
      }
      await self.registration.unregister();
      await self.clients.claim();
    })(),
  );
});
