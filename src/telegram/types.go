package telegram

// RawTelegramMessage carrega os dados trafegados internamente.
type RawTelegramMessage struct {
	ChatID             int64
	Text               string
	Voice              *VoicePayload
	Audio              *AudioPayload
	ResponseAudioBytes []byte // NOVO: Guarda o áudio gerado em RAM, sem tocar no disco
}

type VoicePayload struct {
	FileID   string
	Duration int
	MimeType string
	FileSize int64
}

// ExtractedMessage e MessageType (do seu classifier)
type MessageType int

const (
	Unknown MessageType = iota
	Text
	Audio
)

type ExtractedMessage struct {
	Type   MessageType
	Text   string
	FileID string
}
