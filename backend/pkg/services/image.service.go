package services

import (
	"github.com/MrtnOmwenga/incident-tracking/backend/pkg/models"
	"github.com/MrtnOmwenga/incident-tracking/backend/pkg/repositories"
)

type ImageService struct {
	Repo *repositories.ImageRepository
}

func NewImageService(repo *repositories.ImageRepository) *ImageService {
	return &ImageService{Repo: repo}
}

func (service *ImageService) GetImagesForIncident(incidentID string) ([]models.Image, error) {
	return service.Repo.GetByIncidentID(incidentID)
}

func (service *ImageService) GetImageById(id string) (*models.Image, error) {
	return service.Repo.GetById(id)
}

func (service *ImageService) CreateImage(image *models.Image) (*models.Image, error) {
	return service.Repo.Create(image)
}

func (service *ImageService) UpdateImage(image *models.Image) (*models.Image, error) {
	return service.Repo.Update(image)
}

func (service *ImageService) DeleteImage(id string) error {
	return service.Repo.Delete(id)
}
