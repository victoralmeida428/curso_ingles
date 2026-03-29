package openrouter

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// AccountInfo retorna informações da chave, incluindo créditos.
type AccountInfo struct {
	Credits   float64 `json:"total_credits"`
	Usage     float64 `json:"total_usage"`
	Remaining float64 `json:"remaining_credits"`
}

// GetCredits consulta os créditos restantes da chave.
func (c *Client) GetCredits(ctx context.Context) (*AccountInfo, error) {
	httpReq, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/credits", nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("api error: %d - %s", resp.StatusCode, string(bodyBytes))
	}

	infoResponse := make(map[string]AccountInfo)
	if err := json.NewDecoder(resp.Body).Decode(&infoResponse); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	info, exists := infoResponse["data"]
	if !exists {
		return nil, fmt.Errorf("data not found in response")
	}
	info.Remaining = info.Credits - info.Usage
	return &info, nil
}
