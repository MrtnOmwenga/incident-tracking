<script setup lang="ts">

import { onMounted } from 'vue';
import { storeToRefs } from 'pinia';
import { useIncidentStore } from '@/stores/incidents';
import { format, parseISO } from 'date-fns';
import IncidentBadge from './IncidentBadge.vue';
import { RouterLink } from 'vue-router';

const store = useIncidentStore();
const { incidents, isInitialized, filterStatus, filterSeverity } = storeToRefs(store);

onMounted(async () => {
  await store.init();
});

defineProps<{
  limit?: number;
}>();

</script>

<template>
  <div v-if="isInitialized" class="bg-white rounded-xl shadow-lg overflow-hidden">
    <!-- Filters -->
    <div class="p-4 border-b border-gray-200 bg-gray-50 flex justify-between items-center">
      <div class="flex flex-wrap gap-4">
        <select
          v-model="filterStatus"
          class="rounded-lg border-gray-300 shadow-sm focus:border-blue-900 focus:ring-blue-900 bg-white px-4 py-2"
        >
          <option value="all">All Status</option>
          <option value="open">Open</option>
          <option value="assigned">Assigned</option>
          <option value="in_progress">In Progress</option>
          <option value="resolved">Resolved</option>
          <option value="closed">Closed</option>
        </select>

        <select
          v-model="filterSeverity"
          class="rounded-lg border-gray-300 shadow-sm focus:border-blue-900 focus:ring-blue-900 bg-white px-4 py-2"
        >
          <option value="all">All Severity</option>
          <option value="low">Low</option>
          <option value="medium">Medium</option>
          <option value="high">High</option>
          <option value="critical">Critical</option>
        </select>
      </div>

      <RouterLink to="/incidents" v-if="limit" class="text-blue-500 text-sm mr-4">
        View All
      </RouterLink>
    </div>

    <!-- Incident List -->
    <div class="overflow-x-auto">
      <table class="min-w-full divide-y divide-gray-200">
        <thead class="bg-gray-50">
          <tr>
            <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Title</th>
            <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Severity</th>
            <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Status</th>
            <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Created</th>
          </tr>
        </thead>
        <tbody class="bg-white divide-y divide-gray-200">
          <tr v-for="incident in incidents.slice(0, limit || incidents.length)" :key="incident.id" class="hover:bg-gray-50">
            <td class="px-6 py-4">
              <div v-if="incident" class="text-sm font-medium text-gray-900">{{ incident.title }}</div>
              <div v-if="incident" class="text-sm text-gray-500 line-clamp-1">{{ incident.description }}</div>
            </td>
            <td class="px-6 py-4">
              <IncidentBadge
                v-if="incident && incident.severity" 
                type="severity" 
                :value="incident.severity"  
              />
            </td>
            <td class="px-6 py-4">
              <IncidentBadge 
                v-if="incident && incident.status" 
                type="status" 
                :value="incident.status" 
              />
            </td>
            <td class="px-6 py-4 text-sm text-gray-500">
              <div v-if="incident && incident.created_at" >
                {{ format(parseISO(incident.created_at), 'MMM d, yyyy') }}
              </div>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>

  <div v-else-if="!limit" class="bg-white rounded-xl shadow-lg p-4">
    <div class="flex items-center justify-center p-4">
      <div class="animate-spin rounded-full h-12 w-12 border-b-2 border-red_dark"></div>
      <span class="ml-3 text-red_mid">Loading...</span>
    </div>
  </div>
</template>