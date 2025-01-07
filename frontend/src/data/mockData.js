import { subDays } from 'date-fns';
// import type { Incident } from '../types/incident';

const descriptions = [
  'Network connectivity issues affecting multiple users',
  'Database performance degradation detected',
  'Security alert: Multiple failed login attempts',
  'Application server unresponsive',
  'API rate limit exceeded',
  'Memory leak detected in production server',
  'SSL certificate expiration warning',
  'Backup process failed',
  'High CPU usage on main server',
  'Storage capacity reaching critical levels'
];

const generateIncident = (index) => {
  const severities = ['low', 'medium', 'high'];
  const statuses = 
    ['open', 'in_progress', 'resolved', 'closed'];
  
  return {
    id: crypto.randomUUID(),
    title: `Incident #${index + 1}: ${descriptions[index % descriptions.length]}`,
    description: descriptions[index % descriptions.length],
    severity: severities[Math.floor(Math.random() * severities.length)],
    status: statuses[Math.floor(Math.random() * statuses.length)],
    createdAt: subDays(new Date(), Math.floor(Math.random() * 30)).toISOString(),
    updatedAt: new Date().toISOString(),
    images: [],
    comments: [],
  };
};

export const generateMockIncidents = (count = 20) => {
  return Array.from({ length: count }, (_, i) => generateIncident(i));
};