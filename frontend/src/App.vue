<script setup lang="ts">

import { RouterView } from 'vue-router';
import Navbar from '@/components/layout/navbar.vue';

</script>

<template>
  <div class="min-h-screen bg-gradient-to-br from-gray-50 to-gray-100 font-mono">
    <!-- Navigation -->
    <Navbar />

    <!-- Main Content -->
    <main class="max-w-10xl mx-auto py-6 sm:px-6 lg:px-8">
      <RouterView />
    </main>
  </div>
</template>