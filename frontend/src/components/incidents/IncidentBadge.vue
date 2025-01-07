<script setup lang="ts">
import { computed } from 'vue';
import type { Severity, Status } from '../../types/incident';

const props = defineProps<{
  type: 'severity' | 'status';
  value: Severity | Status;
}>();

const getColor = computed(() => {
  const colors = {
    severity: {
      low: 'bg-red_light text-white ring-red_light',
      medium: 'bg-red_mid text-white ring-red_mid',
      high: 'bg-red_dark text-white ring-red_dark',
    },
    status: {
      open: 'bg-blue-100 text-blue-800 ring-blue-600/20',
      assigned: 'bg-purple-100 text-purple-800 ring-purple-600/20',
      in_progress: 'bg-yellow-100 text-yellow-800 ring-yellow-600/20',
      resolved: 'bg-green-100 text-green-800 ring-green-600/20',
      closed: 'bg-gray-100 text-gray-800 ring-gray-600/20'
    }
  };
  return colors[props.type][props.value as keyof typeof colors[typeof props.type]] || '';
});
</script>

<template>
  <span :class="[
    'inline-flex items-center rounded-md px-2 py-1 text-xs font-medium ring-1 ring-inset',
    getColor
  ]">
    {{ value }}
  </span>
</template>