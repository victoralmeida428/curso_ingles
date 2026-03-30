package main

import (
	"fmt"
	"log"
	"net/http"

	"curso/src/config"
	"curso/src/database"
	"curso/src/openrouter"
	"curso/src/payment"
	"curso/src/telegram"

	"github.com/joho/godotenv"
)

func init() {
	_ = godotenv.Load()
}

func main() {
	// 1. Carrega as configurações globais (lê o .env automaticamente)
	cfg := config.LoadConfig()

	// 2. Inicializa o Banco de Dados (Singleton)
	// Chamamos o GetDB aqui no início para garantir que o arquivo e as tabelas sejam criados
	// antes de qualquer outra parte do sistema tentar acessá-los.
	db, err := database.GetDB(cfg.DatabasePath)
	if err != nil {
		log.Fatalf("❌ Erro fatal ao inicializar o banco de dados: %v", err)
	}
	defer db.Close() // Fecha a conexão graciosamente quando o programa for encerrado

	fmt.Println("✅ Banco de dados inicializado e sincronizado!")

	// 3. Inicializa o cliente do OpenRouter (Preparativo para o próximo passo)
	client := openrouter.NewClient(cfg.OpenRouterAPIKey)

	// 4. Inicia o Worker do Telegram
	if cfg.TelegramBotToken == "" {
		log.Fatal("❌ TELEGRAM_BOT_TOKEN não encontrado. Verifique seu arquivo .env.")
	}

	fmt.Println("\n==================================================")
	fmt.Println(" 🚀 INICIANDO WORKER DO TELEGRAM ")
	fmt.Println("==================================================")

	// Como o banco agora é um Singleton gerenciado pelo próprio pacote 'database',
	// só precisamos passar o token para o bot.
	go func() {
		w := telegram.NewWorker(cfg.TelegramBotToken, db, client)
		w.StartWorker()
	}()

	// 5. Inicia o Webhook do Stripe
	if cfg.StripeSecretKey == "" || cfg.StripeWebhookSecret == "" {
		log.Fatal("❌ Chaves do Stripe não encontradas. Verifique seu arquivo .env.")
	}

	fmt.Println("\n==================================================")
	fmt.Println(" 💳 INICIANDO WEBHOOK DO STRIPE ")
	fmt.Println("==================================================")

	http.HandleFunc("/webhook", payment.HandleWebhook(cfg))
	log.Fatal(http.ListenAndServe(":8080", nil))
}
