package http

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"encoding/csv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/your-org/linkedin-visible-crawler/internal/crawler"
	"github.com/your-org/linkedin-visible-crawler/internal/storage"
	"github.com/your-org/linkedin-visible-crawler/internal/ui"
)

// Handlers gerencia todos os handlers HTTP
type Handlers struct {
	templates     *ui.Templates
	sseBroker     *ui.SSEBroker
	inviteStorage *storage.InviteStorage
	weeklyCounter *storage.WeeklyCounter
	sessionStore  *SessionStore
}

// NewHandlers cria nova instância dos handlers
func NewHandlers(templates *ui.Templates, sseBroker *ui.SSEBroker,
	inviteStorage *storage.InviteStorage, weeklyCounter *storage.WeeklyCounter,
	sessionStore *SessionStore) *Handlers {
	return &Handlers{
		templates:     templates,
		sseBroker:     sseBroker,
		inviteStorage: inviteStorage,
		weeklyCounter: weeklyCounter,
		sessionStore:  sessionStore,
	}
}

// Home renderiza a página principal
func (h *Handlers) Home(c *gin.Context) {
	html, err := h.templates.RenderHome(map[string]interface{}{})
	if err != nil {
		c.String(http.StatusInternalServerError, "Erro ao renderizar página")
		return
	}
	c.Header("Content-Type", "text/html")
	c.String(http.StatusOK, html)
}

// SetCredentials define credenciais do LinkedIn para a sessão
func (h *Handlers) SetCredentials(c *gin.Context) {
	sessionID := c.MustGet("session_id").(string)
	email := c.PostForm("linkedin_email")
	password := c.PostForm("linkedin_password")

	if email == "" || password == "" {
		c.String(http.StatusBadRequest, `<div class="text-red-600">Email e senha são obrigatórios</div>`)
		return
	}

	// Salvar credenciais na sessão
	h.sessionStore.SetCredentials(sessionID, email, password)

	response := fmt.Sprintf(`
		<div class="text-green-600 bg-green-50 p-3 rounded-md">
			<strong>✅ Credenciais ativas para esta sessão</strong><br>
			Email: %s<br>
			<small class="text-gray-600">Senha não será exibida por segurança</small>
		</div>
	`, email)

	c.String(http.StatusOK, response)
}

// UploadQueriesFile faz upload de arquivo .txt com queries
func (h *Handlers) UploadQueriesFile(c *gin.Context) {
	sessionID := c.MustGet("session_id").(string)

	file, err := c.FormFile("queries_file")
	if err != nil {
		c.String(http.StatusBadRequest, `<div class="text-red-600">Erro ao processar arquivo</div>`)
		return
	}

	// Verificar extensão
	if !strings.HasSuffix(file.Filename, ".txt") {
		c.String(http.StatusBadRequest, `<div class="text-red-600">Apenas arquivos .txt são aceitos</div>`)
		return
	}

	// Criar diretório de uploads se não existir
	uploadDir := "data/uploads/queries"
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		c.String(http.StatusInternalServerError, `<div class="text-red-600">Erro ao criar diretório de uploads</div>`)
		return
	}

	// Gerar nome único para o arquivo
	filename := uuid.New().String() + ".txt"
	filepath := filepath.Join(uploadDir, filename)

	// Salvar arquivo
	if err := c.SaveUploadedFile(file, filepath); err != nil {
		c.String(http.StatusInternalServerError, `<div class="text-red-600">Erro ao salvar arquivo</div>`)
		return
	}

	// Associar arquivo à sessão
	h.sessionStore.SetQueriesPath(sessionID, filepath)

	response := fmt.Sprintf(`
		<div class="text-green-600 bg-green-50 p-3 rounded-md">
			<strong>✅ Arquivo carregado com sucesso</strong><br>
			Nome: %s<br>
			<button onclick="document.getElementById('queries-status').innerHTML=''" 
					class="mt-2 text-sm text-blue-600 hover:text-blue-800 underline">
				Trocar arquivo
			</button>
		</div>
	`, file.Filename)

	c.String(http.StatusOK, response)
}

