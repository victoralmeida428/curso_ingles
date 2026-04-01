package telegram

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"os/exec"

	"github.com/go-telegram/bot"
)


// fetchAndConvertAudio baixa o OGG do Telegram e converte para WAV 100% em memória.
func fetchAndConvertAudio(ctx context.Context, b *bot.Bot, fileID string) (string, error) {
	file, err := b.GetFile(ctx, &bot.GetFileParams{
		FileID: fileID,
	})
	if err != nil {
		return "", fmt.Errorf("erro ao obter file_path do telegram: %w", err)
	}

	fileURL := b.FileDownloadLink(file)

	resp, err := http.Get(fileURL)
	if err != nil {
		return "", fmt.Errorf("erro na requisição de download: %w", err)
	}
	defer func(){
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("telegram retornou status code: %d", resp.StatusCode)
	}

	oggBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("erro ao ler bytes para a memória: %w", err)
	}

	// Buffers para capturar o áudio gerado e os logs de erro do FFmpeg
	var mp3Buffer bytes.Buffer
	var errBuffer bytes.Buffer

	// MUDANÇA: Convertendo para MP3 com bitrate de voz (64k) para evitar payloads gigantes
	cmd := exec.CommandContext(ctx, "ffmpeg", "-i", "pipe:0", "-b:a", "64k", "-f", "mp3", "pipe:1")

	cmd.Stdin = bytes.NewReader(oggBytes)
	cmd.Stdout = &mp3Buffer
	cmd.Stderr = &errBuffer // Captura o log de erro para podermos debugar se falhar

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("erro ao converter com ffmpeg: %v, log: %s", err, errBuffer.String())
	}

	// Retorna os bytes já comprimidos em MP3 como Base64
	return base64.StdEncoding.EncodeToString(mp3Buffer.Bytes()), nil
}