<script setup lang="ts">
import { Bar } from 'vue-chartjs';
import { chartDefaults } from '@/utils/chartConfig';
import { BarChartData } from './types';

const props = defineProps<{
  data: BarChartData;
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
    <Bar :data="props.data" :options="options" />
  </div>
</template>