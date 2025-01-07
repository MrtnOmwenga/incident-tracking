package controllers

import (
	"github.com/google/wire"

	"github.com/MrtnOmwenga/incident-tracking/backend/pkg/services"
)

type ControllerContainer struct {
	IncidentController     *IncidentController
	ImageController        *ImageController
	CommentController      *CommentController
	CommentImageController *CommentImageController
}

func NewControllerContainer(serviceContainer *services.ServiceContainer) *ControllerContainer {
	return &ControllerContainer{
		IncidentController:     NewIncidentController(serviceContainer.IncidentService),
		ImageController:        NewImageController(serviceContainer.ImageService),
		CommentController:      NewCommentController(serviceContainer.CommentService),
		CommentImageController: NewCommentImageController(serviceContainer.CommentImageService),
	}
}

var ProviderSet = wire.NewSet(
	NewControllerContainer,
	NewIncidentController,
	NewImageController,
	NewCommentController,
	NewCommentImageController,
)
