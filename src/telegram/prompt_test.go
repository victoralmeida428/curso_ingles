package telegram

import (
	"curso/src/database"
	"strings"
	"testing"
)

func TestBuildGlobalSystemPrompt(t *testing.T) {
	user := database.UserData{
		Lang:  "inglês",
		Tipo:  "formal",
		Nivel: "iniciante",
	}

	prompt := BuildGlobalSystemPrompt(user)

	if !strings.Contains(prompt, "inglês") {
		t.Errorf("Expected prompt to contain Lang 'inglês', got %s", prompt)
	}
	if !strings.Contains(prompt, "formal") {
		t.Errorf("Expected prompt to contain Tipo 'formal', got %s", prompt)
	}
	if !strings.Contains(prompt, "iniciante") {
		t.Errorf("Expected prompt to contain Nivel 'iniciante', got %s", prompt)
	}
}
