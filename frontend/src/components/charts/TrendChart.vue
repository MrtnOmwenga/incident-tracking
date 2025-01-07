<script setup lang="ts">
import { Line } from 'vue-chartjs';
import { chartDefaults } from '@/utils/chartConfig';
import { TrendChartData } from './types';

defineProps<{
  data: TrendChartData;
}>();

const options = {
  ...chartDefaults,
  aspectRatio: 2,
  scales: {
    y: {
      beginAtZero: true,
      ticks: {
        precision: 0  
      },
      grid: {
        color: 'rgba(0, 0, 0, 0.05)'
      }
    },
    x: {
      grid: {
        display: false
      }
    }
  }
};
</script>

<template>
  <div class="h-[300px]">
    <Line :data="data" :options="options" />
  </div>
</template>