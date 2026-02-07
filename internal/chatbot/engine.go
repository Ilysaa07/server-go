package chatbot

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// GroqClient handles communication with Groq API
type GroqClient struct {
	apiKey     string
	httpClient *http.Client
	model      string
}

// GroqMessage represents a message in the Groq chat format
type GroqMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// GroqRequest represents a request to Groq API
type GroqRequest struct {
	Model       string        `json:"model"`
	Messages    []GroqMessage `json:"messages"`
	MaxTokens   int           `json:"max_tokens"`
	Temperature float64       `json:"temperature"`
}

// GroqResponse represents a response from Groq API
type GroqResponse struct {
	ID      string `json:"id"`
	Choices []struct {
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error"`
}

// ChatEngine handles AI-powered chat responses
type ChatEngine struct {
	groq          *GroqClient
	knowledgeBase []KnowledgeItem
	systemPrompt  string
}

// KnowledgeItem represents a piece of knowledge from the database
type KnowledgeItem struct {
	Topic    string   `json:"topic"`
	Question string   `json:"question"`
	Answer   string   `json:"answer"`
	Keywords []string `json:"keywords"`
}

// ChatResponse represents the response from the chat engine
type ChatResponse struct {
	Reply           string  `json:"reply"`
	Confidence      float64 `json:"confidence"`
	SuggestHandover bool    `json:"suggestHandover"`
	Sentiment       string  `json:"sentiment"` // positive, neutral, frustrated
}

// NewGroqClient creates a new Groq API client
func NewGroqClient(apiKey string) *GroqClient {
	return &GroqClient{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		model: "llama-3.3-70b-versatile", // Fast and capable model
	}
}

// NewChatEngine creates a new chat engine with the given Groq API key
func NewChatEngine(groqAPIKey string) *ChatEngine {
	systemPrompt := `Kamu adalah asisten virtual Valpro Intertech, perusahaan jasa legalitas dan perizinan usaha di Indonesia.

PERAN:
- Menjawab pertanyaan tentang layanan: Pendirian PT, CV, Sertifikasi ISO, SBU Konstruksi, HAKI, NIB, dan lainnya
- Memberikan informasi umum tentang proses dan persyaratan legalitas
- Mengarahkan pengunjung ke layanan yang tepat

GAYA KOMUNIKASI:
- Ramah, profesional, dan informatif
- Gunakan Bahasa Indonesia yang baik
- Jawaban singkat dan padat (2-3 kalimat jika memungkinkan)
- Gunakan emoji secukupnya untuk kesan friendly

BATASAN:
- JANGAN memberikan harga spesifik (katakan "untuk detail harga, silakan konsultasi dengan tim kami")
- JANGAN memberikan jaminan waktu penyelesaian yang pasti
- Jika tidak yakin, sarankan untuk bicara dengan admin
- HANYA menjawab topik: Legalitas, Hukum Bisnis, Perizinan, dan Layanan Perusahaan
- TOLAK pertanyaan di luar topik (curhat, coding, politik, dll) dengan sopan: "Maaf, saya hanya bisa membantu seputar legalitas & bisnis."

CONTOH LAYANAN:
- Pendirian PT/CV: Pembuatan akta, SK Kemenkumham, NIB
- Sertifikasi ISO: ISO 9001, 14001, 45001, 27001
- SBU Konstruksi: Sertifikat Badan Usaha dari LPJK
- HAKI: Pendaftaran Merek, Paten, Hak Cipta`

	return &ChatEngine{
		groq:          NewGroqClient(groqAPIKey),
		knowledgeBase: []KnowledgeItem{},
		systemPrompt:  systemPrompt,
	}
}

// SetKnowledgeBase updates the knowledge base
func (e *ChatEngine) SetKnowledgeBase(items []KnowledgeItem) {
	e.knowledgeBase = items
}

// ProcessMessage processes a user message and returns an AI response
func (e *ChatEngine) ProcessMessage(ctx context.Context, userMessage string, conversationHistory []GroqMessage) (*ChatResponse, error) {
	// Analyze sentiment first
	sentiment := e.analyzeSentiment(userMessage)

	// Build context from knowledge base
	relevantKnowledge := e.findRelevantKnowledge(userMessage)
	
	// Build messages array
	messages := []GroqMessage{
		{Role: "system", Content: e.buildSystemPrompt(relevantKnowledge)},
	}
	
	// Add conversation history (last 10 messages)
	historyStart := 0
	if len(conversationHistory) > 10 {
		historyStart = len(conversationHistory) - 10
	}
	messages = append(messages, conversationHistory[historyStart:]...)
	
	// Add current user message
	messages = append(messages, GroqMessage{Role: "user", Content: userMessage})

	// Call Groq API
	reply, err := e.groq.Chat(ctx, messages)
	if err != nil {
		return nil, fmt.Errorf("groq API error: %w", err)
	}

	// Determine if handover should be suggested
	suggestHandover := sentiment == "frustrated" || e.shouldSuggestHandover(userMessage, reply)

	return &ChatResponse{
		Reply:           reply,
		Confidence:      0.8, // Placeholder
		SuggestHandover: suggestHandover,
		Sentiment:       sentiment,
	}, nil
}

// Chat sends a chat request to Groq API
func (g *GroqClient) Chat(ctx context.Context, messages []GroqMessage) (string, error) {
	reqBody := GroqRequest{
		Model:       g.model,
		Messages:    messages,
		MaxTokens:   500,
		Temperature: 0.7,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.groq.com/openai/v1/chat/completions", bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+g.apiKey)

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	var groqResp GroqResponse
	if err := json.Unmarshal(body, &groqResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if groqResp.Error != nil {
		return "", fmt.Errorf("groq API error: %s", groqResp.Error.Message)
	}

	if len(groqResp.Choices) == 0 {
		return "", fmt.Errorf("no response choices returned")
	}

	return groqResp.Choices[0].Message.Content, nil
}

// buildSystemPrompt builds the full system prompt with relevant knowledge
func (e *ChatEngine) buildSystemPrompt(relevantKnowledge []KnowledgeItem) string {
	if len(relevantKnowledge) == 0 {
		return e.systemPrompt
	}

	var sb strings.Builder
	sb.WriteString(e.systemPrompt)
	sb.WriteString("\n\nINFORMASI RELEVAN:\n")
	
	for _, item := range relevantKnowledge {
		sb.WriteString(fmt.Sprintf("- %s: %s\n", item.Topic, item.Answer))
	}

	return sb.String()
}

// findRelevantKnowledge finds knowledge items relevant to the user message
func (e *ChatEngine) findRelevantKnowledge(message string) []KnowledgeItem {
	message = strings.ToLower(message)
	var relevant []KnowledgeItem

	for _, item := range e.knowledgeBase {
		for _, keyword := range item.Keywords {
			if strings.Contains(message, strings.ToLower(keyword)) {
				relevant = append(relevant, item)
				break
			}
		}
	}

	// Limit to 3 most relevant items
	if len(relevant) > 3 {
		relevant = relevant[:3]
	}

	return relevant
}

// analyzeSentiment analyzes the sentiment of a message
func (e *ChatEngine) analyzeSentiment(message string) string {
	message = strings.ToLower(message)
	
	// Frustrated/angry indicators
	frustratedWords := []string{
		"tidak membantu", "payah", "lambat", "bingung", "kesel", "marah",
		"kecewa", "buruk", "jelek", "lama sekali", "susah", "ribet",
		"bodoh", "tolol", "goblok", "bangsat", "anjing", "babi",
	}
	
	for _, word := range frustratedWords {
		if strings.Contains(message, word) {
			return "frustrated"
		}
	}

	// Positive indicators
	positiveWords := []string{
		"terima kasih", "makasih", "bagus", "hebat", "mantap", "keren",
		"membantu", "jelas", "paham", "mengerti", "terbantu", "baik",
	}
	
	for _, word := range positiveWords {
		if strings.Contains(message, word) {
			return "positive"
		}
	}

	return "neutral"
}

// shouldSuggestHandover determines if the bot should suggest human handover
func (e *ChatEngine) shouldSuggestHandover(userMessage, botReply string) bool {
	userMessage = strings.ToLower(userMessage)
	botReply = strings.ToLower(botReply)
	
	// User explicitly asks for human
	humanRequestWords := []string{
		"bicara dengan manusia", "hubungi admin", "chat admin",
		"bicara admin", "mau ke admin", "operator", "cs",
		"customer service", "complaint", "komplain",
	}
	
	for _, word := range humanRequestWords {
		if strings.Contains(userMessage, word) {
			return true
		}
	}

	// Bot admits uncertainty
	uncertaintyPhrases := []string{
		"tidak yakin", "kurang tahu", "sebaiknya hubungi",
		"lebih baik tanya", "konsultasikan", "tim kami",
	}
	
	for _, phrase := range uncertaintyPhrases {
		if strings.Contains(botReply, phrase) {
			return true
		}
	}

	return false
}
