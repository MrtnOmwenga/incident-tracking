<script setup lang="ts">
import { computed, ref } from 'vue';
import { api, type Mode, type Monitor, type MonitorInput, type StatusMonitor } from '../api';
import { ago, pct, ms } from '../format';
import { usePoll } from '../poll';
import { session } from '../session';
import HealthDot from '../components/HealthDot.vue';
import ModeSwitch from '../components/ModeSwitch.vue';

const monitors = ref<Monitor[]>([]);
const stats = ref<Record<string, StatusMonitor>>({});
const error = ref('');
const loaded = ref(false);
const owner = computed(() => session.me?.role === 'owner');

async function refresh() {
  try {
    const [list, status] = await Promise.all([api.monitors(), api.myStatus()]);
    monitors.value = list;
    stats.value = Object.fromEntries(status.monitors.map((m) => [m.slug, m]));
    error.value = '';
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Could not load monitors.';
  } finally {
    loaded.value = true;
  }
}
usePoll(refresh, 5000);

async function setMode(m: Monitor, mode: Mode) {
  const before = m.simulatedMode;
  m.simulatedMode = mode; // show the change at once; the next check runs straight away
  try {
    Object.assign(m, await api.setMode(m.id, mode));
  } catch (e) {
    m.simulatedMode = before;
    error.value = e instanceof Error ? e.message : 'Could not change the mode.';
  }
}

async function remove(m: Monitor) {
  if (!confirm(`Delete ${m.name}? Its checks go with it; its incidents are kept.`)) return;
  try {
    await api.deleteMonitor(m.id);
    await refresh();
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Could not delete.';
  }
}

// Adding a monitor
const adding = ref(false);
const form = ref<MonitorInput & { name: string }>({ name: '', kind: 'simulated', url: '', intervalSeconds: 30, public: true });
const formError = ref('');
async function add() {
  formError.value = '';
  const body: MonitorInput = { ...form.value };
  if (body.kind !== 'http') delete body.url;
  try {
    await api.createMonitor(body);
    adding.value = false;
    form.value = { name: '', kind: 'simulated', url: '', intervalSeconds: 30, public: true };
    await refresh();
  } catch (e) {
    formError.value = e instanceof Error ? e.message : 'Could not add the monitor.';
  }
}
</script>

<template>
  <section class="section stack">
    <div class="view-head">
      <div class="stack-sm">
        <span class="kicker">Monitors</span>
        <h1 class="page-title">{{ monitors.length }} watched, {{ monitors.filter((m) => m.health === 'down').length }} down</h1>
      </div>
      <button type="button" class="button primary" @click="adding = !adding">{{ adding ? 'Cancel' : 'Add a monitor' }}</button>
    </div>
    <p v-if="error" class="error" role="alert">{{ error }}</p>

    <form v-if="adding" class="panel form-grid" @submit.prevent="add">
      <p class="label span-all">New monitor</p>
      <label>Name <input v-model="form.name" required maxlength="100" placeholder="Payments API"></label>
      <label v-if="owner">Kind
        <select v-model="form.kind"><option value="simulated">Simulated</option><option value="http">HTTP</option></select>
      </label>
      <label v-if="form.kind === 'http'" class="span-all">URL <input v-model="form.url" type="url" required placeholder="https://example.com/health"></label>
      <label>Check every (seconds) <input v-model.number="form.intervalSeconds" type="number" :min="owner ? 5 : 10" max="3600"></label>
      <label class="check"><input v-model="form.public" type="checkbox"> Show on the status page</label>
      <p v-if="!owner" class="caption span-all">Sandbox monitors are simulated: the sandbox never sends traffic to real sites.</p>
      <p v-if="formError" class="error span-all" role="alert">{{ formError }}</p>
      <div class="actions span-all"><button type="submit" class="button primary">Add</button></div>
    </form>

    <p v-if="loaded && !monitors.length" class="standfirst">Nothing is monitored yet. Add a monitor to start.</p>

    <div class="table-wrap" v-if="monitors.length">
      <table class="data console-table">
        <thead>
          <tr><th scope="col">Monitor</th><th scope="col">Health</th><th scope="col" class="num">Uptime, 24 h</th><th scope="col" class="num">Median</th><th scope="col">Last check</th><th scope="col">Behaviour</th><th scope="col"><span class="sr-only">Actions</span></th></tr>
        </thead>
        <tbody>
          <tr v-for="m in monitors" :key="m.id">
            <td class="name"><RouterLink :to="`/monitors/${m.id}`">{{ m.name }}</RouterLink>
              <span class="caption block">{{ m.kind === 'http' ? m.url : 'simulated' }} · every {{ m.intervalSeconds }} s</span></td>
            <td><HealthDot :health="m.health" /><RouterLink v-if="m.openIncidentId" class="caption block" :to="`/incidents/${m.openIncidentId}`">open incident</RouterLink></td>
            <td class="num">{{ pct(stats[m.slug]?.uptime24h) }}</td>
            <td class="num">{{ ms(stats[m.slug]?.p50Ms) }}</td>
            <td>{{ ago(m.lastCheckedAt) }}</td>
            <td><ModeSwitch v-if="m.kind === 'simulated'" :mode="m.simulatedMode" :name="m.name" @change="(mode) => setMode(m, mode)" /><span v-else class="caption">live HTTP</span></td>
            <td><button type="button" class="link-button" @click="remove(m)">Delete<span class="sr-only"> {{ m.name }}</span></button></td>
          </tr>
        </tbody>
      </table>
    </div>
  </section>
</template>
