package controllers

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/mux"

	"github.com/MrtnOmwenga/incident-tracking/backend/pkg/models"
	"github.com/MrtnOmwenga/incident-tracking/backend/pkg/services"
)

type ImageController struct {
	Service *services.ImageService
}

func NewImageController(service *services.ImageService) *ImageController {
	return &ImageController{Service: service}
}

func (controller *ImageController) GetImagesForIncident(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	incidentID := vars["id"]

	images, err := controller.Service.GetImagesForIncident(incidentID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(images)
}

func (controller *ImageController) GetImageById(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	id := vars["imageId"]

	image, err := controller.Service.GetImageById(id)
	if err != nil {
		if err.Error() == "image not found" {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(image)
}

func (controller *ImageController) UploadImage(w http.ResponseWriter, req *http.Request) {
	var image models.Image
	if err := json.NewDecoder(req.Body).Decode(&image); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	created, err := controller.Service.CreateImage(&image)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(created)
}

func (controller *ImageController) UpdateImage(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	id := vars["id"]

	var image models.Image
	if err := json.NewDecoder(req.Body).Decode(&image); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	parsedId, _ := uuid.Parse(id)
	image.ID = parsedId

	updated, err := controller.Service.UpdateImage(&image)
	if err != nil {
		if err.Error() == "image not found" {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updated)
}

func (controller *ImageController) DeleteImage(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	id := vars["imageId"]

	if err := controller.Service.DeleteImage(id); err != nil {
		if err.Error() == "image not found" {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
