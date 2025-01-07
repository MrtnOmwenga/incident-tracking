package services

import (
	"github.com/MrtnOmwenga/incident-tracking/backend/pkg/models"
)

type ChartResponse struct {
	BarChartData  models.ChartData `json:"barChartData"`
	LineChartData models.ChartData `json:"lineChartData"`
	PieChartData  models.ChartData `json:"pieChartData"`
}
