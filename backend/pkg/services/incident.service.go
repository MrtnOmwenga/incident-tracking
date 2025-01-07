package services

import (
	"time"

	"github.com/MrtnOmwenga/incident-tracking/backend/pkg/models"
	"github.com/MrtnOmwenga/incident-tracking/backend/pkg/repositories"
)

type IncidentService struct {
	Repo *repositories.IncidentRepository
}

func NewIncidentService(repo *repositories.IncidentRepository) *IncidentService {
	return &IncidentService{Repo: repo}
}

func (service *IncidentService) GetAllIncidents() ([]models.Incident, error) {
	return service.Repo.GetAll()
}

func (service *IncidentService) GetIncidentById(id string) (*models.Incident, error) {
	return service.Repo.GetById(id)
}

func (service *IncidentService) CreateIncident(incident *models.Incident) (*models.Incident, error) {
	return service.Repo.Create(incident)
}

func (service *IncidentService) UpdateIncident(incident *models.Incident) (*models.Incident, error) {
	return service.Repo.Update(incident)
}

func (service *IncidentService) DeleteIncident(id string) error {
	return service.Repo.Delete(id)
}

func (service *IncidentService) GetChartData(startDate, endDate time.Time) (*ChartResponse, error) {
	incidents, err := service.Repo.GetChartData(startDate, endDate)
	if err != nil {
		return nil, err
	}

	var labels []string
	currentDate := startDate
	for currentDate.Before(endDate) {
		labels = append(labels, currentDate.Format("Jan"))
		currentDate = currentDate.AddDate(0, 1, 0)
	}

	type monthlyCounts struct {
		high   int
		medium int
		low    int
	}
	monthlyData := make(map[string]*monthlyCounts)
	for _, label := range labels {
		monthlyData[label] = &monthlyCounts{}
	}

	for _, incident := range incidents {
		month := incident.CreatedAt.Format("Jan")
		if counts, exists := monthlyData[month]; exists {
			switch incident.Severity {
			case "high":
				counts.high++
			case "medium":
				counts.medium++
			case "low":
				counts.low++
			}
		}
	}

	highData := make([]int, len(labels))
	mediumData := make([]int, len(labels))
	lowData := make([]int, len(labels))
	totalData := make([]int, len(labels))

	for i, month := range labels {
		counts := monthlyData[month]
		highData[i] = counts.high
		mediumData[i] = counts.medium
		lowData[i] = counts.low
		totalData[i] = counts.high + counts.medium + counts.low
	}

	var pieData []int
	currentHigh := 0
	currentMedium := 0
	currentLow := 0
	for _, incident := range incidents {
		switch incident.Severity {
		case "high":
			currentHigh++
		case "medium":
			currentMedium++
		case "low":
			currentLow++
		}
	}
	pieData = []int{currentHigh, currentMedium, currentLow}

	return &ChartResponse{
		BarChartData: models.ChartData{
			Labels: labels,
			Datasets: []models.ChartDataset{
				{
					Label: "High",
					Data:  highData,
				},
				{
					Label: "Medium",
					Data:  mediumData,
				},
				{
					Label: "Low",
					Data:  lowData,
				},
			},
		},
		LineChartData: models.ChartData{
			Labels: labels,
			Datasets: []models.ChartDataset{
				{
					Label: "Incidents",
					Data:  totalData,
				},
			},
		},
		PieChartData: models.ChartData{
			Labels: []string{"High", "Medium", "Low"},
			Datasets: []models.ChartDataset{
				{
					Data: pieData,
				},
			},
		},
	}, nil
}
