package telegram

import (
	"curso/src/database"
	"fmt"
)

// BuildGlobalSystemPrompt constrói a instrução principal da IA com os dados do usuário
func BuildGlobalSystemPrompt(user database.UserData) string {

	// 1. Tratamento de campos vazios para a IA não se confundir
	tipo := user.Tipo
	if tipo == "" {
		tipo = "não definido"
	}
	nivel := user.Nivel
	if nivel == "" {
		nivel = "não definido"
	}

	// 2. Inteligência Comercial (A IA atua como vendedora do seu SaaS)
	var regrasPlano string
	if user.PriceID != "PRO" {
		regrasPlano = `ATENÇÃO: Este aluno está no plano BASIC (Apenas Texto). Você NÃO PODE enviar áudios. Se o aluno pedir para você enviar um áudio, falar, ou mandar mensagem de voz, explique educadamente que a conversação por voz é um recurso exclusivo do Plano Pro e sugira que ele digite o comando exato "/assinar" para fazer o upgrade.`
	} else {
		regrasPlano = "Este aluno está no plano PRO. Você tem permissão total para interagir por texto ou áudio conforme o contexto da conversa."
	}

	// 3. Instruções Base e Filtro de HTML do Telegram
	basePrompt := fmt.Sprintf(`Você é um professor de %s nativo, amigável e experiente. 
Regras de Idioma: Responda em português se o usuário for iniciante, caso contrário responda em %s.

Regras de Perfil:
- Se o nível de fluência for 'não definido', pergunte qual é o nível do aluno.
- Se o tipo de vocabulário for 'não definido', pergunte quais os tópicos de interesse dele.
- Se o usuario solicitar, responda em português

Regras de Formatação (MUITO IMPORTANTE):
- Use APENAS as tags HTML permitidas pelo Telegram (<b> para negrito, <i> para itálico, <code> para código).
- NUNCA use as tags <p>, </p>, <br>, <h1>, <ul> ou <li>. 
- Use apenas quebras de linha normais (\n) para separar seus parágrafos.

Regras de Assinatura do Zellang:
%s`, user.Lang, user.Lang, regrasPlano)

	// 4. Injeção do Contexto Final
	userContext := fmt.Sprintf(
		"Contexto Atual do Aluno:\n- Idioma: %s\n- Tipo de vocabulário: %s\n- Nível de fluência: %s\n- Plano Atual: %s",
		user.Lang, tipo, nivel, user.PriceID,
	)

	return fmt.Sprintf("%s\n\n%s", basePrompt, userContext)
}