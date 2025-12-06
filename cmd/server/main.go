package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
	"github.com/gin-gonic/gin"

	"github.com/CyberwizD/AI-Document-Summarizer/internal/config"
	"github.com/CyberwizD/AI-Document-Summarizer/internal/db"
	"github.com/CyberwizD/AI-Document-Summarizer/internal/handlers"
	"github.com/CyberwizD/AI-Document-Summarizer/internal/llm"
	"github.com/CyberwizD/AI-Document-Summarizer/internal/models"
	"github.com/CyberwizD/AI-Document-Summarizer/internal/services"
	"github.com/CyberwizD/AI-Document-Summarizer/internal/storage"
)

func main() {
	// Load .env for local development; ignore if missing.
	if err := godotenv.Load(); err != nil {
		log.Printf("warning: .env not loaded: %v", err)
	}

	cfg := config.Load()

	if err := ensureDataDir(cfg.DatabaseURL); err != nil {
		log.Fatalf("prepare data dir: %v", err)
	}

	database := db.Open(cfg.DatabaseURL)
	if err := database.AutoMigrate(&models.Document{}); err != nil {
		log.Fatalf("auto migrate: %v", err)
	}

	s3, err := storage.NewS3Storage(cfg.S3Endpoint, cfg.S3Region, cfg.S3AccessKey, cfg.S3SecretKey, cfg.S3Bucket, cfg.S3UseSSL)
	if err != nil {
		log.Fatalf("init storage: %v", err)
	}

	llmClient := llm.NewClient(cfg.OpenRouterKey, cfg.OpenRouterModel, cfg.RequestTimeout)

	docService := services.NewDocumentService(database, s3, llmClient)
	docHandler := handlers.NewDocumentHandler(docService)

	router := gin.Default()
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	docHandler.RegisterRoutes(router)

	addr := ":" + cfg.ServerPort
	log.Printf("server listening on %s", addr)
	if err := router.Run(addr); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

func ensureDataDir(databaseURL string) error {
	if databaseURL == "" {
		return nil
	}
	// SQLite DSNs have "file:./data/app.db" or "./data/app.db"
	path := databaseURL
	if len(path) > 5 && path[:5] == "file:" {
		path = path[5:]
	}
	dir := filepath.Dir(path)
	if dir == "." || dir == "" {
		return nil
	}
	return os.MkdirAll(dir, 0o755)
}
