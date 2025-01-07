package controllers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"

	"github.com/MrtnOmwenga/incident-tracking/backend/pkg/models"
	"github.com/MrtnOmwenga/incident-tracking/backend/pkg/services"
)

type IncidentController struct {
	Service *services.IncidentService
}

func NewIncidentController(service *services.IncidentService) *IncidentController {
	return &IncidentController{Service: service}
}

func (controller *IncidentController) GetAllIncidents(w http.ResponseWriter, req *http.Request) {
	incidents, err := controller.Service.GetAllIncidents()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(incidents)
}

func (controller *IncidentController) GetIncidentById(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	id := vars["id"]

	incident, err := controller.Service.GetIncidentById(id)
	if err != nil {
		if err.Error() == "incident not found" {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(incident)
}

func (controller *IncidentController) CreateIncident(w http.ResponseWriter, req *http.Request) {
	var incident models.Incident
	if err := json.NewDecoder(req.Body).Decode(&incident); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	created, err := controller.Service.CreateIncident(&incident)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(created)
}

func (controller *IncidentController) UpdateIncident(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	id := vars["id"]

	var incident models.Incident
	if err := json.NewDecoder(req.Body).Decode(&incident); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	parsedId, _ := uuid.Parse(id)
	incident.ID = parsedId

	updated, err := controller.Service.UpdateIncident(&incident)
	if err != nil {
		if err.Error() == "incident not found" {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updated)
}

func (controller *IncidentController) DeleteIncident(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	id := vars["id"]

	if err := controller.Service.DeleteIncident(id); err != nil {
		if err.Error() == "incident not found" {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (controller *IncidentController) GetIncidentChartData(w http.ResponseWriter, req *http.Request) {
	startDateStr := req.URL.Query().Get("startDate")
	endDateStr := req.URL.Query().Get("endDate")

	startDate, err := time.Parse("2006-01-02", startDateStr)
	if err != nil {
		http.Error(w, "Invalid start date format. Use YYYY-MM-DD", http.StatusBadRequest)
		return
	}

	endDate, err := time.Parse("2006-01-02", endDateStr)
	if err != nil {
		http.Error(w, "Invalid end date format. Use YYYY-MM-DD", http.StatusBadRequest)
		return
	}

	chartData, err := controller.Service.GetChartData(startDate, endDate)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(chartData)
}