// UploadQueriesText salva queries da textarea como arquivo .txt
func (h *Handlers) UploadQueriesText(c *gin.Context) {
	sessionID := c.MustGet("session_id").(string)
	queriesText := c.PostForm("queries_text")

	if queriesText == "" {
		c.String(http.StatusBadRequest, `<div class="text-red-600">Texto das queries é obrigatório</div>`)
		return
	}

	// Criar diretório de uploads se não existir
	uploadDir := "data/uploads/queries"
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		c.String(http.StatusInternalServerError, `<div class="text-red-600">Erro ao criar diretório de uploads</div>`)
		return
	}

	// Gerar nome único para o arquivo
	filename := uuid.New().String() + ".txt"
	filepath := filepath.Join(uploadDir, filename)

	// Salvar arquivo
	if err := os.WriteFile(filepath, []byte(queriesText), 0644); err != nil {
		c.String(http.StatusInternalServerError, `<div class="text-red-600">Erro ao salvar arquivo</div>`)
		return
	}

	// Associar arquivo à sessão
	h.sessionStore.SetQueriesPath(sessionID, filepath)

	response := `
		<div class="text-green-600 bg-green-50 p-3 rounded-md">
			<strong>✅ Queries salvas com sucesso</strong><br>
			<small class="text-gray-600">Texto foi salvo como arquivo temporário</small><br>
			<button onclick="document.getElementById('queries-status').innerHTML=''" 
					class="mt-2 text-sm text-blue-600 hover:text-blue-800 underline">
				Trocar queries
			</button>
		</div>
	`

	c.String(http.StatusOK, response)
}

