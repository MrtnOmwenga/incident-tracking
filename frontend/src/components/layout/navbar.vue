<script setup lang='ts'>

import { RouterLink, useRoute } from 'vue-router';

const isActiveLink = (routePath: string) => {
  const route = useRoute();
  return route.path === routePath;
}

</script>

<template>
  <nav class="bg-red_dark to-blue-700 text-white shadow-lg">
      <div class="max-w-8xl mx-auto px-4 sm:px-6 lg:px-8">
        <div class="flex items-center justify-between h-16">
          <div class="flex items-center">
            <i class="pi pi-code text-white text-3xl mr-4"></i>
            <span class="text-xl font-bold">Incident Tracker</span>
          </div>
          <div class="flex space-x-4">
            <RouterLink 
              to="/"
              :class="['px-3 py-2 rounded-md text-sm font-medium', 
                isActiveLink('/') ? 'bg-white text-black' : 'text-white hover:bg-red_mid']"
            >
              Dashboard
            </RouterLink>
            <RouterLink 
              to="/new-incident"
              :class="['px-3 py-2 rounded-md text-sm font-medium',
                isActiveLink('/new-incident') ? 'bg-white text-black' : 'text-white hover:bg-red_mid']"
            >
              New Incident
            </RouterLink>
            <RouterLink 
              to="/incidents"
              :class="['px-3 py-2 rounded-md text-sm font-medium',
                isActiveLink('/incidents') ? 'bg-white text-black' : 'text-white hover:bg-red_mid']"
            >
              View All
            </RouterLink>
          </div>
        </div>
      </div>
    </nav>
</template>