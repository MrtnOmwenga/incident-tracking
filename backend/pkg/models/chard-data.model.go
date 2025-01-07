package models

type ChartDataset struct {
	Label string `json:"label"`
	Data  []int  `json:"data"`
}

type ChartData struct {
	Labels   []string       `json:"labels"`
	Datasets []ChartDataset `json:"datasets"`
}

type IncidentChartData struct {
	BarChartData  ChartData `json:"barChartData"`
	LineChartData ChartData `json:"lineChartData"`
	PieChartData  ChartData `json:"pieChartData"`
}
