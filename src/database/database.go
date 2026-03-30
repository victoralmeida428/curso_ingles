package database

import (
	"database/sql"
	"fmt"
	"log"
	"sync" // Pacote essencial para o Singleton no Go
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// UserData representa as preferências de estudo do usuário
type UserData struct {
	ID        int64
	Lang      string
	Tipo      string
	Nivel     string
	ExpiresAt int64 // Timestamp de expiração
}

func (u *UserData) IsSubscribed() bool {
	return u.ExpiresAt >= time.Now().Unix()
}

// Variáveis globais privadas do pacote
var (
	db    *sql.DB
	dbErr error     // Armazena qualquer erro que ocorra durante a inicialização única
	once  sync.Once // Garante que a função interna rode apenas uma vez
)

// GetDB retorna a conexão Singleton com o banco de dados e garante a criação da tabela.
// Totalmente seguro (thread-safe) para múltiplas chamadas concorrentes.
func GetDB(dbPath string) (*sql.DB, error) {
	once.Do(func() {
		db, dbErr = sql.Open("sqlite3", dbPath)
		if dbErr != nil {
			dbErr = fmt.Errorf("erro ao abrir banco de dados: %w", dbErr)
			return // Interrompe apenas a função interna do once.Do
		}

		query := `
		CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY,
			lang TEXT DEFAULT '',
			tipo TEXT DEFAULT '',
			nivel TEXT DEFAULT '',
			expires_at INTEGER DEFAULT 0
		);`

		if _, err := db.Exec(query); err != nil {
			dbErr = fmt.Errorf("erro ao criar tabela: %w", err)
			return
		}
	})

	// Retorna a instância global que foi (ou já havia sido) configurada pelo once.Do
	return db, dbErr
}

// SaveUser atualiza ou insere as preferências do usuário de forma segura
func SaveUser(user UserData) error {
	query := `
	INSERT INTO users (id, lang, tipo, nivel) 
	VALUES (?, ?, ?, ?)
	ON CONFLICT(id) DO UPDATE SET 
		lang = excluded.lang,
		tipo = excluded.tipo,
		nivel = excluded.nivel;`

	// Removi o 'db *sql.DB' dos parâmetros, pois agora o pacote gerencia o próprio Singleton
	_, err := db.Exec(query, user.ID, user.Lang, user.Tipo, user.Nivel)
	return err
}

// UpdateSubscription prolonga o acesso do usuário
func UpdateSubscription(chatID int64, days int) error {
	// Calcula a nova data: se já for VIP, soma ao tempo restante; se não, soma a partir de agora
	user := GetUser(chatID)
	var newExpiration int64
	now := time.Now().Unix()

	if user.ExpiresAt > now {
		newExpiration = user.ExpiresAt + int64(days*24*60*60)
	} else {
		newExpiration = now + int64(days*24*60*60)
	}

	_, err := db.Exec("UPDATE users SET expires_at = ? WHERE id = ?", newExpiration, chatID)
	return err
}

// GetUser busca as informações do usuário. Retorna os dados vazios se ele não existir ainda.
func GetUser(id int64) UserData {
	user := UserData{ID: id}
	query := `SELECT lang, tipo, nivel, expires_at FROM users WHERE id = ?`

	// Utiliza o 'db' global diretamente
	err := db.QueryRow(query, id).Scan(&user.Lang, &user.Tipo, &user.Nivel, &user.ExpiresAt)
	if err != nil && err != sql.ErrNoRows {
		log.Printf("Erro ao buscar usuário %d no banco: %v\n", id, err)
	}

	return user
}
