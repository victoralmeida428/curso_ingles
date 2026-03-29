# English Teacher AI Telegram Bot 🤖🇬🇧

Este é um bot do Telegram projetado para funcionar como um professor e avaliador de inglês pessoal, utilizando inteligência artificial através da API do OpenRouter. O bot não apenas conversa em texto, mas também tem suporte avançado e rápido para mensagens de voz e envio de áudios responsivos em tempo real.

## 🌟 Funcionalidades

- **Conversa Natural em Inglês**: Pratique seu inglês como se estivesse conversando com um professor real.
- **Identificação de Habilidades**: Personalize o bot definindo o seu idioma nativo, o tipo de vocabulário focado (formal, informal, técnico) e o nível atual.
- **Teste de Nivelamento CEFR (`/nivelar`)**: O bot atua como um avaliador, aplicando um teste com apenas 4 perguntas sobre (Vocabulário, Escrita, Pronúncia e Leitura) para identificar e classificar o seu nível dentro do Quadro Europeu Comum de Referência (CEFR: A1 ao C2).
- **Suporte Multimodal de Voz Remota**: Envie áudios que são processados e entendidos pela IA. O próprio bot é capaz de responder usando também áudio humano-like.
- **Manipulação na Memória RAM**: Processamentos de conversão de áudio (`MP3 -> OGG Opus`) feitos com o FFmpeg sem a necessidade de gravar arquivos intermediários, entregando uma resposta rápida para o Telegram.

## 🛠️ Tecnologias Utilizadas

- **Linguagem**: [Go (Golang)](https://go.dev/)
- **Biblioteca do Telegram**: [go-telegram/bot](https://github.com/go-telegram/bot)
- **Banco de Dados**: [SQLite](https://sqlite.org/) utilizando `mattn/go-sqlite3` para armazenar perfil/estado de usuários de forma local.
- **IA/LLM**: [OpenRouter API](https://openrouter.ai/). 
  - *google/gemini-2.5-flash-lite* para classificação rápida e processamento de texto.
  - *openai/gpt-4o-audio-preview* para suporte multimodal robusto gerando voz/áudio natural.
- **Áudio**: [FFmpeg](https://ffmpeg.org/) usado para conversão de codecs e formato Opus requisitado pelo Telegram.

## 🚀 Como Executar Localmente

### Pré-requisitos
1. **Go 1.20+** instalado.
2. **FFmpeg** instalado na máquina e acessível no seu `PATH` global (necessário para o processo de voz em memória RAM).
3. Ter um Token do Bot válido usando o [BotFather](https://t.me/BotFather) no Telegram.
4. Ter um Token da API do OpenRouter.

### Configuração
Crie um arquivo `.env` na raiz do projeto com as suas chaves baseando-se no que for necessário (exemplo):
```env
TELEGRAM_BOT_TOKEN=seu_bot_token_aqui
OPENROUTER_API_KEY=sua_or_api_key_aqui
```

### Rodando a aplicação
Construa o projeto e inicialize o servidor:
```bash
go mod tidy
go build -o english_bot ./src/...
./english_bot
```
O console exibirá: `🤖 Bot do Telegram em execução! Pressione Ctrl+C para parar.`

### Rodando os Testes
Para garantir a sanidade da solução e executar os testes unitários do cache limitador, do classificador e das construções do prompt de IA, execute:
```bash
go test ./src/...
```

## 💬 Comandos Disponíveis (Telegram)

Você pode acessar os seguintes comandos interagindo com o seu bot no Telegram:
- `/start` - Inicia o bot e exibe as instruções essenciais de customização.
- `/lang {idioma}` - Define qual o seu idioma nativo.
- `/tipo {formal/informal/tecnico}` - Define qual o modelo/foco do seu vocabulário ensinado.
- `/nivel {seu_nivel}` - Define manualmente o seu nível de prática para o prompt do professor ajustado.
- `/perfil` - Mostra os detalhes do seu perfil atual configurado.
- `/nivelar` - Aciona a IA como modo revisor/avaliador para um teste guiado de nivelamento inglês com análise de fala, escuta e leitura, indicando o seu grau (A1... C2).
# curso_ingles
