package ui

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/your-org/linkedin-visible-crawler/internal/crawler"
)

// SSEEvent representa um evento SSE
type SSEEvent struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

// SSEBroker gerencia conexões SSE
type SSEBroker struct {
	clients    map[chan SSEEvent]bool
	register   chan chan SSEEvent
	unregister chan chan SSEEvent
	mutex      sync.RWMutex
}

// NewSSEBroker cria nova instância do broker
func NewSSEBroker() *SSEBroker {
	return &SSEBroker{
		clients:    make(map[chan SSEEvent]bool),
		register:   make(chan chan SSEEvent),
		unregister: make(chan chan SSEEvent),
	}

}

// Start inicia o broker SSE
func (b *SSEBroker) Start() {
	go func() {
		for {
			select {
			case client := <-b.register:
				b.mutex.Lock()
				b.clients[client] = true
				b.mutex.Unlock()

			case client := <-b.unregister:
				b.mutex.Lock()
				delete(b.clients, client)
				close(client)
				b.mutex.Unlock()
			}
		}
	}()
}

// RegisterClient registra novo cliente SSE
func (b *SSEBroker) RegisterClient() chan SSEEvent {
	client := make(chan SSEEvent, 100)
	b.register <- client
	return client
}

// UnregisterClient remove cliente SSE
func (b *SSEBroker) UnregisterClient(client chan SSEEvent) {
	b.unregister <- client
}

// PublishEvent publica evento para todos os clientes
func (b *SSEBroker) PublishEvent(event SSEEvent) {
	b.mutex.RLock()
	defer b.mutex.RUnlock()

	for client := range b.clients {
		select {
		case client <- event:
		default:
			// Cliente não está respondendo, remover
			go func(c chan SSEEvent) {
				b.unregister <- c
			}(client)
		}
	}
}

// PublishMetrics publica métricas de contadores
func (b *SSEBroker) PublishMetrics(capturedSession, invitesWeek int) {
	event := SSEEvent{
		Type: "metrics",
		Data: map[string]interface{}{
			"captured_session": capturedSession,
			"invites_week":     invitesWeek,
			"invites_limit":    200,
		},
	}
	b.PublishEvent(event)
}

// PublishInvite publica evento de convite enviado
func (b *SSEBroker) PublishInvite(invite crawler.InviteRecord) {
	event := SSEEvent{
		Type: "invite",
		Data: map[string]interface{}{
			"timestamp":    invite.Timestamp.Format(time.RFC3339),
			"user_email":   invite.UserEmail,
			"profile_name": invite.ProfileName,
			"title":        invite.ProfileTitle,
			"company":      invite.Company,
			"location":     invite.Location,
			"linkedin_url": invite.LinkedInURL,
			"query":        invite.Query,
		},
	}
	b.PublishEvent(event)
}

// PublishLog publica linha de log
func (b *SSEBroker) PublishLog(line string) {
	fmt.Printf("📝 Publicando log via SSE: %s\n", line) // Debug
	event := SSEEvent{
		Type: "log",
		Data: map[string]interface{}{
			"line": line,
		},
	}
	b.PublishEvent(event)
}

// PublishError publica erro
func (b *SSEBroker) PublishError(message string) {
	event := SSEEvent{
		Type: "error",
		Data: map[string]interface{}{
			"message": message,
		},
	}
	b.PublishEvent(event)
}

// FormatSSEMessage formata mensagem SSE
func FormatSSEMessage(event SSEEvent) string {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Sprintf("data: {\"type\":\"error\",\"data\":{\"message\":\"Erro ao serializar evento\"}}\n\n")
	}
	return fmt.Sprintf("data: %s\n\n", string(data))
}
