package services

import (
	"github.com/google/wire"

	"github.com/MrtnOmwenga/incident-tracking/backend/pkg/repositories"
)

type ServiceContainer struct {
	IncidentService     *IncidentService
	ImageService        *ImageService
	CommentService      *CommentService
	CommentImageService *CommentImageService
}

func NewServiceContainer(repoContainer *repositories.RepositoryContainer) *ServiceContainer {
	return &ServiceContainer{
		IncidentService:     NewIncidentService(repoContainer.IncidentRepo),
		ImageService:        NewImageService(repoContainer.ImageRepo),
		CommentService:      NewCommentService(repoContainer.CommentRepo),
		CommentImageService: NewCommentImageService(repoContainer.CommentImageRepo),
	}
}

var ProviderSet = wire.NewSet(
	NewServiceContainer,
	NewIncidentService,
	NewImageService,
	NewCommentService,
	NewCommentImageService,
)
