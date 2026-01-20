package utils

import (
	"regexp"
	"strings"

	"go.mau.fi/whatsmeow/types"
)

var nonDigitRegex = regexp.MustCompile(`\D`)

// FormatPhoneNumber formats a phone number to WhatsApp format (628xxx)
func FormatPhoneNumber(number string) string {
	// Remove all non-digits
	phone := nonDigitRegex.ReplaceAllString(number, "")

	// Convert 08xx to 628xx
	if strings.HasPrefix(phone, "0") {
		phone = "62" + phone[1:]
	}

	// Add 62 if starts with 8
	if strings.HasPrefix(phone, "8") {
		phone = "62" + phone
	}

	return phone
}

// PhoneToJID converts a phone number to WhatsApp JID
func PhoneToJID(number string) types.JID {
	phone := FormatPhoneNumber(number)
	return types.NewJID(phone, types.DefaultUserServer)
}

// JIDToPhoneNumber extracts the user (number) part from a JID
func JIDToPhoneNumber(jid types.JID) string {
	return jid.User
}

// FormatPhoneForDisplay formats phone for display (08xxx)
func FormatPhoneForDisplay(number string) string {
	phone := FormatPhoneNumber(number)
	if strings.HasPrefix(phone, "62") {
		return "0" + phone[2:]
	}
	return phone
}

// NormalizeNewlines converts all newline types to LF
func NormalizeNewlines(text string) string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	return text
}

// Truncate shortens a string to maxLen characters
func Truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// GetExtensionFromMimetype returns file extension for a MIME type
func GetExtensionFromMimetype(mimetype string) string {
	extensions := map[string]string{
		"application/pdf":  ".pdf",
		"image/jpeg":       ".jpg",
		"image/png":        ".png",
		"image/gif":        ".gif",
		"image/webp":       ".webp",
		"video/mp4":        ".mp4",
		"audio/mpeg":       ".mp3",
		"audio/ogg":        ".ogg",
		"application/zip":  ".zip",
		"application/msword": ".doc",
		"application/vnd.openxmlformats-officedocument.wordprocessingml.document": ".docx",
		"application/vnd.ms-excel": ".xls",
		"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet": ".xlsx",
	}

	if ext, ok := extensions[mimetype]; ok {
		return ext
	}

	// Try to extract from mimetype (e.g., "image/png" -> ".png")
	parts := strings.Split(mimetype, "/")
	if len(parts) == 2 {
		return "." + parts[1]
	}

	return ""
}
