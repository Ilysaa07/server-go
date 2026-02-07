package firestore

import (
	"context"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// WhatsAppSettings represents the dynamic settings for WhatsApp
type WhatsAppSettings struct {
	AgentPhone      string    `firestore:"agentPhone" json:"agentPhone"`
	MainNumber      string    `firestore:"mainNumber" json:"mainNumber"`
	SecondaryNumber string    `firestore:"secondaryNumber" json:"secondaryNumber"`
	MessageTemplate string    `firestore:"messageTemplate" json:"messageTemplate"`
	UpdatedAt       time.Time `firestore:"updatedAt" json:"updatedAt"`
}

// SettingsRepository provides access to the settings collection
type SettingsRepository struct {
	client             *Client
	settingsCollection string
}

// NewSettingsRepository creates a new settings repository
func NewSettingsRepository(client *Client) *SettingsRepository {
	return &SettingsRepository{
		client:             client,
		settingsCollection: "settings",
	}
}

// GetWhatsAppSettings retrieves the WhatsApp settings
func (r *SettingsRepository) GetWhatsAppSettings(ctx context.Context) (*WhatsAppSettings, error) {
	doc, err := r.client.Collection(r.settingsCollection).Doc("whatsapp_settings").Get(ctx)
	if status.Code(err) == codes.NotFound {
		// Return default empty settings if not found
		return &WhatsAppSettings{
			AgentPhone: "",
			UpdatedAt:  time.Now(),
		}, nil
	}
	if err != nil {
		return nil, err
	}

	var settings WhatsAppSettings
	if err := doc.DataTo(&settings); err != nil {
		return nil, err
	}

	return &settings, nil
}

// UpdateWhatsAppSettings updates the WhatsApp settings
func (r *SettingsRepository) UpdateWhatsAppSettings(ctx context.Context, settings *WhatsAppSettings) error {
	settings.UpdatedAt = time.Now()
	// Use Set with Merge to avoid overwriting other fields (like mainNumber used by frontend)
	_, err := r.client.Collection(r.settingsCollection).Doc("whatsapp_settings").Set(ctx, settings, firestore.MergeAll)
	return err
}
