import api from '@/config/axios';
import type { 
  Incident, 
} from '@/types/incident';
import { IncidentChartData } from '@/types/chart-data';

export class IncidentService {
  private static instance: IncidentService;
  private readonly baseUrl = '/incidents';

  private constructor() {}

  public static get(): IncidentService {
    if (!IncidentService.instance) {
      IncidentService.instance = new IncidentService();
    }
    return IncidentService.instance;
  }

  async getAllIncidents(): Promise<Incident[]> {
    const response = await api.get<Incident[]>(this.baseUrl);
    return response.data;
  }

  async getIncidentById(id: string): Promise<Incident> {
    const response = await api.get<Incident>(`${this.baseUrl}/${id}`);
    return response.data;
  }

  async createIncident(incident: Omit<Incident, 'id' | 'comments' | 'created_at' | 'updated_at'>): Promise<Incident> {
    const response = await api.post<Incident>(this.baseUrl, { ...incident, comments: [] });
    return response.data;
  }

  async updateIncident(id: string, incident: Partial<Incident>): Promise<Incident> {
    const response = await api.put<Incident>(`${this.baseUrl}/${id}`, incident);
    return response.data;
  }

  async deleteIncident(id: string): Promise<void> {
    await api.delete(`${this.baseUrl}/${id}`);
  }

  async getChartData(startDate: Date, endDate: Date): Promise<IncidentChartData> {
    const response = await api.get<IncidentChartData>(`${this.baseUrl}/chart-data`, {
      params: {
        startDate: startDate.toISOString().split('T')[0],
        endDate: endDate.toISOString().split('T')[0]
      }
    });

    return response.data;
  }
}
