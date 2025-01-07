<script setup lang="ts">

import { onMounted } from 'vue';
import { storeToRefs } from 'pinia';
import { useIncidentStore } from '../stores/incidents';
import StatCard from './stats/StatCard.vue';
import TrendChart from './charts/TrendChart.vue';
import DistributionChart from './charts/DistributionChart.vue';
import BarChart from './charts/BarChart.vue';

const store = useIncidentStore();
const { chartData, isInitialized } = storeToRefs(store);

onMounted(async () => {
  await store.init();
});

</script>

<template>
  <div v-if="isInitialized" class="p-6 space-y-6">
    <!-- Stats Grid -->
    <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
      <StatCard 
        title="Project"
        value="Docker demo"
        color="text-black"
      />
      <StatCard 
        title="Total Incidents"
        :value="store.stats.total"
        color="text-red_dark"
      />
      <StatCard 
        title="Open Incidents"
        :value="store.stats.openIncidents"
        color="text-[#ea580c]"
      />
      <StatCard 
        title="Resolved Today"
        :value="store.stats.resolvedToday"
        color="text-green-600"
      />
    </div>

    <!-- Charts Grid -->
    <div class="grid grid-cols-1 lg:grid-cols-3 gap-6">
      <!-- Bar Chart -->
      <div class="bg-white rounded-xl shadow-lg p-6">
        <h3 class="text-lg font-semibold text-gray-900 mb-4">Incident Severity Comparison</h3>
        <BarChart :data="chartData.barChartData" />
      </div>

      <!-- Trend Chart -->
      <div class="bg-white rounded-xl shadow-lg p-6">
        <h3 class="text-lg font-semibold text-gray-900 mb-4">Incident Trends</h3>
        <TrendChart :data="chartData.lineChartData" />
      </div>

      <!-- Distribution Chart -->
      <div class="bg-white rounded-xl shadow-lg p-6">
        <h3 class="text-lg font-semibold text-gray-900 mb-4">Severity Distribution</h3>
        <DistributionChart :data="chartData.pieChartData" />
      </div>
    </div>
  </div>

  <div v-else class="bg-white rounded-xl shadow-lg p-4">
    <div class="flex items-center justify-center p-4">
      <div class="animate-spin rounded-full h-12 w-12 border-b-2 border-red_dark"></div>
      <span class="ml-3 text-red_mid">Loading...</span>
    </div>
  </div>
</template>