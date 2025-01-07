package services

import (
	"github.com/MrtnOmwenga/incident-tracking/backend/pkg/models"
	"github.com/MrtnOmwenga/incident-tracking/backend/pkg/repositories"
)

type CommentImageService struct {
	Repo *repositories.CommentImageRepository
}

func NewCommentImageService(repo *repositories.CommentImageRepository) *CommentImageService {
	return &CommentImageService{Repo: repo}
}

func (service *CommentImageService) GetImagesForComment(incidentID string) ([]models.CommentImage, error) {
	return service.Repo.GetByCommentID(incidentID)
}

func (service *CommentImageService) GetCommentImageById(id string) (*models.CommentImage, error) {
	return service.Repo.GetById(id)
}

func (service *CommentImageService) CreateCommentImage(commentImage *models.CommentImage) (*models.CommentImage, error) {
	return service.Repo.Create(commentImage)
}

func (service *CommentImageService) UpdateCommentImage(commentImage *models.CommentImage) (*models.CommentImage, error) {
	return service.Repo.Update(commentImage)
}

func (service *CommentImageService) DeleteCommentImage(id string) error {
	return service.Repo.Delete(id)
}
