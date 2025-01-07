import { defineStore } from 'pinia';
import { ref, computed } from 'vue';
import type { Incident, Status, Severity } from '../types/incident';
import { IncidentService } from '@/services/incident.service';
import { IncidentChartData } from '@/types/chart-data';

export const useIncidentStore = defineStore('incidents', () => {
  const _incidents = ref<Incident[]>([]);
  const _chartData = ref<IncidentChartData>({} as IncidentChartData);
  const filterStatus = ref<Status | 'all'>('all');
  const filterSeverity = ref<Severity | 'all'>('all');
  const dateRange = ref<{ start: Date | null; end: Date | null }>({
    start: null,
    end: null
  });
  const isInitialized = ref(false); 

  async function init() {
    if (!isInitialized.value) {
      const today = new Date();
      const startDate = new Date(today.getTime() - 365 * 24 * 60 * 60 * 1000); // 30 days ago
      const endDate = new Date(today.getTime() + 24 * 60 * 60 * 1000); // Tomorrow

      _incidents.value = await IncidentService.get().getAllIncidents();
      _chartData.value = await IncidentService.get().getChartData(startDate, endDate);

      isInitialized.value = true;
    }
  }

  const incidents = computed(() => {
    return _incidents.value
      .filter((incident) => {
        const matchesStatus = filterStatus.value === 'all' || incident.status === filterStatus.value;
        const matchesSeverity = filterSeverity.value === 'all' || incident.severity === filterSeverity.value;
        const matchesDate = !dateRange.value.start || !dateRange.value.end || 
          ( new Date(incident.created_at) >= dateRange.value.start && new Date(incident.created_at) <= dateRange.value.end);
        
        return matchesStatus && matchesSeverity && matchesDate;
      })
      .sort((a, b) => (new Date(b.created_at)).getTime() - (new Date(a.created_at)).getTime());
  });

  const stats = computed(() => {
    const total = _incidents.value.length;
    const openIncidents = _incidents.value.filter(i => i.status === 'open').length;
    const resolvedToday = _incidents.value.filter(i => 
      i.status === 'resolved' && 
      new Date(i.updated_at).toDateString() === new Date().toDateString()
    ).length;
  
  
    return {
      total,
      openIncidents,
      resolvedToday,
    };
  });

  function addIncident(incident: Omit<Incident, 'id' | 'comments' | 'created_at' | 'updated_at'>) {
    const newIncident: Incident = {
      ...incident,
      id: crypto.randomUUID(),
      comments: [],
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString(),
    };
    _incidents.value.unshift(newIncident);
  }

  const chartData = computed(() => {
    return {
      barChartData: {
        labels: _chartData.value.barChartData.labels,
        datasets: _chartData.value.barChartData.datasets.map((dataset, index) => ({
          ...dataset,
          label: dataset.label as Severity,
          backgroundColor: [
            '#7f1d1d',  // high
            '#b91c1c',  // medium
            '#ef4444'   // low
          ][index],
          tension: 0.4
        }))
      },
      lineChartData: {
        labels: _chartData.value.lineChartData.labels,
        datasets: _chartData.value.lineChartData.datasets.map(dataset => ({
          ...dataset,
          borderColor: '#7f1d1d',
          backgroundColor: 'rgba(220, 38, 38, 0.1)',
          fill: true,
          tension: 0.4
        }))
      },
      pieChartData: {
        labels: _chartData.value.pieChartData.labels,
        datasets: _chartData.value.pieChartData.datasets.map(dataset => ({
          ...dataset,
          backgroundColor: [
            '#7f1d1d',  // high
            '#b91c1c',  // medium
            '#ef4444'   // low
          ]
        }))
      }
    }
  });

  console.log(_incidents);
  console.log(_chartData);
  
  return {
    incidents,
    stats,
    chartData,
    filterStatus,
    filterSeverity,
    dateRange,
    addIncident,
    init,
    isInitialized, 
  };
});