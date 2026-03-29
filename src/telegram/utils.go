package telegram

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"

	"github.com/go-telegram/bot"
)

// downloadTelegramAudio baixa um arquivo do Telegram usando seu FileID e retorna o caminho do arquivo temporário.
func downloadTelegramAudio(ctx context.Context, b *bot.Bot, fileID string) (string, error) {
	// 1. Obtém as informações do arquivo na API do Telegram (isso nos dá o FilePath real)
	file, err := b.GetFile(ctx, &bot.GetFileParams{
		FileID: fileID,
	})
	if err != nil {
		return "", fmt.Errorf("erro ao obter informações do arquivo: %w", err)
	}

	// 2. Obtém o link de download direto com a autenticação do seu Bot
	fileURL := b.FileDownloadLink(file)

	// 3. Faz a requisição HTTP GET para baixar o conteúdo
	resp, err := http.Get(fileURL)
	if err != nil {
		return "", fmt.Errorf("erro ao baixar o arquivo via HTTP: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("falha no download, status HTTP inválido: %d", resp.StatusCode)
	}

	// 6. Retorna o caminho absoluto do arquivo recém-criado
	bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("erro ao ler o corpo da resposta: %w", err)
	}
	return base64.StdEncoding.EncodeToString(bytes), nil
}
