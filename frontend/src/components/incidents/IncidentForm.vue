<script setup lang="ts">

import { onMounted, ref } from 'vue';
import { storeToRefs } from 'pinia';
import { useIncidentStore } from '@/stores/incidents';
import type { Severity, Status } from '@/types/incident';
import { IncidentService } from '@/services/incident.service';

const store = useIncidentStore();
const { isInitialized } = storeToRefs(store);

onMounted(async () => {
  await store.init();
});

const form = ref<{
  title: string;
  description: string;
  severity: Severity;
  images: File[];
}>({
  title: '',
  description: '',
  severity: 'low',
  images: [],
});

const submitIncident = async () => {
  const body = {
    title: form.value.title,
    description: form.value.description,
    severity: form.value.severity,
    status: 'open' as Status,
    images: form.value.images.map(file => URL.createObjectURL(file)),
  }
  await IncidentService.get().createIncident(body);
  store.addIncident(body);

  // Reset the form
  form.value = {
    title: '',
    description: '',
    severity: 'medium',
    images: [],
  };
};
</script>

<template>
  <div v-if="isInitialized" class="max-w-2xl mx-auto mt-12">
    <div class="bg-white rounded-xl shadow-lg p-8">
      <h2 class="text-2xl font-bold text-red_dark mb-8">Report New Incident</h2>
      <form @submit.prevent="submitIncident" class="space-y-6">
        <div>
          <label class="block text-sm font-medium text-gray-700 mb-1">Title</label>
          <input
            v-model="form.title"
            type="text"
            required
            class="block w-full rounded border-gray-900 shadow-sm focus:outline-none transition-colors rounded p-2"
            placeholder="Brief description of the incident"
          />
        </div>

        <div>
          <label class="block text-sm font-medium text-gray-700 mb-2">Severity</label>
          <select
            v-model="form.severity"
            class="block w-full rounded-lg border-gray-900 shadow-sm focus:outline-none transition-colors rounded p-2"
          >
            <option value="low">Low - Minor impact</option>
            <option value="medium">Medium - Moderate impact</option>
            <option value="high">High - Significant impact</option>
            <option value="critical">Critical - Severe impact</option>
          </select>
        </div>

        <div>
          <label class="block text-sm font-medium text-gray-700 mb-1">Description</label>
          <textarea
            v-model="form.description"
            rows="4"
            required
            class="block w-full rounded-lg border-gray-900 shadow-sm focus:outline-none transition-colors rounded p-2"
            placeholder="Detailed description of the incident..."
          ></textarea>
        </div>

        <div>
          <label class="block text-sm font-medium text-gray-700 mb-1">Supporting Images</label>
          <div class="mt-1 flex justify-center px-6 pt-5 pb-6 border-2 border-gray-300 border-dashed rounded-lg hover:border-red_dark transition-colors">
            <div class="space-y-1 text-center">
              <svg class="mx-auto h-12 w-12 text-gray-400" stroke="currentColor" fill="none" viewBox="0 0 48 48" aria-hidden="true">
                <path d="M28 8H12a4 4 0 00-4 4v20m32-12v8m0 0v8a4 4 0 01-4 4H12a4 4 0 01-4-4v-4m32-4l-3.172-3.172a4 4 0 00-5.656 0L28 28M8 32l9.172-9.172a4 4 0 015.656 0L28 28m0 0l4 4m4-24h8m-4-4v8m-12 4h.02" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" />
              </svg>
              <div class="flex text-sm text-gray-600">
                <label class="relative cursor-pointer rounded-md font-medium text-red_dark hover:text-red_mid">
                  <span>Upload files</span>
                  <input
                    type="file"
                    multiple
                    accept="image/*"
                    class="sr-only"
                    @change="(e) => form.images = Array.from((e.target as HTMLInputElement).files || [])"
                  />
                </label>
                <p class="pl-1">or drag and drop</p>
              </div>
              <p class="text-xs text-gray-500">PNG, JPG, GIF up to 10MB</p>
            </div>
          </div>
        </div>

        <button
          type="submit"
          class="w-full bg-red_dark text-white rounded-lg py-3 px-4 hover:bg-red_mid focus:outline-none focus:ring-0 focus:ring-offset-2 transition-colors"
        >
          Submit Incident Report
        </button>
      </form>
    </div>
  </div>

  <div v-else class="bg-white rounded-xl shadow-lg p-4">
    <div class="flex items-center justify-center p-4">
      <div class="animate-spin rounded-full h-12 w-12 border-b-2 border-red_dark"></div>
      <span class="ml-3 text-red_mid">Loading...</span>
    </div>
  </div>
</template>