package services

import (
	"github.com/MrtnOmwenga/incident-tracking/backend/pkg/models"
	"github.com/MrtnOmwenga/incident-tracking/backend/pkg/repositories"
)

type CommentService struct {
	Repo *repositories.CommentRepository
}

func NewCommentService(repo *repositories.CommentRepository) *CommentService {
	return &CommentService{Repo: repo}
}

func (service *CommentService) GetAllComments() ([]models.Comment, error) {
	return service.Repo.GetAll()
}

func (service *CommentService) GetCommentsForIncident(incidentID string) ([]models.Comment, error) {
	return service.Repo.GetByIncidentID(incidentID)
}

func (service *CommentService) GetCommentById(id string) (*models.Comment, error) {
	return service.Repo.GetById(id)
}

func (service *CommentService) CreateComment(Comment *models.Comment) (*models.Comment, error) {
	return service.Repo.Create(Comment)
}

func (service *CommentService) UpdateComment(Comment *models.Comment) (*models.Comment, error) {
	return service.Repo.Update(Comment)
}

func (service *CommentService) DeleteComment(id string) error {
	return service.Repo.Delete(id)
}
