package tests

import (
	"archive/zip"
	"bytes"
	"context"
	"io"
	"mime/multipart"
	"net/textproto"
	"os"
	"path/filepath"
	"testing"

	"github.com/CyberwizD/AI-Document-Summarizer/internal/llm"
	"github.com/CyberwizD/AI-Document-Summarizer/internal/models"
	"github.com/CyberwizD/AI-Document-Summarizer/internal/services"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type mockStorage struct {
	uploads []string
}

func (m *mockStorage) Upload(ctx context.Context, key string, reader io.Reader, size int64, contentType string) (string, error) {
	m.uploads = append(m.uploads, key)
	_, _ = io.Copy(io.Discard, reader)
	return key, nil
}

type mockLLM struct {
	result llm.AnalysisResult
	err    error
}

func (m *mockLLM) AnalyzeDocument(ctx context.Context, text string) (llm.AnalysisResult, error) {
	return m.result, m.err
}

func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&models.Document{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func makeDocxFile(t *testing.T) (*os.File, *multipart.FileHeader, func()) {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.Create("word/document.xml")
	if err != nil {
		t.Fatalf("create xml entry: %v", err)
	}
	_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:body>
    <w:p><w:r><w:t>Hello World</w:t></w:r></w:p>
  </w:body>
</w:document>`))
	if err := zw.Close(); err != nil {
		t.Fatalf("close zip: %v", err)
	}

	tmp, err := os.CreateTemp("", "doc-*.docx")
	if err != nil {
		t.Fatalf("create temp: %v", err)
	}
	if _, err := tmp.Write(buf.Bytes()); err != nil {
		t.Fatalf("write temp: %v", err)
	}
	if _, err := tmp.Seek(0, 0); err != nil {
		t.Fatalf("rewind: %v", err)
	}

	stat, _ := tmp.Stat()
	header := &multipart.FileHeader{
		Filename: filepath.Base(tmp.Name()),
		Size:     stat.Size(),
		Header: textproto.MIMEHeader{
			"Content-Type": []string{"application/vnd.openxmlformats-officedocument.wordprocessingml.document"},
		},
	}

	cleanup := func() { os.Remove(tmp.Name()) }
	return tmp, header, cleanup
}

func TestUploadDocument_Succeeds(t *testing.T) {
	db := setupTestDB(t)
	storage := &mockStorage{}
	llmClient := &mockLLM{}
	svc := services.NewDocumentService(db, storage, llmClient)

	file, header, cleanup := makeDocxFile(t)
	defer cleanup()
	defer file.Close()

	doc, err := svc.UploadDocument(context.Background(), file, header)
	if err != nil {
		t.Fatalf("UploadDocument returned error: %v", err)
	}
	if doc.ID == "" {
		t.Fatalf("expected ID to be set")
	}
	if len(storage.uploads) != 1 {
		t.Fatalf("expected upload to be called once, got %d", len(storage.uploads))
	}
	var count int64
	if err := db.Model(&models.Document{}).Count(&count).Error; err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 document in db, got %d", count)
	}
}

func TestAnalyzeDocument_UpdatesRecord(t *testing.T) {
	db := setupTestDB(t)
	storage := &mockStorage{}
	llmClient := &mockLLM{
		result: llm.AnalysisResult{
			Summary:      "Short summary",
			DocumentType: "report",
			Metadata: map[string]interface{}{
				"date": "2025-01-01",
			},
		},
	}
	svc := services.NewDocumentService(db, storage, llmClient)

	doc := &models.Document{
		ID:            "doc-123",
		Filename:      "file.docx",
		MimeType:      "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		ExtractedText: "hello world",
	}
	if err := db.Create(doc).Error; err != nil {
		t.Fatalf("seed doc: %v", err)
	}

	updated, err := svc.AnalyzeDocument(context.Background(), doc.ID)
	if err != nil {
		t.Fatalf("AnalyzeDocument returned error: %v", err)
	}
	if updated.Summary != "Short summary" {
		t.Fatalf("expected summary saved, got %q", updated.Summary)
	}
	if updated.DocumentType != "report" {
		t.Fatalf("expected type saved, got %q", updated.DocumentType)
	}
	if string(updated.Metadata) == "" {
		t.Fatalf("expected metadata to be saved")
	}
}
