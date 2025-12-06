package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/CyberwizD/AI-Document-Summarizer/internal/documents"
	"github.com/CyberwizD/AI-Document-Summarizer/internal/llm"
	"github.com/CyberwizD/AI-Document-Summarizer/internal/models"
)

const maxFileSize int64 = 5 * 1024 * 1024 // 5MB

// ObjectStorage defines the minimal interface needed for storing files.
type ObjectStorage interface {
	Upload(ctx context.Context, key string, reader io.Reader, size int64, contentType string) (string, error)
}

// LLMClient defines the minimal interface to analyze text.
type LLMClient interface {
	AnalyzeDocument(ctx context.Context, text string) (llm.AnalysisResult, error)
}

type DocumentService struct {
	db      *gorm.DB
	storage ObjectStorage
	llm     LLMClient
}

func NewDocumentService(db *gorm.DB, storage ObjectStorage, llmClient LLMClient) *DocumentService {
	return &DocumentService{
		db:      db,
		storage: storage,
		llm:     llmClient,
	}
}

// UploadDocument persists the file to object storage, extracts text, and saves a DB record.
func (s *DocumentService) UploadDocument(ctx context.Context, file multipart.File, header *multipart.FileHeader) (*models.Document, error) {
	if header.Size > maxFileSize {
		return nil, fmt.Errorf("file too large: max %d bytes", maxFileSize)
	}

	ext := strings.ToLower(filepath.Ext(header.Filename))
	if ext != ".pdf" && ext != ".docx" {
		return nil, fmt.Errorf("unsupported file type: %s", ext)
	}

	tmp, err := os.CreateTemp("", "upload-*"+ext)
	if err != nil {
		return nil, fmt.Errorf("create temp file: %w", err)
	}
	defer os.Remove(tmp.Name())

	written, err := io.Copy(tmp, io.LimitReader(file, maxFileSize+1))
	if err != nil {
		return nil, fmt.Errorf("copy file: %w", err)
	}
	if written > maxFileSize {
		return nil, fmt.Errorf("file too large: exceeds %d bytes", maxFileSize)
	}

	if _, err := tmp.Seek(0, 0); err != nil {
		return nil, fmt.Errorf("rewind temp: %w", err)
	}

	mimeType := header.Header.Get("Content-Type")
	extractedText, err := documents.ExtractTextFromFile(tmp.Name(), mimeType)
	if err != nil {
		return nil, fmt.Errorf("extract text: %w", err)
	}

	if _, err := tmp.Seek(0, 0); err != nil {
		return nil, fmt.Errorf("rewind temp for upload: %w", err)
	}

	id := uuid.NewString()
	objectKey := fmt.Sprintf("documents/%s%s", id, ext)
	if _, err := s.storage.Upload(ctx, objectKey, tmp, written, mimeType); err != nil {
		return nil, fmt.Errorf("upload to storage: %w", err)
	}

	doc := &models.Document{
		ID:            id,
		Filename:      header.Filename,
		MimeType:      mimeType,
		SizeBytes:     written,
		StorageKey:    objectKey,
		ExtractedText: extractedText,
	}

	if err := s.db.WithContext(ctx).Create(doc).Error; err != nil {
		return nil, fmt.Errorf("save document: %w", err)
	}
	return doc, nil
}

// AnalyzeDocument sends extracted text to the LLM and stores the response.
func (s *DocumentService) AnalyzeDocument(ctx context.Context, id string) (*models.Document, error) {
	var doc models.Document
	if err := s.db.WithContext(ctx).First(&doc, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("document not found")
		}
		return nil, err
	}

	if strings.TrimSpace(doc.ExtractedText) == "" {
		return nil, fmt.Errorf("document has no extracted text")
	}

	result, err := s.llm.AnalyzeDocument(ctx, doc.ExtractedText)
	if err != nil {
		return nil, fmt.Errorf("call llm: %w", err)
	}

	metaBytes, err := json.Marshal(result.Metadata)
	if err != nil {
		return nil, fmt.Errorf("marshal metadata: %w", err)
	}

	doc.Summary = result.Summary
	doc.DocumentType = result.DocumentType
	doc.Metadata = metaBytes
	doc.UpdatedAt = time.Now()

	if err := s.db.WithContext(ctx).Save(&doc).Error; err != nil {
		return nil, fmt.Errorf("update document: %w", err)
	}

	return &doc, nil
}

func (s *DocumentService) GetDocument(ctx context.Context, id string) (*models.Document, error) {
	var doc models.Document
	if err := s.db.WithContext(ctx).First(&doc, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("document not found")
		}
		return nil, err
	}
	return &doc, nil
}
