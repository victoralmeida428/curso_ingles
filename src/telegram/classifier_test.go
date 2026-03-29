package telegram

import (
	"testing"
)

func TestClassifyMessage_Text(t *testing.T) {
	msg := &RawTelegramMessage{Text: "Test message"}
	extracted, err := ClassifyMessage(msg)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if extracted.Type != Text || extracted.Text != "Test message" {
		t.Errorf("Expected Text message, got %v", extracted)
	}
}

func TestClassifyMessage_Voice(t *testing.T) {
	msg := &RawTelegramMessage{Voice: &VoicePayload{FileID: "voice123"}}
	extracted, err := ClassifyMessage(msg)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if extracted.Type != Audio || extracted.FileID != "voice123" {
		t.Errorf("Expected Voice message with ID voice123, got %v", extracted)
	}
}

func TestClassifyMessage_Unknown(t *testing.T) {
	msg := &RawTelegramMessage{}
	extracted, err := ClassifyMessage(msg)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if extracted.Type != Unknown {
		t.Errorf("Expected Unknown message type, got %v", extracted)
	}
}

func TestClassifyMessage_Nil(t *testing.T) {
	_, err := ClassifyMessage(nil)
	if err == nil {
		t.Errorf("Expected error for nil message, got nil")
	}
}
