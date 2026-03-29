package telegram

import (
	"errors"
)


// AudioPayload representa um arquivo de áudio enviado como anexo.
type AudioPayload struct {
	FileID string
}

// ClassifyMessage recebe a mensagem bruta do Telegram, analisa seus campos
// e retorna uma estrutura padronizada identificando se é texto ou áudio.
func ClassifyMessage(msg *RawTelegramMessage) (*ExtractedMessage, error) {
	if msg == nil {
		return nil, errors.New("mensagem nula recebida")
	}

	if msg.Text != "" {
		return &ExtractedMessage{
			Type: Text,
			Text: msg.Text,
		}, nil
	}

	if msg.Voice != nil && msg.Voice.FileID != "" {
		return &ExtractedMessage{
			Type:   Audio,
			FileID: msg.Voice.FileID,
		}, nil
	}

	if msg.Audio != nil && msg.Audio.FileID != "" {
		return &ExtractedMessage{
			Type:   Audio,
			FileID: msg.Audio.FileID,
		}, nil
	}

	// 4. Retorna Unknown para qualquer outra coisa (fotos, vídeos, stickers, documentos).
	return &ExtractedMessage{
		Type: Unknown,
	}, nil
}