// RunCrawler executa o crawler em goroutine
func (h *Handlers) RunCrawler(c *gin.Context) {
	sessionID := c.MustGet("session_id").(string)
	session := c.MustGet("session").(*SessionState)

	// Validar credenciais
	if session.LinkedInEmail == "" || session.LinkedInPass == "" {
		c.String(http.StatusBadRequest, `<div class="text-red-600">Configure as credenciais do LinkedIn primeiro</div>`)
		return
	}

	// Validar arquivo de queries
	if session.QueriesPath == "" {
		c.String(http.StatusBadRequest, `<div class="text-red-600">Configure o arquivo de queries primeiro</div>`)
		return
	}

	// Verificar limite semanal
	canSend, count, err := h.weeklyCounter.CanSendInvite(session.LinkedInEmail)
	if err != nil {
		c.String(http.StatusInternalServerError, `<div class="text-red-600">Erro ao verificar limite semanal</div>`)
		return
	}

	if !canSend {
		c.String(http.StatusBadRequest, fmt.Sprintf(`
			<div class="text-red-600 bg-red-50 p-3 rounded-md">
				<strong>❌ Limite semanal atingido</strong><br>
				Você já enviou %d convites esta semana (limite: 200)
			</div>
		`, count))
		return
	}

	// Obter parâmetros
	maxCards, _ := strconv.Atoi(c.PostForm("max_cards"))
	if maxCards == 0 {
		maxCards = 60
	}

	maxConnects, _ := strconv.Atoi(c.PostForm("max_connects"))
	if maxConnects == 0 {
		maxConnects = 3
	}

	// Verificar modo headless
	headlessMode := c.PostForm("headless_mode") == "on"

	// Ler queries do arquivo
	queriesBytes, err := os.ReadFile(session.QueriesPath)
	if err != nil {
		c.String(http.StatusInternalServerError, `<div class="text-red-600">Erro ao ler arquivo de queries</div>`)
		return
	}

	queries := strings.Split(string(queriesBytes), "\n")
	// Filtrar linhas vazias
	var cleanQueries []string
	for _, q := range queries {
		if strings.TrimSpace(q) != "" {
			cleanQueries = append(cleanQueries, strings.TrimSpace(q))
		}
	}

	if len(cleanQueries) == 0 {
		c.String(http.StatusBadRequest, `<div class="text-red-600">Arquivo de queries está vazio</div>`)
		return
	}

	// Configurar crawler
	cfg := crawler.RunConfig{
		MaxCardsRead:       maxCards,
		MaxConnectsPerPage: maxConnects,
		Queries:            cleanQueries,
		Headless:           headlessMode,
	}

	creds := crawler.Creds{
		Email:    session.LinkedInEmail,
		Password: session.LinkedInPass,
	}

	// Callbacks para integração com UI
	callbacks := crawler.Callbacks{
		OnCaptured: func(contact crawler.Contact) {
			// Incrementar contador de sessão
			h.sessionStore.IncrementCaptured(sessionID)

			// Obter valores atualizados
			updatedSession, _ := h.sessionStore.GetSession(sessionID)
			weekly, _ := h.weeklyCounter.CountThisWeek(session.LinkedInEmail)

			// Publicar métricas via SSE
			h.sseBroker.PublishMetrics(updatedSession.CapturedCount, weekly)
			h.sseBroker.PublishLog(fmt.Sprintf("📊 Contato capturado: %s (%d total)", contact.Name, updatedSession.CapturedCount))
		},
		OnInviteSent: func(contact crawler.Contact) {
			h.sseBroker.PublishLog(fmt.Sprintf("🎯 Callback OnInviteSent chamado para: %s", contact.Name))

			// Verificar limite antes de enviar
			canSend, _, err := h.weeklyCounter.CanSendInvite(session.LinkedInEmail)
			if err != nil || !canSend {
				h.sseBroker.PublishLog("Limite semanal atingido, pulando convite")
				return
			}

			// Salvar no CSV
			invite := crawler.InviteRecord{
				Timestamp:    time.Now(),
				UserEmail:    session.LinkedInEmail,
				ProfileName:  contact.Name,
				ProfileTitle: contact.Title,
				Company:      contact.Company,
				Location:     contact.Location,
				LinkedInURL:  contact.LinkedIn,
				Query:        cleanQueries[0], // Query atual
			}

			if err := h.inviteStorage.AppendInvite(invite); err != nil {
				h.sseBroker.PublishError("Erro ao salvar convite: " + err.Error())
				return
			}

			// Publicar via SSE
			h.sseBroker.PublishInvite(invite)

			// Atualizar métricas com valores atualizados
			updatedSession, _ := h.sessionStore.GetSession(sessionID)
			weekly, _ := h.weeklyCounter.CountThisWeek(session.LinkedInEmail)
			h.sseBroker.PublishMetrics(updatedSession.CapturedCount, weekly)
			h.sseBroker.PublishLog(fmt.Sprintf("✅ Convite enviado para: %s (%d convites esta semana)", contact.Name, weekly))
		},
		OnLog: func(line string) {
			h.sseBroker.PublishLog(line)
		},
	}

	// Executar crawler em goroutine
	go func() {
		engine := crawler.NewEngine()
		if err := engine.Run(cfg, creds, callbacks); err != nil {
			h.sseBroker.PublishError("Erro no crawler: " + err.Error())
		}
	}()

	response := `
		<div class="text-green-600 bg-green-50 p-3 rounded-md">
			<strong>🚀 Crawler iniciado com sucesso!</strong><br>
			<small class="text-gray-600">Acompanhe o progresso na área de status ao vivo</small>
		</div>
	`

	c.String(http.StatusOK, response)
}

// ListInvites lista convites com paginação
func (h *Handlers) ListInvites(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "0"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "50"))

	invites, total, err := h.inviteStorage.ListInvites(page, pageSize)
	if err != nil {
		c.String(http.StatusInternalServerError, "Erro ao listar convites")
		return
	}

	html, err := h.templates.RenderInvites(invites, total, page, pageSize)
	if err != nil {
		c.String(http.StatusInternalServerError, "Erro ao renderizar tabela")
		return
	}

	c.Header("Content-Type", "text/html")
	c.String(http.StatusOK, html)
}

