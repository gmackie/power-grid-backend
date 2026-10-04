// Service Worker for Power Grid Digital PWA
//
// Pages are network-first so a deploy reaches phones on the next load; only
// content-hashed bundles and art are served cache-first.

const CACHE_NAME = 'power-grid-v2';

self.addEventListener('install', () => {
  self.skipWaiting();
});

self.addEventListener('activate', (event) => {
  event.waitUntil(
    caches.keys()
      .then((names) => Promise.all(names.filter((n) => n !== CACHE_NAME).map((n) => caches.delete(n))))
      .then(() => self.clients.claim())
  );
});

const isImmutable = (url) => url.pathname.startsWith('/assets/') || url.pathname.startsWith('/art/');

self.addEventListener('fetch', (event) => {
  const { request } = event;
  if (request.method !== 'GET') return;
  const url = new URL(request.url);
  // Same-origin static files only; API, WebSocket upgrades and dev-server modules go straight to the network.
  if (url.origin !== self.location.origin) return;
  if (request.mode === 'navigate') {
    event.respondWith(networkFirst(request));
  } else if (isImmutable(url)) {
    event.respondWith(cacheFirst(request));
  }
});

async function cacheFirst(request) {
  const cached = await caches.match(request);
  if (cached) return cached;
  const response = await fetch(request);
  if (response.ok) {
    const cache = await caches.open(CACHE_NAME);
    cache.put(request, response.clone());
  }
  return response;
}

async function networkFirst(request) {
  try {
    const response = await fetch(request);
    if (response.ok) {
      const cache = await caches.open(CACHE_NAME);
      cache.put('/index.html', response.clone());
    }
    return response;
  } catch (error) {
    const cached = await caches.match('/index.html');
    if (cached) return cached;
    throw error;
  }
}
