// Package ai provides AI-driven metadata generation for UCP messages.
package ai

import (
	"strings"
	"sync"
)

// MetadataProcessor generates AI metadata for messages.
type MetadataProcessor struct {
	mu sync.RWMutex
}

// MessageMetadata contains AI-generated metadata for a message.
type MessageMetadata struct {
	Summary      string   `json:"summary,omitempty"`
	Categories   []string `json:"categories,omitempty"`
	Sentiment    string   `json:"sentiment,omitempty"`
	Keywords     []string `json:"keywords,omitempty"`
	Priority     int      `json:"priority,omitempty"` // 1-5, 5 highest
	IsSpam       bool     `json:"is_spam,omitempty"`
	EmbeddingRef string   `json:"embedding_ref,omitempty"` // Reference to stored embedding
}

// New creates a new metadata processor.
func New() *MetadataProcessor {
	return &MetadataProcessor{}
}

// ProcessMessage generates metadata for a message body.
// Implementation is simplified; real system would use ML models.
func (mp *MetadataProcessor) ProcessMessage(body string) *MessageMetadata {
	mp.mu.Lock()
	defer mp.mu.Unlock()

	metadata := &MessageMetadata{
		Priority: 3, // Default medium priority
	}

	// Extract summary (first 100 chars)
	if len(body) > 100 {
		metadata.Summary = strings.TrimSpace(body[:100]) + "..."
	} else {
		metadata.Summary = body
	}

	// Categorize based on keywords
	metadata.Categories = categorizeMessage(body)

	// Analyze sentiment
	metadata.Sentiment = analyzeSentiment(body)

	// Extract keywords
	metadata.Keywords = extractKeywords(body)

	// Detect spam patterns
	metadata.IsSpam = detectSpam(body)

	// Adjust priority based on sentiment
	if metadata.Sentiment == "urgent" {
		metadata.Priority = 5
	} else if metadata.Sentiment == "low" {
		metadata.Priority = 1
	}

	return metadata
}

// categorizeMessage identifies message categories.
func categorizeMessage(body string) []string {
	var categories []string

	lower := strings.ToLower(body)

	if strings.Contains(lower, "meeting") || strings.Contains(lower, "call") {
		categories = append(categories, "meeting")
	}

	if strings.Contains(lower, "invoice") || strings.Contains(lower, "payment") || strings.Contains(lower, "receipt") {
		categories = append(categories, "invoice")
	}

	if strings.Contains(lower, "urgent") || strings.Contains(lower, "asap") || strings.Contains(lower, "important") {
		categories = append(categories, "urgent")
	}

	if strings.Contains(lower, "followup") || strings.Contains(lower, "reminder") {
		categories = append(categories, "followup")
	}

	if strings.Contains(lower, "question") || strings.Contains(lower, "help") || strings.Contains(lower, "?") {
		categories = append(categories, "question")
	}

	if len(categories) == 0 {
		categories = append(categories, "general")
	}

	return categories
}

// analyzeSentiment determines message sentiment.
func analyzeSentiment(body string) string {
	lower := strings.ToLower(body)

	urgentWords := []string{"urgent", "asap", "critical", "emergency", "immediately"}
	for _, word := range urgentWords {
		if strings.Contains(lower, word) {
			return "urgent"
		}
	}

	positiveWords := []string{"great", "thanks", "excellent", "happy", "love", "perfect"}
	positiveCount := 0
	for _, word := range positiveWords {
		if strings.Contains(lower, word) {
			positiveCount++
		}
	}

	negativeWords := []string{"bad", "terrible", "awful", "hate", "disgusted", "angry"}
	negativeCount := 0
	for _, word := range negativeWords {
		if strings.Contains(lower, word) {
			negativeCount++
		}
	}

	if negativeCount > positiveCount {
		return "negative"
	} else if positiveCount > negativeCount {
		return "positive"
	}

	return "neutral"
}

