<script setup lang="ts">
import { ref } from 'vue';
import { api, type IncidentDetail, type Severity, type Status } from '../api';
import { when, words } from '../format';
import { usePoll } from '../poll';
import { session } from '../session';

const props = defineProps<{ id: string }>();
const owner = session.me?.role === 'owner';
const incident = ref<IncidentDetail | null>(null);
const error = ref('');

async function refresh() {
  try {
    incident.value = await api.incident(props.id);
    error.value = '';
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Could not load the incident.';
  }
}
usePoll(refresh, 5000);

async function change(update: { status?: Status; severity?: Severity }) {
  try {
    await api.updateIncident(props.id, update);
    await refresh();
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Could not update.';
  }
}

const message = ref('');
const isPublic = ref(true);
const posting = ref(false);
async function post() {
  posting.value = true;
  try {
    await api.comment(props.id, message.value, isPublic.value);
    message.value = '';
    await refresh();
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Could not post the update.';
  } finally {
    posting.value = false;
  }
}
</script>

<template>
  <section class="section grid">
    <div class="span-8 stack">
      <RouterLink class="more" to="/incidents">← All incidents</RouterLink>
      <p v-if="error" class="error" role="alert">{{ error }}</p>
      <template v-if="incident">
        <span class="kicker">{{ incident.automatic ? 'Opened automatically' : 'Opened by hand' }}{{ incident.public ? ' · public' : ' · private' }}</span>
        <h1 class="page-title">{{ incident.title }}</h1>
        <p class="byline"><span class="pill" :class="incident.status">{{ words(incident.status) }}</span> · {{ incident.severity }} severity · started {{ when(incident.startedAt) }}</p>

        <ol class="timeline">
          <li v-for="e in incident.events" :key="e.id" :class="e.kind">
            <time :datetime="e.at">{{ when(e.at) }}</time>
            <p><b class="sans">{{ words(e.kind) }}.</b> {{ e.message }}</p>
            <p class="caption">{{ e.author }}{{ e.public ? '' : ' · internal note, not on the status page' }}</p>
          </li>
        </ol>

        <form class="panel form-grid" @submit.prevent="post">
          <p class="label span-all">Post an update</p>
          <label class="span-all"><span class="sr-only">Update</span><textarea v-model="message" required maxlength="2000" rows="3" placeholder="What changed?"></textarea></label>
          <label class="check"><input v-model="isPublic" type="checkbox" :disabled="!incident.public"> Public update</label>
          <div class="actions"><button type="submit" class="button primary" :disabled="posting">Post</button></div>
          <p v-if="!incident.public" class="caption span-all">This incident is private, so its updates are internal notes.</p>
        </form>
      </template>
    </div>
    <aside v-if="incident" class="span-4 panel align-start stack">
      <p class="label">Status</p>
      <div class="actions">
        <button v-for="s in (['open', 'in_progress', 'resolved'] as Status[])" :key="s" type="button" class="button"
                :class="{ primary: incident.status === s }" :aria-pressed="incident.status === s" @click="change({ status: s })">{{ words(s) }}</button>
      </div>
      <p class="label">Severity</p>
      <div class="actions">
        <button v-for="s in (['low', 'medium', 'high'] as Severity[])" :key="s" type="button" class="button"
                :class="{ primary: incident.severity === s }" :aria-pressed="incident.severity === s" @click="change({ severity: s })">{{ s }}</button>
      </div>
      <p v-if="incident.public && owner" class="caption"><a :href="`/status/incidents/${incident.id}`">See it on the public status page</a></p>
      <p v-else-if="incident.public" class="caption">Public updates appear on <RouterLink to="/status">your status page</RouterLink>.</p>
    </aside>
  </section>
</template>
