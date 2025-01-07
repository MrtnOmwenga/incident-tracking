package routes

import (
	"net/http"
	"github.com/gorilla/mux"

	"github.com/MrtnOmwenga/incident-tracking/backend/pkg/controllers"
	"github.com/MrtnOmwenga/incident-tracking/backend/pkg/middleware"
)

func RegisterRoutes(
	router *mux.Router,
	incidentController *controllers.IncidentController,
	imageController *controllers.ImageController,
	commentController *controllers.CommentController,
	commentImageController *controllers.CommentImageController,
) {
	apiRouter := router.PathPrefix("/api/v1").Subrouter()

	// Middleware that applies to all routes
	apiRouter.Use(middleware.JSONContentType)
	apiRouter.Use(middleware.Logging)

	// Incidents endpoints
	incidents := apiRouter.PathPrefix("/incidents").Subrouter()
	incidents.HandleFunc("", incidentController.GetAllIncidents).Methods("GET")
	incidents.HandleFunc("", incidentController.CreateIncident).Methods("POST")
	incidents.HandleFunc("/chart-data", incidentController.GetIncidentChartData).Methods("GET")
	incidents.HandleFunc("/{id}", incidentController.UpdateIncident).Methods("PUT")
	incidents.HandleFunc("/{id}", incidentController.GetIncidentById).Methods("GET")
	incidents.HandleFunc("/{id}", incidentController.DeleteIncident).Methods("DELETE")

	// Images endpoints
	images := apiRouter.PathPrefix("/incidents/{id}/images").Subrouter()
	images.HandleFunc("", imageController.UploadImage).Methods("POST")
	images.HandleFunc("", imageController.GetImagesForIncident).Methods("GET")
	images.HandleFunc("/{imageId}", imageController.GetImageById).Methods("GET")
	images.HandleFunc("/{imageId}", imageController.DeleteImage).Methods("DELETE")

	// Comments endpoints
	comments := apiRouter.PathPrefix("/incidents/{id}/comments").Subrouter()
	comments.HandleFunc("", commentController.CreateComment).Methods("POST")
	comments.HandleFunc("", commentController.GetCommentsForIncident).Methods("GET")
	comments.HandleFunc("/{commentId}", commentController.UpdateComment).Methods("PUT")
	comments.HandleFunc("/{commentId}", commentController.GetCommentById).Methods("GET")
	comments.HandleFunc("/{commentId}", commentController.DeleteComment).Methods("DELETE")

	// Comment images endpoints
	commentImages := apiRouter.PathPrefix("/incidents/{id}/comments/images").Subrouter()
	commentImages.HandleFunc("", commentImageController.GetImagesForComment).Methods("GET")
	commentImages.HandleFunc("", commentImageController.UploadCommentImage).Methods("POST")
	commentImages.HandleFunc("/{commentImageId}", commentImageController.UpdateCommentImage).Methods("PUT")
	commentImages.HandleFunc("/{commentImageId}", commentImageController.GetCommentImageById).Methods("GET")
	commentImages.HandleFunc("/{commentImageId}", commentImageController.DeleteCommentImage).Methods("DELETE")

	apiRouter.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}
