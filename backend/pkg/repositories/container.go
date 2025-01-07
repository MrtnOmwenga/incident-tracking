package repositories

import (
	"github.com/google/wire"
	"gorm.io/gorm"
)

type RepositoryContainer struct {
	IncidentRepo     *IncidentRepository
	ImageRepo        *ImageRepository
	CommentRepo      *CommentRepository
	CommentImageRepo *CommentImageRepository
}

func NewRepositoryContainer(db *gorm.DB) *RepositoryContainer {
	return &RepositoryContainer{
		IncidentRepo:     NewIncidentRepository(db),
		ImageRepo:        NewImageRepository(db),
		CommentRepo:      NewCommentRepository(db),
		CommentImageRepo: NewCommentImageRepository(db),
	}
}

var ProviderSet = wire.NewSet(
	NewRepositoryContainer,
	NewIncidentRepository,
	NewImageRepository,
	NewCommentRepository,
	NewCommentImageRepository,
)
