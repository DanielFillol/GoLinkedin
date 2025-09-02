package crawler

import "time"

// Contact representa um perfil capturado
type Contact struct {
	Name     string `json:"name"`
	Title    string `json:"title"`
	Company  string `json:"company"`
	Location string `json:"location"`
	LinkedIn string `json:"linkedin_url"`
}

// Creds representa credenciais do LinkedIn
type Creds struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// RunConfig configuração para execução do crawler
type RunConfig struct {
	MaxCardsRead       int      `json:"max_cards"`
	MaxConnectsPerPage int      `json:"max_connects"`
	Queries            []string `json:"queries"`
	Headless           bool     `json:"headless"`
}

// Callbacks para integração com a UI
type Callbacks struct {
	OnCaptured   func(c Contact) // incrementa captured_session via SSE
	OnInviteSent func(c Contact) // grava CSV + atualiza invites_week
	OnLog        func(line string)
}

// InviteRecord registro de convite enviado
type InviteRecord struct {
	Timestamp    time.Time `json:"timestamp"`
	UserEmail    string    `json:"user_email"`
	ProfileName  string    `json:"profile_name"`
	ProfileTitle string    `json:"profile_title"`
	Company      string    `json:"company"`
	Location     string    `json:"location"`
	LinkedInURL  string    `json:"linkedin_url"`
	Query        string    `json:"query"`
}
