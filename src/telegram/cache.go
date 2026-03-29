package telegram

import (
	"sync"

	"curso/src/openrouter/types"
)

// HistoryCache gerencia o histórico de conversas em memória RAM de forma segura para concorrência.
type HistoryCache struct {
	mu       sync.RWMutex
	limit    int
	messages map[int64][]types.Message
}

// NewHistoryCache inicializa o cache definindo o tamanho máximo da fila (ex: 30 mensagens).
func NewHistoryCache(limit int) *HistoryCache {
	return &HistoryCache{
		limit:    limit,
		messages: make(map[int64][]types.Message),
	}
}

// AddMessage insere uma nova mensagem na fila. Se a fila encher, a mais antiga é descartada (FIFO).
func (c *HistoryCache) AddMessage(chatID int64, role string, content string) {
	c.mu.Lock()         // Tranca a memória para escrita exclusiva
	defer c.mu.Unlock() // Destranca ao final

	// Não salvamos mensagens vazias no contexto
	if content == "" {
		return
	}

	msg := types.Message{
		Role:    role,
		Content: content,
	}

	history := c.messages[chatID]

	// Lógica da Fila (FIFO): Se atingiu o limite, cortamos o primeiro elemento (índice 0)
	if len(history) >= c.limit {
		history = history[1:]
	}

	// Adiciona a nova mensagem no final da fila
	history = append(history, msg)
	c.messages[chatID] = history
}

// GetMessages retorna uma CÓPIA do histórico atual do usuário.
func (c *HistoryCache) GetMessages(chatID int64) []types.Message {
	c.mu.RLock()         // Tranca apenas para leitura (permite múltiplas leituras simultâneas)
	defer c.mu.RUnlock() // Destranca ao final

	history, exists := c.messages[chatID]
	if !exists {
		return nil
	}

	copyHistory := make([]types.Message, len(history))
	copy(copyHistory, history)

	return copyHistory
}