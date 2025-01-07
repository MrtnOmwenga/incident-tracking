package controllers

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/mux"

	"github.com/MrtnOmwenga/incident-tracking/backend/pkg/models"
	"github.com/MrtnOmwenga/incident-tracking/backend/pkg/services"
)

type CommentImageController struct {
	Service *services.CommentImageService
}

func NewCommentImageController(service *services.CommentImageService) *CommentImageController {
	return &CommentImageController{Service: service}
}

func (controller *CommentImageController) GetImagesForComment(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	incidentID := vars["id"]

	commentImages, err := controller.Service.GetImagesForComment(incidentID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(commentImages)
}

func (controller *CommentImageController) GetCommentImageById(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	id := vars["commentImageId"]

	commentImage, err := controller.Service.GetCommentImageById(id)
	if err != nil {
		if err.Error() == "comment image not found" {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(commentImage)
}

func (controller *CommentImageController) UploadCommentImage(w http.ResponseWriter, req *http.Request) {
	var commentImage models.CommentImage
	if err := json.NewDecoder(req.Body).Decode(&commentImage); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	created, err := controller.Service.CreateCommentImage(&commentImage)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(created)
}

func (controller *CommentImageController) UpdateCommentImage(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	id := vars["id"]

	var commentImage models.CommentImage
	if err := json.NewDecoder(req.Body).Decode(&commentImage); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	parsedId, _ := uuid.Parse(id)
	commentImage.ID = parsedId

	updated, err := controller.Service.UpdateCommentImage(&commentImage)
	if err != nil {
		if err.Error() == "comment image not found" {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updated)
}

func (controller *CommentImageController) DeleteCommentImage(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	id := vars["commentImageId"]

	if err := controller.Service.DeleteCommentImage(id); err != nil {
		if err.Error() == "comment image not found" {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
