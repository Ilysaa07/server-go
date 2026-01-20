package handlers

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.mau.fi/whatsmeow/appstate"
)

// SyncContacts handles POST /sync-contacts
// Fetches contacts from Number B (Leads) filtered by "Leads for Web" label
func (h *Handler) SyncContacts(c *gin.Context) {
	// Auto-Start 'leads' client logic
	clientID := "leads"
	client, exists := h.WAManager.GetClient(clientID)

	if !exists {
		// Create and Connect
		err := h.WAManager.CreateClient(context.Background(), clientID, "session-leads.db")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
			return
		}
		_ = h.WAManager.SetupEventHandlers(clientID)
		
		go func() {
			_ = h.WAManager.Connect(context.Background(), clientID)
		}()

		c.JSON(http.StatusOK, gin.H{
			"success":    false,
			"requiresQR": true,
			"message":    "Session started. Please scan QR code.",
		})
		return
	}

	if !client.IsReady() {
		c.JSON(http.StatusOK, gin.H{
			"success":    false,
			"requiresQR": true,
			"message":    "Session not ready. Please scan QR code.",
		})
		return
	}

	ctx := context.Background()

	// Sync app state to get latest labels (no QR reconnect needed!)
	// Labels can be in different app state patches - try multiple
	fmt.Printf("🏷️ [leads] Fetching app state for labels...\n")
	
	// Try Regular patch (most label data is here)
	if err := client.WAClient.FetchAppState(ctx, appstate.WAPatchRegular, false, false); err != nil {
		fmt.Printf("⚠️ Failed to fetch WAPatchRegular: %v\n", err)
	}
	
	// Try RegularLow patch
	if err := client.WAClient.FetchAppState(ctx, appstate.WAPatchRegularLow, false, false); err != nil {
		fmt.Printf("⚠️ Failed to fetch WAPatchRegularLow: %v\n", err)
	}
	
	// Try RegularHigh patch (label associations might be here)
	if err := client.WAClient.FetchAppState(ctx, appstate.WAPatchRegularHigh, false, false); err != nil {
		fmt.Printf("⚠️ Failed to fetch WAPatchRegularHigh: %v\n", err)
	}
	
	fmt.Printf("🏷️ [leads] App state fetch completed. Labels in store: %d\n", len(h.WAManager.LabelStore.GetAllLabels()))

	// Get all contacts first
	contacts, err := client.WAClient.Store.Contacts.GetAllContacts(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to fetch contacts"})
		return
	}

	// Get JIDs that have "Leads for Web" label
	targetLabel := "Leads for Web"
	labeledJIDs := h.WAManager.LabelStore.GetJIDsForLabelName(targetLabel)
	labeledSet := make(map[string]bool)
	for _, jid := range labeledJIDs {
		labeledSet[jid] = true
	}

	// Log label store status for debugging
	allLabels := h.WAManager.LabelStore.GetAllLabels()
	fmt.Printf("🏷️ Available labels in store: %v\n", allLabels)
	fmt.Printf("🏷️ JIDs with '%s' label: %d\n", targetLabel, len(labeledJIDs))

	// Filter contacts by label
	filtered := len(labeledSet) > 0
	result := make([]map[string]interface{}, 0)
	
	for jid, contact := range contacts {
		// Skip if we have labels but this contact doesn't have the target label
		if filtered && !labeledSet[jid.User] {
			continue
		}
		
		if contact.FullName == "" && contact.PushName == "" {
			continue
		}
		name := contact.FullName
		if name == "" {
			name = contact.PushName
		}
		
		result = append(result, map[string]interface{}{
			"id":    jid.User,
			"name":  name,
			"phone": jid.User,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"success":       true,
		"count":         len(result),
		"contacts":      result,
		"filtered":      filtered,
		"targetLabel":   targetLabel,
		"labelsInStore": len(allLabels),
	})
}

// StartLeadsClient handles POST /start-leads-client
func (h *Handler) StartLeadsClient(c *gin.Context) {
	clientID := "leads"
	
	// Check if already exists
	if client, exists := h.WAManager.GetClient(clientID); exists {
		if client.IsReady() {
			c.JSON(http.StatusOK, gin.H{
				"success": true,
				"message": "Leads client is already running and ready",
			})
			return
		}
		// If exists but not ready, try to connect again logic could go here
		// For now assume if exists we just let it be or user should stop first
	}

	// Create client
	// Use a separate DB for leads session
	err := h.WAManager.CreateClient(context.Background(), clientID, "session-leads.db")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to create leads client: " + err.Error(),
		})
		return
	}

	// Setup events
	err = h.WAManager.SetupEventHandlers(clientID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to setup event handlers: " + err.Error(),
		})
		return
	}

	// Connect in background
	go func() {
		// Use background context for long running connection
		err := h.WAManager.Connect(context.Background(), clientID)
		if err != nil {
			// Log error (can't write to response anymore)
			fmt.Printf("❌ Failed to connect leads client: %v\n", err)
		}
	}()

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Leads client starting... please scan QR code",
	})
}

// StopLeadsClient handles POST /stop-leads-client
func (h *Handler) StopLeadsClient(c *gin.Context) {
	clientID := "leads"
	
	err := h.WAManager.DestroyClient(clientID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to stop leads client: " + err.Error(),
		})
		return
	}

	// Remove DB file to ensure fresh start next time (optional, maybe keep for persistence)
	// For "on-demand" usually we want persistence so don't delete DB
	// os.Remove("session-leads.db")

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Leads client stopped successfully",
	})
}
