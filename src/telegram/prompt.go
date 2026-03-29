package telegram

import (
	"curso/src/database"
	"fmt"
)

// BuildGlobalSystemPrompt constrói a instrução principal da IA com os dados do usuário
func BuildGlobalSystemPrompt(user database.UserData) string {

	basePrompt := fmt.Sprintf(`Você é um professor de %s e experiente. Responda em português se o usuario for iniciante, caso contrario responda em %s.
	 se o nível não for informado, pergunte qual é o nível do aluno.
	 se o tipo não for informado, pergunte qual é o tipo de vocabulário do aluno.`, user.Lang, user.Lang)

	userContext := fmt.Sprintf(
		"Contexto do Aluno:\n- Idioma: %s\n- Tipo de vocabulário: %s\n- Nível de fluência: %s",
		user.Lang, user.Tipo, user.Nivel,
	)

	return fmt.Sprintf("%s\n\n%s", basePrompt, userContext)
}
