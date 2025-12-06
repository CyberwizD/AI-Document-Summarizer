package models

import (
	"time"

	"gorm.io/datatypes"
)

type Document struct {
	ID            string         `gorm:"primaryKey;size:36" json:"id"`
	Filename      string         `json:"filename"`
	MimeType      string         `json:"mime_type"`
	SizeBytes     int64          `json:"size_bytes"`
	StorageKey    string         `json:"storage_key"`
	ExtractedText string         `json:"extracted_text"`
	Summary       string         `json:"summary"`
	DocumentType  string         `json:"document_type"`
	Metadata      datatypes.JSON `json:"metadata"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
}
