package utils

import (
	"curso/src/openrouter/types"
	"encoding/base64"
	"encoding/binary"
)

func AudioContent(data []byte, format string) types.ContentPart {
	return types.ContentPart{
		Type: "input_audio",
		InputAudio: &types.InputAudio{
			Data:   base64.StdEncoding.EncodeToString(data),
			Format: format,
		},
	}
}

// TextContent cria um ContentPart de texto.
func TextContent(text string) types.ContentPart {
	return types.ContentPart{
		Type: "text",
		Text: &text,
	}
}

// addWAVHeader adiciona um cabeçalho RIFF/WAVE a dados PCM puros.
func AddWAVHeader(rawPCM []byte, sampleRate uint32) []byte {
	dataLen := uint32(len(rawPCM))
	header := make([]byte, 44)

	// Chunk RIFF
	copy(header[0:4], []byte("RIFF"))
	binary.LittleEndian.PutUint32(header[4:8], dataLen+36)
	copy(header[8:12], []byte("WAVE"))

	// Chunk fmt (formato)
	copy(header[12:16], []byte("fmt "))
	binary.LittleEndian.PutUint32(header[16:20], 16) // Tamanho do subchunk (16 para PCM)
	binary.LittleEndian.PutUint16(header[20:22], 1)  // Formato do Áudio (1 = PCM)
	binary.LittleEndian.PutUint16(header[22:24], 1)  // Canais (1 = Mono)
	binary.LittleEndian.PutUint32(header[24:28], sampleRate) // Taxa de amostragem
	binary.LittleEndian.PutUint32(header[28:32], sampleRate*2) // Byte Rate
	binary.LittleEndian.PutUint16(header[32:34], 2)  // Block Align
	binary.LittleEndian.PutUint16(header[34:36], 16) // Bits por amostra (16 bits)

	// Chunk data (dados)
	copy(header[36:40], []byte("data"))
	binary.LittleEndian.PutUint32(header[40:44], dataLen)

	// Retorna o cabeçalho colado com o áudio original
	return append(header, rawPCM...)
}