package openrouter

import (
	"context"
	"curso/src/openrouter/types"
	"net/http"
	"time"
)

type (
	IClient interface {
		SetBaseURL(url string)
		SetHTTPClient(client *http.Client)
		ChatComplete(ctx context.Context, req *types.ChatCompletionRequest) (*types.ChatCompletionResponse, error)
		ChatCompleteStream(ctx context.Context, req *types.ChatCompletionRequest) (<-chan types.StreamEvent, error)
		GetCredits(ctx context.Context) (*AccountInfo, error)
	}

	Client struct {
		apiKey     string
		baseURL    string
		httpClient *http.Client
	}
)

func NewClient(apiKey string) IClient {
	return &Client{
		apiKey:     apiKey,
		baseURL:    "https://openrouter.ai/api/v1",
		httpClient: &http.Client{Timeout: 180 * time.Second},
	}
}

// SetBaseURL permite trocar a URL base (útil para testes).
func (c *Client) SetBaseURL(url string) {
	c.baseURL = url
}

// SetHTTPClient permite injetar um cliente HTTP personalizado.
func (c *Client) SetHTTPClient(client *http.Client) {
	c.httpClient = client
}
