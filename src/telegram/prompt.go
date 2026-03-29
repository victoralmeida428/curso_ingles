package telegram

import (
	"curso/src/database"
	"fmt"
)

// BuildGlobalSystemPrompt constrói a instrução principal da IA com os dados do usuário
func BuildGlobalSystemPrompt(user database.UserData) string {
	basePrompt := fmt.Sprintf(`Você é um professor de %s nativo e experiente. Responda sempre em %s.
	 se o nível não for informado, pergunte qual é o nível do aluno.
	 se o tipo não for informado, pergunte qual é o tipo de vocabulário do aluno.`, user.Lang, user.Lang)

	userContext := fmt.Sprintf(
		"Contexto do Aluno:\n- Idioma: %s\n- Tipo de vocabulário: %s\n- Nível de fluência: %s",
		user.Lang, user.Tipo, user.Nivel,
	)

	return fmt.Sprintf("%s\n\n%s", basePrompt, userContext)
}

// Exemplo de como ficaria a função ou bloco onde o `case Text` está inserido:
// Presumindo que você tenha acesso a `ctx`, `client`, `rawMsg` e `msgInfo`

/*
case Text:
	// 1. Busque os dados do usuário no banco (Simulação)
	// userProfile := db.GetUserProfile(rawMsg.UserID)
	userProfile := UserProfile{
		Name:         "Aluno",
		EnglishLevel: "B1 (Intermediário)",
		Goals:        "Melhorar vocabulário corporativo.",
		Preferences:  "Corrija meus erros gramaticais de forma gentil no final da resposta.",
	}

	// 2. Gere o Prompt de Sistema Global
	systemPrompt := BuildGlobalSystemPrompt(userProfile)

	// 3. Inicialize o slice de mensagens com o System Prompt
	messages := []types.Message{
		{
			Role:    "system",
			Content: systemPrompt,
		},
	}

	// 4. Busque o histórico de mensagens (limitado a 20) no banco de dados
	// history := db.GetRecentMessages(rawMsg.UserID, 20)
	// for _, hMsg := range history {
	// 	messages = append(messages, types.Message{
	// 		Role:    hMsg.Role,    // "user" ou "assistant"
	// 		Content: hMsg.Content,
	// 	})
	// }

	// 5. Anexe a nova mensagem atual do usuário
	messages = append(messages, types.Message{
		Role:    "user",
		Content: msgInfo.Text,
	})

	// 6. Monte o payload para o OpenRouter
	chat := types.ChatCompletionRequest{
		Model:    "google/gemini-2.5-flash-lite",
		Messages: messages,
	}

	// 7. Envie a requisição
	resp, err := client.ChatComplete(ctx, &chat)
	if err != nil {
		return fmt.Errorf("erro ao chamar API do OpenRouter: %w", err)
	}

	// 8. Validação de segurança crucial para evitar "panic: index out of range"
	if len(resp.Choices) == 0 || resp.Choices[0].Message.Content == "" {
		return errors.New("a API retornou uma resposta vazia ou sem choices")
	}

	// 9. Atribua a resposta
	rawMsg.Text = resp.Choices[0].Message.Content

	// (A partir daqui, você enviaria rawMsg.Text de volta para o usuário no Telegram)
*/
