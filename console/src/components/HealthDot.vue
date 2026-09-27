<script setup lang="ts">
import type { Health } from '../api';
defineProps<{ health: Health }>();
</script>

<template>
  <span class="status" :class="health">● {{ health === 'up' ? 'Up' : health === 'down' ? 'Down' : 'Checking' }}</span>
</template>
