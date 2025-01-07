package models

import (
	"github.com/google/uuid"
)

type Image struct {
	ID         uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4()" json:"id"`
	FilePath   string    `gorm:"size:255;not null" json:"file_path"`
	IncidentID uuid.UUID `gorm:"type:uuid;not null" json:"incident_id"`
}