// GetMetrics retorna métricas atuais
func (h *Handlers) GetMetrics(c *gin.Context) {
	session := c.MustGet("session").(*SessionState)

	// Obter contadores
	captured := session.CapturedCount
	weekly := 0

	// Se há email configurado, obter contadores da semana
	if session.LinkedInEmail != "" {
		weekly, _ = h.weeklyCounter.CountThisWeek(session.LinkedInEmail)
	} else {
		// Se não há email configurado, tentar obter total geral do CSV
		// Isso permite mostrar métricas mesmo sem credenciais configuradas
		total, err := h.inviteStorage.GetTotalCount()
		if err == nil && total > 0 {
			weekly = total
		}
	}

	// Se não há contatos capturados na sessão mas há convites da semana,
	// mostrar o total de convites como "capturados" (já que foram capturados em execuções anteriores)
	if captured == 0 && weekly > 0 {
		captured = weekly
	}

	// Retornar como JSON
	c.JSON(http.StatusOK, gin.H{
		"captured_session": captured,
		"invites_week":     weekly,
		"invites_limit":    200,
	})
}

// ExportInvitesCSV exporta convites como CSV
func (h *Handlers) ExportInvitesCSV(c *gin.Context) {
	// Listar todos os convites
	invites, _, err := h.inviteStorage.ListInvites(0, 10000)
	if err != nil {
		c.String(http.StatusInternalServerError, "Erro ao exportar CSV")
		return
	}

	// Configurar headers para download
	filename := fmt.Sprintf("linkedin_invites_%s.csv", time.Now().Format("20060102_150405"))
	c.Header("Content-Disposition", "attachment; filename="+filename)
	c.Header("Content-Type", "text/csv")

	// Escrever CSV
	writer := csv.NewWriter(c.Writer)
	defer writer.Flush()

	// Cabeçalho
	header := []string{
		"timestamp",
		"user_email",
		"profile_name",
		"profile_title",
		"company",
		"location",
		"linkedin_url",
		"query",
	}
	writer.Write(header)

	// Dados
	for _, invite := range invites {
		row := []string{
			invite.Timestamp.Format(time.RFC3339),
			invite.UserEmail,
			invite.ProfileName,
			invite.ProfileTitle,
			invite.Company,
			invite.Location,
			invite.LinkedInURL,
			invite.Query,
		}
		writer.Write(row)
	}
}

// SSEEvents endpoint para Server-Sent Events
func (h *Handlers) SSEEvents(c *gin.Context) {
	// Configurar headers para SSE
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Access-Control-Allow-Origin", "*")
	c.Header("X-Accel-Buffering", "no") // Desabilitar buffering do nginx

	// Registrar cliente
	client := h.sseBroker.RegisterClient()
	defer h.sseBroker.UnregisterClient(client)

	// Enviar evento de teste inicial
	h.sseBroker.PublishLog("🔗 Cliente SSE conectado - logs ativos")

	// Enviar métricas iniciais se houver sessão
	if sessionID, exists := c.Get("session_id"); exists {
		if session, ok := h.sessionStore.GetSession(sessionID.(string)); ok {
			weekly, _ := h.weeklyCounter.CountThisWeek(session.LinkedInEmail)
			captured := session.CapturedCount

			// Se não há contatos capturados na sessão mas há convites da semana,
			// mostrar o total de convites como "capturados"
			if captured == 0 && weekly > 0 {
				captured = weekly
			}

			h.sseBroker.PublishMetrics(captured, weekly)
		}
	}

	// Manter conexão ativa
	c.Stream(func(w io.Writer) bool {
		select {
		case event := <-client:
			// Log de debug
			fmt.Printf("📡 Enviando evento SSE: %+v\n", event)
			// Enviar evento diretamente como JSON
			c.SSEvent("message", event)
			return true
		case <-c.Request.Context().Done():
			fmt.Printf("📡 Cliente SSE desconectado\n")
			return false
		}
	})
}
