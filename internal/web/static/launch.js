// The launch page: introduces a demo while it wakes up, then opens it.
//
// Without JavaScript the page still works: every slide is visible and "Open" is a plain link.
(() => {
  const root = document.querySelector('[data-launch]');
  if (!root) return;
  document.documentElement.classList.add('js');

  const { slug, demo, name, tour } = root.dataset;
  const $ = (sel) => root.querySelector(sel);
  const slides = [...root.querySelectorAll('.slide')];
  const last = slides.length - 1;
  const state = $('[data-state]');
  const detail = $('[data-detail]');
  const skip = $('[data-skip]');
  const progress = $('[data-progress]');
  const dots = $('[data-dots]');
  const reducedMotion = matchMedia('(prefers-reduced-motion: reduce)').matches;
  const SLIDE_MS = 7000;

  let index = 0;
  let ready = false;
  let introDone = false;
  let autoplay = true;
  let timer;

  slides.forEach((_, i) => {
    const dot = document.createElement('button');
    dot.type = 'button';
    dot.className = 'dot';
    dot.setAttribute('aria-label', i === last ? 'Finish' : `Slide ${i + 1}`);
    dot.addEventListener('click', () => { takeControl(); show(i); });
    dots.append(dot);
  });

  function show(i) {
    index = Math.max(0, Math.min(i, last));
    slides.forEach((s, k) => {
      s.classList.toggle('active', k === index);
      s.setAttribute('aria-hidden', String(k !== index));
    });
    [...dots.children].forEach((d, k) => d.setAttribute('aria-current', String(k === index)));
    $('[data-prev]').disabled = index === 0;
    $('[data-next]').disabled = index === last;

    clearTimeout(timer);
    progress.style.transition = 'none';
    progress.style.width = '0';
    if (index === last) {
      introDone = true;
      maybeOpen();
      return;
    }
    if (autoplay) {
      if (!reducedMotion) {
        progress.getBoundingClientRect(); // restart the transition
        progress.style.transition = `width ${SLIDE_MS}ms linear`;
        progress.style.width = '100%';
      }
      timer = setTimeout(() => show(index + 1), SLIDE_MS);
    }
  }

  // Once someone navigates themselves, stop moving the slides under them.
  function takeControl() {
    autoplay = false;
    clearTimeout(timer);
    progress.style.transition = 'none';
    progress.style.width = '0';
  }

  function open(url) {
    state.textContent = `Opening ${name}…`;
    location.assign(url);
  }

  // Open automatically only when both the demo and the visitor are ready, and there is no choice
  // to make: a demo with a guided tour waits on the last slide for the visitor to pick.
  function maybeOpen() {
    if (ready && introDone && !tour) setTimeout(() => open(demo), 800);
  }

  function markReady() {
    ready = true;
    root.classList.add('ready');
    state.textContent = `${name} is ready.`;
    detail.textContent = introDone ? '' : 'Finish the introduction, or skip it.';
    skip.hidden = introDone;
    skip.textContent = tour ? 'Skip intro' : `Skip intro and open ${name}`;
    maybeOpen();
  }

  skip.addEventListener('click', () => {
    takeControl();
    if (ready && !tour) open(demo);
    else show(last);
  });
  $('[data-prev]').addEventListener('click', () => { takeControl(); show(index - 1); });
  $('[data-next]').addEventListener('click', () => { takeControl(); show(index + 1); });
  document.addEventListener('keydown', (e) => {
    if (e.key === 'ArrowRight') { takeControl(); show(index + 1); }
    if (e.key === 'ArrowLeft') { takeControl(); show(index - 1); }
  });

  // Poll until the demo answers. Quickly at first, then more patiently; after a few minutes say
  // so, and let the visitor try it anyway.
  const started = Date.now();
  let warned = false;
  async function poll() {
    try {
      const res = await fetch(`/api/projects/${encodeURIComponent(slug)}/ready`, { cache: 'no-store' });
      if (res.ok && (await res.json()).ready) return markReady();
    } catch { /* offline for a moment: keep trying */ }
    const waited = Date.now() - started;
    if (waited > 180_000 && !warned) {
      warned = true;
      root.classList.add('slow');
      state.textContent = `${name} is taking longer than usual.`;
      detail.textContent = 'You can keep waiting, or try opening it anyway.';
      skip.hidden = false;
      skip.textContent = 'Open anyway';
      skip.onclick = () => open(demo);
    }
    setTimeout(poll, waited < 60_000 ? 1500 : 5000);
  }

  skip.hidden = false;
  show(0);
  poll();
})();
