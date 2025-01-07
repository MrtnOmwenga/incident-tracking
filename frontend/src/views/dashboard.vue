<script setup>

import Dashboard from '@/components/Dashboard.vue';
import IncidentList from '@/components/incidents/IncidentList.vue';

</script>

<template>
  <Dashboard />

  <div class="mx-6 my-2">
    <IncidentList :limit="4"/>
  </div>
</template>