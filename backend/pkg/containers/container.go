package container

import (
	"github.com/google/wire"
	"github.com/gorilla/mux"

	"github.com/MrtnOmwenga/incident-tracking/backend/pkg/controllers"
	"github.com/MrtnOmwenga/incident-tracking/backend/pkg/repositories"
	"github.com/MrtnOmwenga/incident-tracking/backend/pkg/routes"
	"github.com/MrtnOmwenga/incident-tracking/backend/pkg/services"
)

type AppContainer struct {
	Controllers  *controllers.ControllerContainer
	Services     *services.ServiceContainer
	Repositories *repositories.RepositoryContainer
	Router       *mux.Router
}

func ProvideRouter() *mux.Router {
	return mux.NewRouter()
}

func ProvideAppContainer(
	controllers *controllers.ControllerContainer,
	services *services.ServiceContainer,
	repositories *repositories.RepositoryContainer,
	router *mux.Router,
) *AppContainer {
	return &AppContainer{
		Controllers:  controllers,
		Services:     services,
		Repositories: repositories,
		Router:       router,
	}
}

func (ac *AppContainer) GetRouter() *mux.Router {
	routes.RegisterRoutes(
		ac.Router,
		ac.Controllers.IncidentController,
		ac.Controllers.ImageController,
		ac.Controllers.CommentController,
		ac.Controllers.CommentImageController,
	)

	return ac.Router
}

var ProviderSet = wire.NewSet(
	ProvideRouter,
	ProvideAppContainer,
	controllers.ProviderSet,
	services.ProviderSet,
	repositories.ProviderSet,
)
