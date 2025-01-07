import { generateMockIncidents } from './src/data/mockData.js'; 
import axios from 'axios';

const api = axios.create({
  baseURL: 'http://backend:8080/api/v1',
  headers: {
    'Content-Type': 'application/json',
  },
});

async function createIncident(incident) {
  const response = await api.post('/incidents', { ...incident, comments: [] });
  return response.data;
}

async function saveMockIncidents(count) {
  const incidents = generateMockIncidents(count);
  
  for (const incident of incidents) {
    try {
      const newIncident = await createIncident({
        title: incident.title,
        description: incident.description,
        severity: incident.severity,
        status: incident.status,
        images: incident.images,  
        createdAt: new Date(),
        updatedAt: new Date(),
      });
      
      console.log(`Incident ${newIncident.title} created successfully.`);
    } catch (error) {
      console.error(`Failed to create incident: ${incident.title}`, error);
    }
  }
}

// Example usage: Save 50 mock incidents
saveMockIncidents(50);
