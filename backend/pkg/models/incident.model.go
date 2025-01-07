package models

import (
	"time"

	"github.com/google/uuid"
)

type Status string

const (
	StatusOpen       Status = "open"
	StatusInProgress Status = "in_progress"
	StatusResolved   Status = "resolved"
	StatusClosed     Status = "closed"
)

type Severity string

const (
	SeverityHigh   Severity = "high"
	SeverityMedium Severity = "medium"
	SeverityLow    Severity = "low"
)

type Incident struct {
	ID          uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4()" json:"id"`
	Title       string    `gorm:"size:255;not null" json:"title"`
	Description string    `gorm:"type:text;not null" json:"description"`
	Status      Status    `gorm:"type:\"Status\";default:'open'" json:"status"`
	Severity    Severity  `gorm:"type:\"Severity\";default:'low'" json:"severity"`
	Images      []Image   `gorm:"foreignKey:IncidentID" json:"images"`
	Comments    []Comment `gorm:"foreignKey:IncidentID" json:"comments"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
