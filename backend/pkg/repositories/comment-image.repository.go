package repositories

import (
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/MrtnOmwenga/incident-tracking/backend/pkg/models"
)

type CommentImageRepository struct {
	DB *gorm.DB
}

func NewCommentImageRepository(db *gorm.DB) *CommentImageRepository {
	return &CommentImageRepository{DB: db}
}

func (repo *CommentImageRepository) GetByCommentID(incidentID string) ([]models.CommentImage, error) {
	var commentImages []models.CommentImage
	if err := repo.DB.Where("incident_id = ?", incidentID).Find(&commentImages).Error; err != nil {
		return nil, err
	}
	return commentImages, nil
}

func (repo *CommentImageRepository) GetById(commentImageId string) (*models.CommentImage, error) {
	var commentImage models.CommentImage

	id, err := uuid.Parse(commentImageId)
	if err != nil {
		return nil, errors.New("invalid comment image ID format")
	}

	if err := repo.DB.First(&commentImage, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("comment image not found")
		}
		return nil, err
	}

	return &commentImage, nil
}

func (repo *CommentImageRepository) Create(commentImage *models.CommentImage) (*models.CommentImage, error) {
	if commentImage.ID == uuid.Nil {
		commentImage.ID = uuid.New()
	}

	if err := repo.DB.Create(commentImage).Error; err != nil {
		return nil, err
	}

	return commentImage, nil
}

func (repo *CommentImageRepository) Update(commentImage *models.CommentImage) (*models.CommentImage, error) {
	var existing models.CommentImage
	if err := repo.DB.First(&existing, "id = ?", commentImage.ID).Error; err != nil {
		return nil, errors.New("comment image not found")
	}

	if err := repo.DB.Save(commentImage).Error; err != nil {
		return nil, err
	}

	return commentImage, nil
}

func (repo *CommentImageRepository) Delete(commentImageId string) error {
	id, err := uuid.Parse(commentImageId)
	if err != nil {
		return errors.New("invalid comment image ID format")
	}

	result := repo.DB.Delete(&models.CommentImage{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return errors.New("comment image not found")
	}

	return nil
}
