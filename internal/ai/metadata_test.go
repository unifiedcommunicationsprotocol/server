package ai

import (
	"testing"
)

func TestProcessMessage(t *testing.T) {
	mp := New()

	body := "This is an urgent meeting reminder for tomorrow at 10 AM. Please confirm your attendance."
	metadata := mp.ProcessMessage(body)

	if metadata == nil {
		t.Error("metadata is nil")
	}

	if metadata.Summary == "" {
		t.Error("summary is empty")
	}

	if len(metadata.Categories) == 0 {
		t.Error("categories is empty")
	}

	if metadata.Sentiment == "" {
		t.Error("sentiment is empty")
	}

	if len(metadata.Keywords) == 0 {
		t.Error("keywords is empty")
	}
}

func TestCategorization(t *testing.T) {
	mp := New()

	// Test meeting categorization
	metadata := mp.ProcessMessage("Let's schedule a meeting for next week")
	if !contains(metadata.Categories, "meeting") {
		t.Error("should categorize as meeting")
	}

	// Test urgent categorization
	metadata = mp.ProcessMessage("URGENT: This needs to be done ASAP")
	if !contains(metadata.Categories, "urgent") {
		t.Error("should categorize as urgent")
	}

	// Test question categorization
	metadata = mp.ProcessMessage("Can you help me with this? I have a question about the process.")
	if !contains(metadata.Categories, "question") {
		t.Error("should categorize as question")
	}
}

func TestSentimentAnalysis(t *testing.T) {
	mp := New()

	// Test positive sentiment
	metadata := mp.ProcessMessage("This is great! I love this solution, it's excellent!")
	if metadata.Sentiment != "positive" {
		t.Errorf("sentiment: got %q, want %q", metadata.Sentiment, "positive")
	}

	// Test urgent sentiment
	metadata = mp.ProcessMessage("This is urgent and needs immediate attention!")
	if metadata.Sentiment != "urgent" {
		t.Errorf("sentiment: got %q, want %q", metadata.Sentiment, "urgent")
	}

	// Test negative sentiment
	metadata = mp.ProcessMessage("This is awful and terrible, I hate it")
	if metadata.Sentiment != "negative" {
		t.Errorf("sentiment: got %q, want %q", metadata.Sentiment, "negative")
	}
}

func TestSpamDetection(t *testing.T) {
	mp := New()

	// Test spam
	metadata := mp.ProcessMessage("Click here now for limited time offer! Verify your account immediately!")
	if !metadata.IsSpam {
		t.Error("should detect as spam")
	}

	// Test legitimate
	metadata = mp.ProcessMessage("Hi, I wanted to follow up on our conversation from yesterday.")
	if metadata.IsSpam {
		t.Error("should not detect as spam")
	}
}

func TestPriorityAssignment(t *testing.T) {
	mp := New()

	// Urgent gets high priority
	metadata := mp.ProcessMessage("URGENT: Critical issue that needs immediate attention")
	if metadata.Priority != 5 {
		t.Errorf("priority: got %d, want 5 for urgent", metadata.Priority)
	}
}

func TestEmbeddingStore(t *testing.T) {
	store := NewEmbeddingStore()

	// Store embedding
	embedding := []float32{0.1, 0.2, 0.3, 0.4, 0.5}
	store.Store("msg_123", embedding)

	// Retrieve embedding
	retrieved := store.Get("msg_123")
	if len(retrieved) != 5 {
		t.Errorf("embedding length: got %d, want 5", len(retrieved))
	}
}

func TestSimilaritySearch(t *testing.T) {
	store := NewEmbeddingStore()

	// Store some embeddings
	store.Store("msg_1", []float32{1.0, 0.0, 0.0, 0.0, 0.0})
	store.Store("msg_2", []float32{0.9, 0.1, 0.0, 0.0, 0.0})
	store.Store("msg_3", []float32{0.0, 0.0, 1.0, 0.0, 0.0})

	// Search for similar
	query := []float32{1.0, 0.0, 0.0, 0.0, 0.0}
	results := store.SimilaritySearch(query, 2)

	if len(results) != 2 {
		t.Errorf("result count: got %d, want 2", len(results))
	}

	// First result should be msg_1 (identical)
	if results[0] != "msg_1" {
		t.Errorf("top result: got %q, want msg_1", results[0])
	}
}

func contains(arr []string, val string) bool {
	for _, v := range arr {
		if v == val {
			return true
		}
	}
	return false
}
