<script setup lang="ts">
defineProps<{
  title: string;
  value: number | string;
  trend?: number;
  icon?: string;
  color?: string;
}>();
</script>

<template>
  <div class="bg-white rounded-xl shadow-lg p-6 transform transition-all hover:scale-105">
    <div class="flex justify-between">
      <div>
        <h3 class="text-gray-500 text-sm font-medium mb-2">{{ title }}</h3>
        <p :class="`text-3xl font-bold ${color || 'text-blue-900'}`">{{ value }}</p>
      </div>
      <div v-if="trend !== undefined" class="flex items-start">
        <span :class="`${trend >= 0 ? 'text-green-500' : 'text-red-500'} text-sm font-medium`">
          {{ trend >= 0 ? '+' : '' }}{{ trend }}%
        </span>
      </div>
    </div>
  </div>
</template>