// extractKeywords extracts important keywords from message.
func extractKeywords(body string) []string {
	// Simple keyword extraction: split by spaces and filter
	words := strings.Fields(body)
	var keywords []string

	// Common stop words to exclude
	stopWords := map[string]bool{
		"the": true, "a": true, "an": true, "and": true, "or": true,
		"but": true, "in": true, "on": true, "at": true, "to": true,
		"for": true, "of": true, "with": true, "by": true, "from": true,
		"is": true, "are": true, "was": true, "were": true,
	}

	for _, word := range words {
		clean := strings.ToLower(strings.Trim(word, ",.!?;:"))
		if len(clean) > 4 && !stopWords[clean] {
			found := false
			for _, k := range keywords {
				if k == clean {
					found = true
					break
				}
			}
			if !found && len(keywords) < 5 {
				keywords = append(keywords, clean)
			}
		}
	}

	return keywords
}

// detectSpam identifies likely spam messages.
func detectSpam(body string) bool {
	lower := strings.ToLower(body)

	// Spam indicators
	spamPatterns := []string{
		"click here", "limited time", "act now", "verify account",
		"confirm password", "update payment", "claim reward",
		"nigerian prince", "inheritance", "bitcoin", "crypto",
	}

	for _, pattern := range spamPatterns {
		if strings.Contains(lower, pattern) {
			return true
		}
	}

	// Check for excessive URLs
	urlCount := strings.Count(body, "http")
	if urlCount > 3 {
		return true
	}

	// Check for excessive capitalization
	upper := 0
	lower_count := 0
	for _, ch := range body {
		if ch >= 'A' && ch <= 'Z' {
			upper++
		} else if ch >= 'a' && ch <= 'z' {
			lower_count++
		}
	}

	if lower_count > 0 && upper > lower_count*2 {
		return true
	}

	return false
}

// EmbeddingStore stores message embeddings (simplified mock).
type EmbeddingStore struct {
	mu        sync.RWMutex
	embeddings map[string][]float32
}

// NewEmbeddingStore creates a new embedding store.
func NewEmbeddingStore() *EmbeddingStore {
	return &EmbeddingStore{
		embeddings: make(map[string][]float32),
	}
}

// Store saves an embedding for a message.
func (es *EmbeddingStore) Store(messageID string, embedding []float32) {
	es.mu.Lock()
	defer es.mu.Unlock()
	es.embeddings[messageID] = embedding
}

// Get retrieves an embedding by message ID.
func (es *EmbeddingStore) Get(messageID string) []float32 {
	es.mu.RLock()
	defer es.mu.RUnlock()
	return es.embeddings[messageID]
}

// SimilaritySearch finds messages similar to the given embedding.
func (es *EmbeddingStore) SimilaritySearch(embedding []float32, topK int) []string {
	es.mu.RLock()
	defer es.mu.RUnlock()

	type scoreResult struct {
		messageID string
		score     float32
	}

	var results []scoreResult

	for msgID, stored := range es.embeddings {
		score := cosineSimilarity(embedding, stored)
		results = append(results, scoreResult{msgID, score})
	}

	// Sort by score (descending)
	for i := 0; i < len(results); i++ {
		for j := i + 1; j < len(results); j++ {
			if results[j].score > results[i].score {
				results[i], results[j] = results[j], results[i]
			}
		}
	}

	// Return top K
	if topK > len(results) {
		topK = len(results)
	}

	var ids []string
	for i := 0; i < topK; i++ {
		ids = append(ids, results[i].messageID)
	}

	return ids
}

// cosineSimilarity computes cosine similarity between two vectors.
func cosineSimilarity(a, b []float32) float32 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}

	var dotProduct, normA, normB float32
	for i := 0; i < len(a); i++ {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dotProduct / (sqrt(normA) * sqrt(normB))
}

// sqrt is a simple square root approximation.
func sqrt(x float32) float32 {
	if x < 0 {
		return 0
	}
	if x == 0 {
		return 0
	}

	z := x / 2
	for i := 0; i < 10; i++ {
		z = (z + x/z) / 2
	}
	return z
}
