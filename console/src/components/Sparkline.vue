<script setup lang="ts">
import { computed } from 'vue';
import type { Check } from '../api';

// Response times of the most recent checks, oldest on the left; failed checks are red ticks.
const props = defineProps<{ checks: Check[] }>();
const W = 600, H = 80, PAD = 4;

const shape = computed(() => {
  const list = [...props.checks].reverse();
  const n = list.length;
  const peak = Math.max(1, ...list.filter((c) => c.ok).map((c) => c.latencyMs));
  const x = (i: number) => (n <= 1 ? W / 2 : PAD + (i / (n - 1)) * (W - 2 * PAD));
  const y = (v: number) => H - PAD - (v / peak) * (H - 2 * PAD);
  const points = list.map((c, i) => (c.ok ? `${x(i).toFixed(1)},${y(c.latencyMs).toFixed(1)}` : null)).filter(Boolean).join(' ');
  const fails = list.map((c, i) => (c.ok ? null : x(i))).filter((v): v is number => v !== null);
  return { points, fails, peak };
});
</script>

<template>
  <svg class="spark big" :viewBox="`0 0 ${W} ${H}`" preserveAspectRatio="none" role="img"
       :aria-label="`Response times of the last ${checks.length} checks, up to ${shape.peak} ms`">
    <rect v-for="(fx, i) in shape.fails" :key="i" class="fail" :x="fx - 1.5" y="0" width="3" :height="H" />
    <polyline :points="shape.points" />
  </svg>
</template>
