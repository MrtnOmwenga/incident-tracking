// The edge in front of Cloud Run. Each hostname maps to a service's *.run.app origin (ORIGINS);
// the request is forwarded as is (WebSocket upgrades included), plus:
//   X-Client-IP      the visitor's address, which Cloudflare knows and the origin otherwise wouldn't
//   X-Forwarded-Host the hostname the visitor used
//   X-Edge-Secret    proof the request came through here (Lighthouse refuses requests without it)
// Headers a visitor sends with those names are replaced, never passed through.

export default {
  async fetch(request, env) {
    const url = new URL(request.url);
    const origins = JSON.parse(env.ORIGINS);
    const origin = origins[url.hostname];
    if (!origin) return new Response("Not found", { status: 404 });

    const target = new URL(url.pathname + url.search, origin);
    const headers = new Headers(request.headers);
    headers.set("X-Client-IP", request.headers.get("CF-Connecting-IP") ?? "");
    headers.set("X-Forwarded-Host", url.hostname);
    headers.set("X-Forwarded-Proto", "https");
    headers.set("X-Edge-Secret", env.EDGE_SECRET);

    return fetch(target, {
      method: request.method,
      headers,
      body: request.body,
      redirect: "manual", // redirects go back to the browser, which follows them via the edge
    });
  },
};
