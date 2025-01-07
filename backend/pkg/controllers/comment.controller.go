package controllers

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/mux"

	"github.com/MrtnOmwenga/incident-tracking/backend/pkg/models"
	"github.com/MrtnOmwenga/incident-tracking/backend/pkg/services"
)

type CommentController struct {
	Service *services.CommentService
}

func NewCommentController(service *services.CommentService) *CommentController {
	return &CommentController{Service: service}
}

func (controller *CommentController) GetAllComments(w http.ResponseWriter, req *http.Request) {
	comments, err := controller.Service.GetAllComments()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(comments)
}

func (controller *CommentController) GetCommentById(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	id := vars["commentId"]

	comment, err := controller.Service.GetCommentById(id)
	if err != nil {
		if err.Error() == "comment not found" {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(comment)
}

func (controller *CommentController) GetCommentsForIncident(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	incidentID := vars["id"]

	comments, err := controller.Service.GetCommentsForIncident(incidentID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(comments)
}

func (controller *CommentController) CreateComment(w http.ResponseWriter, req *http.Request) {
	var comment models.Comment
	if err := json.NewDecoder(req.Body).Decode(&comment); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	created, err := controller.Service.CreateComment(&comment)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(created)
}

func (controller *CommentController) UpdateComment(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	id := vars["id"]

	var comment models.Comment
	if err := json.NewDecoder(req.Body).Decode(&comment); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	parsedId, _ := uuid.Parse(id)
	comment.ID = parsedId

	updated, err := controller.Service.CreateComment(&comment)
	if err != nil {
		if err.Error() == "Comment not found" {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updated)
}

func (controller *CommentController) DeleteComment(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	id := vars["id"]

	if err := controller.Service.DeleteComment(id); err != nil {
		if err.Error() == "Comment not found" {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
