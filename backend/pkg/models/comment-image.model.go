package models

import (
	"time"

	"github.com/google/uuid"
)

type CommentImage struct {
	ID        uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4()" json:"id"`
	FilePath  string    `gorm:"size:255;not null" json:"file_path"`
	CommentID uuid.UUID `gorm:"type:uuid;not null" json:"comment_id"`
	CreatedAt time.Time `json:"created_at"`
}
