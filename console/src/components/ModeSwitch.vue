<script setup lang="ts">
import type { Mode } from '../api';

// A simulated site's behaviour, as a row of radio buttons: the sandbox's "break it" control.
const props = defineProps<{ mode: Mode; name: string; disabled?: boolean }>();
const emit = defineEmits<{ change: [Mode] }>();
const modes: { value: Mode; label: string }[] = [
  { value: 'up', label: 'Up' },
  { value: 'slow', label: 'Slow' },
  { value: 'flaky', label: 'Flaky' },
  { value: 'down', label: 'Down' },
];
</script>

<template>
  <fieldset class="mode-switch" :disabled="props.disabled">
    <legend class="sr-only">How {{ props.name }} behaves</legend>
    <label v-for="m in modes" :key="m.value" :class="['mode', m.value, { on: props.mode === m.value }]">
      <input type="radio" :name="`mode-${props.name}`" :value="m.value" :checked="props.mode === m.value" @change="emit('change', m.value)">
      {{ m.label }}
    </label>
  </fieldset>
</template>
