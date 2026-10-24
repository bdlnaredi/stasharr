/* Stasharr service worker.
 *
 * Makes Stasharr installable as an app and lets it open quickly. Only the
 * app's own build files are cached; library data, images and videos always
 * come from the server, so nothing goes stale.
 */
const CACHE = "stasharr-app-v1";
const SHELL = "stasharr-shell";

self.addEventListener("install", () => {
  self.skipWaiting();
});

self.addEventListener("activate", (event) => {
  event.waitUntil(
    caches
      .keys()
      .then((keys) =>
        Promise.all(keys.filter((k) => k !== CACHE).map((k) => caches.delete(k)))
      )
      .then(() => self.clients.claim())
  );
});

const offlinePage = `<!doctype html><html><head><meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Stasharr</title><style>body{background:#202b33;color:#f5f8fa;font-family:sans-serif;
display:flex;align-items:center;justify-content:center;height:100vh;margin:0;text-align:center}
button{margin-top:1rem;padding:.6rem 1.2rem;border:0;border-radius:4px;background:#137cbd;color:#fff;font-size:1rem}
</style></head><body><div><h2>Can't reach your server</h2>
<p>Check that you are on your home network or VPN.</p>
<button onclick="location.reload()">Try again</button></div></body></html>`;

self.addEventListener("fetch", (event) => {
  const req = event.request;
  if (req.method !== "GET") return;
  const url = new URL(req.url);
  if (url.origin !== self.location.origin) return;

  // built files have a hash in their name and never change
  if (url.pathname.includes("/assets/")) {
    event.respondWith(
      caches.open(CACHE).then((cache) =>
        cache.match(req).then(
          (hit) =>
            hit ||
            fetch(req).then((res) => {
              if (res.ok) cache.put(req, res.clone());
              return res;
            })
        )
      )
    );
    return;
  }

  // pages: always ask the server, fall back to the last copy when offline
  if (req.mode === "navigate") {
    event.respondWith(
      fetch(req)
        .then((res) => {
          if (res.ok && !res.redirected) {
            const copy = res.clone();
            caches.open(CACHE).then((cache) => cache.put(SHELL, copy));
          }
          return res;
        })
        .catch(() =>
          caches
            .open(CACHE)
            .then((cache) => cache.match(SHELL))
            .then(
              (hit) =>
                hit ||
                new Response(offlinePage, {
                  headers: { "Content-Type": "text/html; charset=utf-8" },
                })
            )
        )
    );
  }
});
