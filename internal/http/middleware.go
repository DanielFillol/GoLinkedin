package http

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// SessionState representa o estado da sessão
type SessionState struct {
	LinkedInEmail string
	LinkedInPass  string
	QueriesPath   string
	CapturedCount int
	CreatedAt     time.Time
}

// SessionStore gerencia sessões em memória
type SessionStore struct {
	sessions sync.Map
}

// NewSessionStore cria nova instância do store
func NewSessionStore() *SessionStore {
	return &SessionStore{}
}

// CreateSession cria nova sessão
func (s *SessionStore) CreateSession() string {
	sessionID := generateSessionID()
	s.sessions.Store(sessionID, &SessionState{
		CreatedAt: time.Now(),
	})
	return sessionID
}

// GetSession obtém sessão por ID
func (s *SessionStore) GetSession(sessionID string) (*SessionState, bool) {
	if value, ok := s.sessions.Load(sessionID); ok {
		if session, ok := value.(*SessionState); ok {
			return session, true
		}
	}
	return nil, false
}

// SetCredentials define credenciais para uma sessão
func (s *SessionStore) SetCredentials(sessionID string, email, password string) bool {
	if value, ok := s.sessions.Load(sessionID); ok {
		if session, ok := value.(*SessionState); ok {
			session.LinkedInEmail = email
			session.LinkedInPass = password
			return true
		}
	}
	return false
}

// SetQueriesPath define caminho do arquivo de queries para uma sessão
func (s *SessionStore) SetQueriesPath(sessionID string, path string) bool {
	if value, ok := s.sessions.Load(sessionID); ok {
		if session, ok := value.(*SessionState); ok {
			session.QueriesPath = path
			return true
		}
	}
	return false
}

// IncrementCaptured incrementa contador de contatos capturados
func (s *SessionStore) IncrementCaptured(sessionID string) bool {
	if value, ok := s.sessions.Load(sessionID); ok {
		if session, ok := value.(*SessionState); ok {
			session.CapturedCount++
			return true
		}
	}
	return false
}

// CleanupExpired remove sessões expiradas (24h)
func (s *SessionStore) CleanupExpired() {
	expiryTime := time.Now().Add(-24 * time.Hour)

	s.sessions.Range(func(key, value interface{}) bool {
		if session, ok := value.(*SessionState); ok {
			if session.CreatedAt.Before(expiryTime) {
				s.sessions.Delete(key)
			}
		}
		return true
	})
}

// generateSessionID gera ID único para sessão
func generateSessionID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// SessionMiddleware middleware para gerenciar sessões
func SessionMiddleware(store *SessionStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Verificar cookie de sessão
		sessionID, err := c.Cookie("session_id")
		if err != nil || sessionID == "" {
			// Criar nova sessão
			sessionID = store.CreateSession()
			c.SetCookie("session_id", sessionID, 86400, "/", "", false, true)
		}

		// Verificar se sessão existe
		session, exists := store.GetSession(sessionID)
		if !exists {
			// Sessão expirada, criar nova
			sessionID = store.CreateSession()
			c.SetCookie("session_id", sessionID, 86400, "/", "", false, true)
			session, _ = store.GetSession(sessionID)
		}

		// Adicionar sessão ao contexto
		c.Set("session_id", sessionID)
		c.Set("session", session)

		c.Next()
	}
}

// RequireCredentials middleware para rotas que precisam de credenciais
func RequireCredentials() gin.HandlerFunc {
	return func(c *gin.Context) {
		session, exists := c.Get("session")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Sessão não encontrada"})
			c.Abort()
			return
		}

		sessionState, ok := session.(*SessionState)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro interno da sessão"})
			c.Abort()
			return
		}

		if sessionState.LinkedInEmail == "" || sessionState.LinkedInPass == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Credenciais do LinkedIn não configuradas"})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireQueries middleware para rotas que precisam de arquivo de queries
func RequireQueries() gin.HandlerFunc {
	return func(c *gin.Context) {
		session, exists := c.Get("session")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Sessão não encontrada"})
			c.Abort()
			return
		}

		sessionState, ok := session.(*SessionState)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro interno da sessão"})
			c.Abort()
			return
		}

		if sessionState.QueriesPath == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Arquivo de queries não configurado"})
			c.Abort()
			return
		}

		c.Next()
	}
}
