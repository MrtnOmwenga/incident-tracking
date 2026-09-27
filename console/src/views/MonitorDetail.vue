<script setup lang="ts">
import { computed, ref } from 'vue';
import { api, type Check, type Mode, type Monitor } from '../api';
import { ago, when } from '../format';
import { usePoll } from '../poll';
import HealthDot from '../components/HealthDot.vue';
import ModeSwitch from '../components/ModeSwitch.vue';
import Sparkline from '../components/Sparkline.vue';

const props = defineProps<{ id: string }>();
const monitor = ref<Monitor | null>(null);
const checks = ref<Check[]>([]);
const error = ref('');

async function refresh() {
  try {
    [monitor.value, checks.value] = await Promise.all([api.monitor(props.id), api.checks(props.id)]);
    error.value = '';
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Could not load the monitor.';
  }
}
usePoll(refresh, 5000);

const passing = computed(() => checks.value.filter((c) => c.ok).length);

async function setMode(mode: Mode) {
  if (!monitor.value) return;
  monitor.value = await api.setMode(monitor.value.id, mode);
}
</script>

<template>
  <section class="section stack">
    <RouterLink class="more" to="/monitors">← All monitors</RouterLink>
    <p v-if="error" class="error" role="alert">{{ error }}</p>
    <template v-if="monitor">
      <div class="view-head">
        <div class="stack-sm">
          <span class="kicker">{{ monitor.kind === 'http' ? 'HTTP monitor' : 'Simulated site' }}</span>
          <h1 class="page-title">{{ monitor.name }}</h1>
          <p class="byline">
            <HealthDot :health="monitor.health" /> · checked {{ ago(monitor.lastCheckedAt) }} · every {{ monitor.intervalSeconds }} s ·
            down after {{ monitor.failureThreshold }} failures, up after {{ monitor.recoveryThreshold }} successes
          </p>
        </div>
        <ModeSwitch v-if="monitor.kind === 'simulated'" :mode="monitor.simulatedMode" :name="monitor.name" @change="setMode" />
      </div>

      <div class="panel ink chart-panel">
        <p class="label">Last {{ checks.length }} checks · {{ passing }} passed</p>
        <Sparkline v-if="checks.length" :checks="checks" />
        <p v-else class="caption">No checks yet. The first runs within {{ monitor.intervalSeconds }} seconds.</p>
      </div>

      <div class="table-wrap">
        <table class="data console-table">
          <thead><tr><th scope="col">When</th><th scope="col">Result</th><th scope="col" class="num">Status</th><th scope="col" class="num">Response</th><th scope="col">Failure</th></tr></thead>
          <tbody>
            <tr v-for="c in checks.slice(0, 30)" :key="c.id">
              <td>{{ when(c.at) }}</td>
              <td><span class="status" :class="c.ok ? 'up' : 'down'">{{ c.ok ? '● Passed' : '● Failed' }}</span></td>
              <td class="num">{{ c.statusCode ?? '–' }}</td>
              <td class="num">{{ c.latencyMs }} ms</td>
              <td>{{ c.failure ?? '' }}</td>
            </tr>
          </tbody>
        </table>
      </div>
    </template>
  </section>
</template>
