package documents

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	pdf "github.com/ledongthuc/pdf"
)

// ExtractTextFromFile loads a PDF or DOCX file and returns extracted text.
func ExtractTextFromFile(path string, mimeType string) (string, error) {
	ext := strings.ToLower(filepath.Ext(path))
	if mimeType == "" {
		mimeType = mimeFromExt(ext)
	}

	switch {
	case strings.Contains(mimeType, "pdf") || ext == ".pdf":
		return extractFromPDF(path)
	case strings.Contains(mimeType, "word") || strings.Contains(mimeType, "docx") || ext == ".docx":
		return extractFromDocx(path)
	default:
		return "", fmt.Errorf("unsupported file type: %s", mimeType)
	}
}

func mimeFromExt(ext string) string {
	switch ext {
	case ".pdf":
		return "application/pdf"
	case ".docx":
		return "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	default:
		return ""
	}
}

func extractFromPDF(path string) (string, error) {
	f, r, err := pdf.Open(path)
	if err != nil {
		return "", fmt.Errorf("open pdf: %w", err)
	}
	defer f.Close()

	var sb strings.Builder
	totalPage := r.NumPage()
	for i := 1; i <= totalPage; i++ {
		p := r.Page(i)
		if p.V.IsNull() {
			continue
		}
		text, err := p.GetPlainText(nil)
		if err != nil {
			return "", fmt.Errorf("read pdf page %d: %w", i, err)
		}
		sb.WriteString(text)
		sb.WriteString("\n")
	}
	return sb.String(), nil
}

func extractFromDocx(path string) (string, error) {
	file, err := zip.OpenReader(path)
	if err != nil {
		return "", fmt.Errorf("open docx zip: %w", err)
	}
	defer file.Close()

	var docFile *zip.File
	for _, f := range file.File {
		if f.Name == "word/document.xml" {
			docFile = f
			break
		}
	}
	if docFile == nil {
		return "", fmt.Errorf("document.xml not found in docx")
	}

	rc, err := docFile.Open()
	if err != nil {
		return "", fmt.Errorf("open document.xml: %w", err)
	}
	defer rc.Close()

	data, err := io.ReadAll(rc)
	if err != nil {
		return "", fmt.Errorf("read document.xml: %w", err)
	}

	decoder := xml.NewDecoder(bytes.NewReader(data))
	var sb strings.Builder
	for {
		tok, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", fmt.Errorf("decode docx xml: %w", err)
		}
		switch el := tok.(type) {
		case xml.StartElement:
			if el.Name.Local == "t" {
				var text string
				if err := decoder.DecodeElement(&text, &el); err != nil {
					return "", fmt.Errorf("decode text element: %w", err)
				}
				sb.WriteString(text)
			}
		case xml.EndElement:
			if el.Name.Local == "p" {
				sb.WriteString("\n")
			}
		}
	}
	return sb.String(), nil
}
