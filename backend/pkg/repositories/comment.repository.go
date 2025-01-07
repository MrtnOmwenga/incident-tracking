package repositories

import (
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/MrtnOmwenga/incident-tracking/backend/pkg/models"
)

// type definition
type CommentRepository struct {
	DB *gorm.DB
}

// constructor
func NewCommentRepository(db *gorm.DB) *CommentRepository {
	return &CommentRepository{DB: db}
}

func (repo *CommentRepository) GetAll() ([]models.Comment, error) {
	var comments []models.Comment
	if err := repo.DB.Preload("Images").Find(&comments).Error; err != nil {
		return nil, err
	}

	return comments, nil
}

func (repo *CommentRepository) GetById(commentId string) (*models.Comment, error) {
	var comment models.Comment

	id, err := uuid.Parse(commentId)
	if err != nil {
		return nil, errors.New("invalid Comment ID format")
	}

	if err := repo.DB.Preload("Images").First(&comment, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("comment not found")
		}
		return nil, err
	}

	return &comment, nil
}

func (repo *CommentRepository) GetByIncidentID(incidentID string) ([]models.Comment, error) {
	var comments []models.Comment
	if err := repo.DB.Preload("Images").Where("incident_id = ?", incidentID).Find(&comments).Error; err != nil {
		return nil, err
	}
	return comments, nil
}

func (repo *CommentRepository) Create(comment *models.Comment) (*models.Comment, error) {
	if comment.ID == uuid.Nil {
		comment.ID = uuid.New()
	}

	if err := repo.DB.Create(comment).Error; err != nil {
		return nil, err
	}

	return comment, nil
}

func (repo *CommentRepository) Update(comment *models.Comment) (*models.Comment, error) {
	var existing models.Comment
	if err := repo.DB.First(&existing, "id = ?", comment.ID).Error; err != nil {
		return nil, errors.New("comment not found")
	}

	if err := repo.DB.Save(comment).Error; err != nil {
		return nil, err
	}

	return comment, nil
}

func (repo *CommentRepository) Delete(commentId string) error {
	id, err := uuid.Parse(commentId)
	if err != nil {
		return errors.New("invalid comment ID format")
	}

	result := repo.DB.Delete(&models.Comment{}, "id = ?", id)
	if result.Error != nil {
		return err
	}

	if result.RowsAffected == 0 {
		return errors.New("comment not found")
	}

	return nil
}
