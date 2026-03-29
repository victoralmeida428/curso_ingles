# Nome do arquivo executável final
BINARY_NAME=./bin/telegram-bot

# Caminho do arquivo main.go (ajuste se o seu main estiver em ./cmd/bot/main.go, por exemplo)
MAIN_PATH=./src/main.go

# Evita conflitos com arquivos que tenham o mesmo nome das regras
.PHONY: all build run dev test clean fmt vet tidy

# Regra padrão executada quando você digita apenas "make"
all: tidy fmt vet build

build:
	@echo "🔨 Compilando o projeto..."
	go build -o ${BINARY_NAME} ${MAIN_PATH}

run: build
	@echo "🚀 Iniciando o bot (Binário compilado)..."
	./${BINARY_NAME}

dev:
	@echo "⚡ Rodando em modo de desenvolvimento (go run)..."
	go run ${MAIN_PATH}

test:
	@echo "🧪 Rodando todos os testes..."
	go test -v ./...

clean:
	@echo "🧹 Limpando arquivos compilados e cache..."
	go clean
	rm -f ${BINARY_NAME}

fmt:
	@echo "✨ Formatando o código fonte..."
	go fmt ./...

vet:
	@echo "🔍 Inspecionando o código em busca de erros (go vet)..."
	go vet ./...

tidy:
	@echo "📦 Baixando e limpando dependências do go.mod..."
	go mod tidy