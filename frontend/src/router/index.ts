import { createRouter, createWebHistory } from 'vue-router';
import Dashboard from '@/views/dashboard.vue';
import Incidents from '@/views/incidents.vue';
import NewIncident from '@/views/new-incident.vue';

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    {
      path: '/',
      name: 'dashboard',
      component: Dashboard,
    },
    {
      path: '/incidents',
      name: 'incidents',
      component: Incidents,
    },
    {
      path: '/new-incident',
      name: 'new-incident',
      component: NewIncident,
    }
  ]
});

export default router;