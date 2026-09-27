import { onMounted, onUnmounted } from 'vue';

// usePoll runs fn now and then every `ms` while the page is visible, and stops with the view.
export function usePoll(fn: () => Promise<unknown> | void, ms: number) {
  let timer: ReturnType<typeof setTimeout> | undefined;
  let stopped = false;
  const tick = async () => {
    if (stopped) return;
    if (document.visibilityState === 'visible') {
      try { await fn(); } catch { /* the view shows its own error state */ }
    }
    timer = setTimeout(tick, ms);
  };
  onMounted(tick);
  onUnmounted(() => { stopped = true; clearTimeout(timer); });
}
