<script setup lang="ts">
import { computed, ref } from 'vue';
import { api, type StatusPage } from '../api';
import { pct, ms, when } from '../format';
import { usePoll } from '../poll';
import { session } from '../session';

// The tenant's status page as a visitor would see it: public monitors, public incidents and
// public updates only.
const page = ref<StatusPage | null>(null);
const error = ref('');
async function refresh() {
  try {
    page.value = await api.myStatus();
    error.value = '';
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Could not load the status page.';
  }
}
usePoll(refresh, 10000);

const label = computed(() => ({
  operational: 'All systems operational',
  degraded: 'Some systems are having problems',
  major_outage: 'Major outage',
  no_monitors: 'Nothing monitored yet',
}[page.value?.overall ?? 'no_monitors']));

const bar = (u: number | null) => (u === null ? 'none' : u >= 99.5 ? 'good' : u >= 95 ? 'fair' : 'bad');
</script>

<template>
  <section class="section stack">
    <span class="kicker">Status page preview</span>
    <p class="caption">
      What visitors see: public monitors, public incidents and public updates only.
      <template v-if="session.me?.role === 'owner'">The live version is at <a href="/status">/status</a>.</template>
      <template v-else>A sandbox's status page stays inside the console.</template>
    </p>
    <p v-if="error" class="error" role="alert">{{ error }}</p>
    <template v-if="page">
      <h1 class="page-title banner-status"><span class="dot" :class="page.overall" aria-hidden="true"></span>{{ label }}</h1>
      <article v-for="i in page.activeIncidents" :key="i.id" class="report">
        <span class="pill" :class="i.status">{{ i.status.replace('_', ' ') }}</span>
        <h3>{{ i.title }}</h3>
        <p v-if="i.events.length">{{ i.events[i.events.length - 1]?.message }}</p>
      </article>
      <div v-for="m in page.monitors" :key="m.slug" class="history-row">
        <div class="top"><span class="name">{{ m.name }}</span><span class="muted">{{ pct(m.uptime24h) }} today · median {{ ms(m.p50Ms) }}</span></div>
        <div class="bars" role="img" :aria-label="`${m.name}: daily uptime, last 90 days`">
          <span v-for="d in m.days" :key="d.date" :class="bar(d.uptime)" :title="`${d.date}: ${d.uptime === null ? 'no data' : pct(d.uptime)}`"></span>
        </div>
      </div>
      <p class="caption">Updated {{ when(page.updatedAt) }}</p>
    </template>
  </section>
</template>
