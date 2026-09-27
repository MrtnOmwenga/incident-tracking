// Visit counting, without cookies or storage. See /privacy for what is recorded and why.
(() => {
  // Respect the browser's "don't track me" signals before doing anything at all.
  if (navigator.globalPrivacyControl || navigator.doNotTrack === '1') {
    window.lighthouse = { track() {} };
    return;
  }
  const id = crypto.randomUUID ? crypto.randomUUID()
    : ([1e7] + -1e3 + -4e3 + -8e3 + -1e11).replace(/[018]/g, (c) => (c ^ (crypto.getRandomValues(new Uint8Array(1))[0] & (15 >> (c / 4)))).toString(16));
  const send = (path, body) => fetch(path, {
    method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body),
    keepalive: true, credentials: 'same-origin',
  }).catch(() => {});

  // A ?ref= tag (one per job application, say) is read once, then removed from the address bar
  // so it isn't passed on if the link is shared.
  const params = new URLSearchParams(location.search);
  const ref = params.get('ref') || '';
  if (ref) {
    params.delete('ref');
    const q = params.toString();
    history.replaceState(history.state, '', location.pathname + (q ? `?${q}` : '') + location.hash);
  }
  send('/api/a/view', { id, path: location.pathname, ref, referrer: document.referrer });

  // Engaged time: a heartbeat every 15 s, only while the page is visible and someone has scrolled,
  // clicked or typed in the last minute. The server measures the gaps itself.
  let active = Date.now();
  for (const e of ['scroll', 'pointermove', 'keydown', 'touchstart', 'click']) {
    addEventListener(e, () => { active = Date.now(); }, { passive: true });
  }
  const engaged = () => document.visibilityState === 'visible' && Date.now() - active < 60_000;
  setInterval(() => { if (engaged()) send('/api/a/ping', { id }); }, 15_000);
  addEventListener('visibilitychange', () => {
    if (document.visibilityState === 'hidden' && Date.now() - active < 60_000) send('/api/a/ping', { id });
  });

  window.lighthouse = { track: (name) => send('/api/a/event', { id, name }) };
})();
