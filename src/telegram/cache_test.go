package telegram

import (
	"testing"
)

func TestHistoryCache_AddAndGetMessages(t *testing.T) {
	cache := NewHistoryCache(3)
	chatID := int64(12345)

	cache.AddMessage(chatID, "user", "Hello")
	msgs := cache.GetMessages(chatID)
	if len(msgs) != 1 || msgs[0].Content != "Hello" {
		t.Errorf("Expected 1 message 'Hello', got %v", msgs)
	}

	cache.AddMessage(chatID, "assistant", "Hi there")
	cache.AddMessage(chatID, "user", "How are you?")
	msgs = cache.GetMessages(chatID)
	if len(msgs) != 3 {
		t.Errorf("Expected 3 messages, got %d", len(msgs))
	}

	// Should drop the first one ("user: Hello")
	cache.AddMessage(chatID, "assistant", "I am fine")
	msgs = cache.GetMessages(chatID)
	if len(msgs) != 3 {
		t.Errorf("Expected 3 messages after limit, got %d", len(msgs))
	}
	if msgs[0].Content != "Hi there" {
		t.Errorf("Expected first message to be 'Hi there', got %s", msgs[0].Content)
	}
}

func TestHistoryCache_EmptyContent(t *testing.T) {
	cache := NewHistoryCache(3)
	chatID := int64(12345)

	cache.AddMessage(chatID, "user", "")
	msgs := cache.GetMessages(chatID)
	if len(msgs) != 0 {
		t.Errorf("Expected empty messages not to be added, but got %v", msgs)
	}
}
