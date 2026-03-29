package openrouter

import (
	"bufio"
	"bytes"
	"context"
	"curso/src/openrouter/types"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type streamChunk struct {
	Choices []struct {
		Delta *types.MessageDelta `json:"delta"`
	} `json:"choices"`
}

// ChatComplete envia uma requisição de chat completion.
func (c *Client) ChatComplete(ctx context.Context, req *types.ChatCompletionRequest) (*types.ChatCompletionResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("api error: %d - %s", resp.StatusCode, string(bodyBytes))
	}

	var chatResp types.ChatCompletionResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &chatResp, nil
}

// ChatCompleteStream envia uma requisição de chat completion e retorna um canal de eventos (SSE).
func (c *Client) ChatCompleteStream(ctx context.Context, req *types.ChatCompletionRequest) (<-chan types.StreamEvent, error) {
	// Garante que a requisição pedirá stream
	req.Stream = true

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}

	// Tratamento de erro na resposta HTTP
	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("api error: %d - %s", resp.StatusCode, string(bodyBytes))
	}

	eventChan := make(chan types.StreamEvent)

	go func() {
		defer close(eventChan)
		defer resp.Body.Close()

		reader := bufio.NewReader(resp.Body)

		for {
			// Verifica cancelamento de contexto
			select {
			case <-ctx.Done():
				eventChan <- types.StreamEvent{Error: ctx.Err()}
				return
			default:
			}

			line, err := reader.ReadBytes('\n')
			if err != nil {
				if err != io.EOF {
					eventChan <- types.StreamEvent{Error: fmt.Errorf("read stream: %w", err)}
				}
				return
			}

			line = bytes.TrimSpace(line)

			// Ignora linhas vazias ou que não comecem com "data: "
			if len(line) == 0 || !bytes.HasPrefix(line, []byte("data: ")) {
				continue
			}

			data := bytes.TrimPrefix(line, []byte("data: "))

			// Sinal de encerramento padrão da API
			if string(data) == "[DONE]" {
				return
			}

			var chunk streamChunk
			if err := json.Unmarshal(data, &chunk); err != nil {
				eventChan <- types.StreamEvent{Error: fmt.Errorf("unmarshal chunk: %w", err)}
				continue
			}

			// Navega com segurança pela estrutura para evitar nil pointer dereference ou index out of range
			if len(chunk.Choices) > 0 && chunk.Choices[0].Delta != nil {
				delta := chunk.Choices[0].Delta

				// Monta o evento mapeando os campos para a sua estrutura StreamEvent
				event := types.StreamEvent{
					Delta: delta,
				}

				if delta.Content != "" {
					event.Transcript = delta.Content
				}

				// Se houver áudio no delta, expõe diretamente na raiz do StreamEvent para facilitar o consumo
				if delta.Audio != nil {
					event.Audio = delta.Audio
					if delta.Audio.Transcript != "" {
						event.Transcript = delta.Audio.Transcript
					}
				}

				eventChan <- event
			}
		}
	}()

	return eventChan, nil
}
