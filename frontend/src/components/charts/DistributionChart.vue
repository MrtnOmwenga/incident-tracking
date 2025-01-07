<script setup lang="ts">
import { Doughnut } from 'vue-chartjs';
import { chartDefaults } from '@/utils/chartConfig';
import { PieChartData } from './types';

defineProps<{
  data: PieChartData;
}>();

const options = {
  ...chartDefaults,
  aspectRatio: 1.5
};
</script>

<template>
  <div class="h-[300px]">
    <Doughnut :data="data" :options="options" />
  </div>
</template>