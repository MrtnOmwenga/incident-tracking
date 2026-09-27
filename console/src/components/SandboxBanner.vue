<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue';

const props = defineProps<{ expiresAt: string }>();
const now = ref(Date.now());
let timer: ReturnType<typeof setInterval> | undefined;
onMounted(() => { timer = setInterval(() => { now.value = Date.now(); }, 30_000); });
onUnmounted(() => clearInterval(timer));

const left = computed(() => {
  const m = Math.max(0, Math.round((new Date(props.expiresAt).getTime() - now.value) / 60_000));
  return m >= 60 ? `${Math.floor(m / 60)} h ${m % 60} min` : `${m} min`;
});
</script>

<template>
  <div class="sandbox-band" role="note">
    <div class="wrap">
      <span class="tag">SANDBOX</span>
      <span>Everything here is simulated and yours alone. It disappears in {{ left }}.</span>
    </div>
  </div>
</template>
