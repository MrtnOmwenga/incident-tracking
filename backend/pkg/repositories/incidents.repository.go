package repositories

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/MrtnOmwenga/incident-tracking/backend/pkg/models"
)

// type definition
type IncidentRepository struct {
	DB *gorm.DB
}

// constructor
func NewIncidentRepository(db *gorm.DB) *IncidentRepository {
	return &IncidentRepository{DB: db}
}

func (repo *IncidentRepository) GetAll() ([]models.Incident, error) {
	var incidents []models.Incident
	if err := repo.DB.Preload("Images").Preload("Comments").Find(&incidents).Error; err != nil {
		return nil, err
	}

	return incidents, nil
}

func (repo *IncidentRepository) GetById(incidentId string) (*models.Incident, error) {
	var incident models.Incident

	id, err := uuid.Parse(incidentId)
	if err != nil {
		return nil, errors.New("invalid incident ID format")
	}

	if err := repo.DB.Preload("Images").Preload("Comments").First(&incident, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("incident not found")
		}
		return nil, err
	}

	return &incident, nil
}

func (repo *IncidentRepository) Create(incident *models.Incident) (*models.Incident, error) {
	if incident.ID == uuid.Nil {
		incident.ID = uuid.New()
	}

	if err := repo.DB.Create(incident).Error; err != nil {
		return nil, err
	}

	return incident, nil
}

func (repo *IncidentRepository) Update(incident *models.Incident) (*models.Incident, error) {
	var existing models.Incident
	if err := repo.DB.First(&existing, "id = ?", incident.ID).Error; err != nil {
		return nil, errors.New("incident not found")
	}

	if err := repo.DB.Save(incident).Error; err != nil {
		return nil, err
	}

	return incident, nil
}

func (repo *IncidentRepository) Delete(incidentId string) error {
	id, err := uuid.Parse(incidentId)
	if err != nil {
		return errors.New("invalid incident ID format")
	}

	result := repo.DB.Delete(&models.Incident{}, "id = ?", id)
	if result.Error != nil {
		return err
	}

	if result.RowsAffected == 0 {
		return errors.New("incident not found")
	}

	return nil
}

func (repo *IncidentRepository) GetChartData(startDate, endDate time.Time) ([]models.Incident, error) {
	var incidents []models.Incident
	if err := repo.DB.Where("created_at BETWEEN ? AND ?", startDate, endDate).Find(&incidents).Error; err != nil {
		return nil, err
	}
	return incidents, nil
}
