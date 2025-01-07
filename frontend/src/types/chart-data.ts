export interface ChartDataset {
  label: string;
  data: number[];
}

export interface ChartData {
  labels: string[];
  datasets: ChartDataset[];
}

export interface IncidentChartData {
  barChartData: ChartData;
  lineChartData: ChartData;
  pieChartData: ChartData;
}