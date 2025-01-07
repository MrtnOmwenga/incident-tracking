package models

import (
	"time"

	"github.com/google/uuid"
)

type Comment struct {
	ID         uuid.UUID      `gorm:"type:uuid;default:uuid_generate_v4()" json:"id"`
	Text       string         `gorm:"type:text;not null" json:"text"`
	IncidentID uuid.UUID      `gorm:"type:uuid;not null" json:"incident_id"`
	Images     []CommentImage `gorm:"foreignKey:CommentID" json:"images"`
	CreatedAt  time.Time      `json:"created_at"`
}
