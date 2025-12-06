package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/CyberwizD/AI-Document-Summarizer/internal/services"
)

type DocumentHandler struct {
	service *services.DocumentService
}

func NewDocumentHandler(svc *services.DocumentService) *DocumentHandler {
	return &DocumentHandler{service: svc}
}

func (h *DocumentHandler) RegisterRoutes(router *gin.Engine) {
	router.POST("/documents/upload", h.upload)
	router.POST("/documents/:id/analyze", h.analyze)
	router.GET("/documents/:id", h.getDocument)
}

func (h *DocumentHandler) upload(c *gin.Context) {
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file is required"})
		return
	}
	defer file.Close()

	doc, err := h.service.UploadDocument(c.Request.Context(), file, header)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, doc)
}

func (h *DocumentHandler) analyze(c *gin.Context) {
	id := c.Param("id")
	doc, err := h.service.AnalyzeDocument(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, doc)
}

func (h *DocumentHandler) getDocument(c *gin.Context) {
	id := c.Param("id")
	doc, err := h.service.GetDocument(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, doc)
}
