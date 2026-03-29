// --------------------------------------------------------------------
// Tipos para Chat Completions
// --------------------------------------------------------------------
package types

type (
	// Message representa uma mensagem na conversa.
	Message struct {
		Role    string      `json:"role"`
		Content interface{} `json:"content"`
		Name    *string     `json:"name,omitempty"`
	}

	// ContentPart é usado para mensagens multimodais (imagem, áudio).
	ContentPart struct {
		Type       string      `json:"type"`
		Text       *string     `json:"text,omitempty"`
		ImageURL   *ImageURL   `json:"image_url,omitempty"`
		InputAudio *InputAudio `json:"input_audio,omitempty"`
	}

	// ImageURL representa uma imagem.
	ImageURL struct {
		URL    string `json:"url"`
		Detail string `json:"detail,omitempty"`
	}

	// InputAudio representa um áudio de entrada.
	InputAudio struct {
		Data   string `json:"data"`
		Format string `json:"format,omitempty"`
	}

	// ChatCompletionRequest é o corpo da requisição para /chat/completions.
	ChatCompletionRequest struct {
		Model       string       `json:"model"`
		Messages    []Message    `json:"messages"`
		Stream      bool         `json:"stream,omitempty"`
		MaxTokens   *int         `json:"max_tokens,omitempty"`
		Temperature *float32     `json:"temperature,omitempty"`
		TopP        *float32     `json:"top_p,omitempty"`
		Modalities  []string     `json:"modalities,omitempty"` // "text", "audio"
		Audio       *AudioConfig `json:"audio,omitempty"`
		// outros campos podem ser adicionados conforme necessidade
	}

	// AudioConfig configura a resposta em áudio (para modelos que suportam).
	AudioConfig struct {
		Voice  string `json:"voice"`  // ex: "alloy", "echo", "fable", etc.
		Format string `json:"format"` // "mp3" ou "wav"
	}

	// ChatCompletionResponse é a resposta da API.
	ChatCompletionResponse struct {
		ID      string `json:"id"`
		Choices []struct {
			Message struct {
				Role    string `json:"role"`
				Content string `json:"content"`
				// Alguns modelos podem retornar áudio no campo 'audio'
				Audio *AudioResponse `json:"audio,omitempty"`
			} `json:"message"`
			FinishReason string `json:"finish_reason"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
		} `json:"usage"`
	}

	// AudioResponse contém o áudio gerado pelo modelo.
	AudioResponse struct {
		ID         string `json:"id"`
		Data       string `json:"data"` // base64 do áudio
		Expires    int64  `json:"expires"`
		Transcript string `json:"transcript"`
	}

	StreamEvent struct {
		Delta      *MessageDelta
		Audio      *AudioResponse // contém Data (base64)
		Transcript string
		Error      error
	}

	MessageDelta struct {
		Role    string         `json:"role,omitempty"`
		Content string         `json:"content,omitempty"`
		Audio   *AudioResponse `json:"audio,omitempty"`
	}
)
