package repositories

import (
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/MrtnOmwenga/incident-tracking/backend/pkg/models"
)

type ImageRepository struct {
	DB *gorm.DB
}

func NewImageRepository(db *gorm.DB) *ImageRepository {
	return &ImageRepository{DB: db}
}

func (repo *ImageRepository) GetByIncidentID(incidentID string) ([]models.Image, error) {
	var images []models.Image
	if err := repo.DB.Where("incident_id = ?", incidentID).Find(&images).Error; err != nil {
		return nil, err
	}
	return images, nil
}

func (repo *ImageRepository) GetById(imageId string) (*models.Image, error) {
	var image models.Image

	id, err := uuid.Parse(imageId)
	if err != nil {
		return nil, errors.New("invalid image ID format")
	}

	if err := repo.DB.First(&image, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("image not found")
		}
		return nil, err
	}

	return &image, nil
}

func (repo *ImageRepository) Create(image *models.Image) (*models.Image, error) {
	if image.ID == uuid.Nil {
		image.ID = uuid.New()
	}

	if err := repo.DB.Create(image).Error; err != nil {
		return nil, err
	}

	return image, nil
}

func (repo *ImageRepository) Update(image *models.Image) (*models.Image, error) {
	var existing models.Image
	if err := repo.DB.First(&existing, "id = ?", image.ID).Error; err != nil {
		return nil, errors.New("image not found")
	}

	if err := repo.DB.Save(image).Error; err != nil {
		return nil, err
	}

	return image, nil
}

func (repo *ImageRepository) Delete(imageId string) error {
	id, err := uuid.Parse(imageId)
	if err != nil {
		return errors.New("invalid image ID format")
	}

	result := repo.DB.Delete(&models.Image{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return errors.New("image not found")
	}

	return nil
}
