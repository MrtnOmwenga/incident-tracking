export type Severity = 'low' | 'medium' | 'high';
export type Status = 'open' | 'in_progress' | 'resolved' | 'closed';

export interface IncidentStats {
  total: number;
  openIncidents: number;
  resolvedToday: number;
  averageResolutionTime: number;
}

export interface Incident {
  id: string
  title: string
  description: string
  status: Status
  severity: Severity
  images: string[]
  comments: Comment[]
  created_at: string
  updated_at: string
}
