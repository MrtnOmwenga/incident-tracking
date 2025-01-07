import { Severity } from '@/types/incident';

interface BarDataSet {
  label: Severity,
  data: number[],
  backgroundColor: string,
  tension: number,
}

interface TrendDataSet {
  label: string,
  data: number[],
  borderColor: string,
  backgroundColor: string,
  fill: boolean,
  tension: number
}

interface PieDataSet {
  backgroundColor:string[],
  data: number[]
}

export interface BarChartData {
  labels: string[],
  datasets: BarDataSet[],
}

export interface TrendChartData {
  labels: string[],
  datasets: TrendDataSet[],
}

export interface PieChartData {
  labels: string[],
  datasets: PieDataSet[],
}