package handlers

import (
	"net/http"

	"wa-server-go/internal/firestore"

	"github.com/gin-gonic/gin"
)

// GetWhatsAppSettings handles GET /settings/whatsapp
func (h *Handler) GetWhatsAppSettings(c *gin.Context) {
	settings, err := h.SettingsRepo.GetWhatsAppSettings(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to get settings"})
		return
	}

	// If empty, fetch from Config as fallback (migration strategy)
	if settings.AgentPhone == "" && h.Config.AgentPhone != "" {
		settings.AgentPhone = h.Config.AgentPhone
	}

	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"settings": settings,
	})
}

// UpdateWhatsAppSettings handles POST /settings/whatsapp
func (h *Handler) UpdateWhatsAppSettings(c *gin.Context) {
	var req struct {
		AgentPhone      string `json:"agentPhone"`
		MainNumber      string `json:"mainNumber"`
		SecondaryNumber string `json:"secondaryNumber"`
		MessageTemplate string `json:"messageTemplate"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid request"})
		return
	}

	settings := &firestore.WhatsAppSettings{
		AgentPhone:      req.AgentPhone,
		MainNumber:      req.MainNumber,
		SecondaryNumber: req.SecondaryNumber,
		MessageTemplate: req.MessageTemplate,
	}

	if err := h.SettingsRepo.UpdateWhatsAppSettings(c.Request.Context(), settings); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to update settings"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Settings updated successfully",
	})
}
