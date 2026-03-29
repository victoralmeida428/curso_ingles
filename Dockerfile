# ==========================================
# Estágio 1: Builder (Compilação)
# ==========================================
FROM golang:1.25.4-bookworm AS builder

# Define o diretório de trabalho dentro do container
WORKDIR /app

# Copia os arquivos de dependência e baixa os pacotes
COPY go.mod go.sum ./
RUN go mod download

# Copia o restante do código fonte
COPY . .

# Compila o binário com CGO ativado (obrigatório para o SQLite).
# As flags -ldflags="-s -w" removem dados de debug para deixar o binário menor e mais rápido.
RUN CGO_ENABLED=1 GOOS=linux go build -ldflags="-s -w" -o telegram-bot ./src/main.go

# ==========================================
# Estágio 2: Produção (Runtime)
# ==========================================
FROM debian:bookworm-slim

# Define a variável de ambiente para não pedir interações na instalação do apt
ENV DEBIAN_FRONTEND=noninteractive

# Define o fuso horário (ajuste se necessário)
ENV TZ=America/Sao_Paulo

WORKDIR /app

# Atualiza os pacotes e instala as dependências de runtime:
# - ffmpeg: Para conversão de áudio em RAM
# - ca-certificates: Obrigatório para fazer requisições HTTPS para o OpenRouter e Telegram
# - tzdata: Para o Go gerenciar fusos horários corretamente
RUN apt-get update && apt-get install -y --no-install-recommends \
    ffmpeg \
    ca-certificates \
    tzdata \
    && rm -rf /var/lib/apt/lists/*

# Copia o binário compilado do estágio anterior para a imagem final
COPY --from=builder /app/telegram-bot .

# Cria uma pasta dedicada para o banco de dados do SQLite
RUN mkdir -p /app/data

# Comando de inicialização do bot
CMD ["./telegram-bot"]