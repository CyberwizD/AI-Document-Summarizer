package tests

import (
	"archive/zip"
	"bytes"
	"os"
	"testing"

	"github.com/CyberwizD/AI-Document-Summarizer/internal/documents"
)

func TestExtractTextFromDocx(t *testing.T) {
	// Build a minimal DOCX file in memory
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.Create("word/document.xml")
	if err != nil {
		t.Fatalf("create entry: %v", err)
	}
	_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:body>
    <w:p><w:r><w:t>Sample Docx Text</w:t></w:r></w:p>
  </w:body>
</w:document>`))
	if err := zw.Close(); err != nil {
		t.Fatalf("close zip: %v", err)
	}

	tmp, err := os.CreateTemp("", "docx-*.docx")
	if err != nil {
		t.Fatalf("create temp: %v", err)
	}
	defer os.Remove(tmp.Name())

	if _, err := tmp.Write(buf.Bytes()); err != nil {
		t.Fatalf("write temp: %v", err)
	}
	tmp.Close()

	text, err := documents.ExtractTextFromFile(tmp.Name(), "application/vnd.openxmlformats-officedocument.wordprocessingml.document")
	if err != nil {
		t.Fatalf("ExtractTextFromFile error: %v", err)
	}
	if want := "Sample Docx Text\n"; text != want {
		t.Fatalf("expected %q, got %q", want, text)
	}
}
