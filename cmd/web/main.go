package main

import (
	"log"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/your-org/linkedin-visible-crawler/internal/http"
	"github.com/your-org/linkedin-visible-crawler/internal/storage"
	"github.com/your-org/linkedin-visible-crawler/internal/ui"
)

func main() {
	// Carregar variáveis de ambiente
	if err := godotenv.Load(); err != nil {
		log.Println("Arquivo .env não encontrado, usando variáveis padrão")
	}

	// Configurar porta
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Inicializar componentes
	log.Println("Inicializando LinkedIn Visible Crawler...")

	// Templates
	templates := ui.NewTemplates()
	log.Println("✅ Templates carregados")

	// SSE Broker
	sseBroker := ui.NewSSEBroker()
	sseBroker.Start()
	log.Println("✅ SSE Broker iniciado")

	// Storage
	inviteStorage := storage.NewInviteStorage()
	weeklyCounter := storage.NewWeeklyCounter(inviteStorage)
	log.Println("✅ Storage inicializado")

	// Session Store
	sessionStore := http.NewSessionStore()
	log.Println("✅ Session Store inicializado")

	// Handlers
	handlers := http.NewHandlers(templates, sseBroker, inviteStorage, weeklyCounter, sessionStore)
	log.Println("✅ Handlers inicializados")

	// Configurar Gin
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	// Middleware de sessão
	router.Use(http.SessionMiddleware(sessionStore))

	// Rotas
	log.Println("Configurando rotas...")

	// Página principal
	router.GET("/", handlers.Home)

	// Sessão
	router.POST("/session/creds", handlers.SetCredentials)

	// Upload de queries
	router.POST("/upload/queries", handlers.UploadQueriesFile)
	router.POST("/upload/queries-text", handlers.UploadQueriesText)

	// Execução do crawler
	router.POST("/run", handlers.RunCrawler)

	// Listagem e exportação de convites
	router.GET("/invites", handlers.ListInvites)
	router.GET("/export/invites.csv", handlers.ExportInvitesCSV)

	// Métricas
	router.GET("/metrics", handlers.GetMetrics)

	// Server-Sent Events
	router.GET("/events", handlers.SSEEvents)

	// Limpeza periódica de sessões expiradas
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()

		for range ticker.C {
			sessionStore.CleanupExpired()
			log.Println("🧹 Sessões expiradas removidas")
		}
	}()

	// Iniciar servidor
	log.Printf("🚀 Servidor iniciando na porta %s", port)
	log.Printf("📱 Acesse: http://localhost:%s", port)
	log.Println("⚠️  Aviso: Este é um crawler experimental. Use com responsabilidade e respeite os ToS do LinkedIn.")

	if err := router.Run(":" + port); err != nil {
		log.Fatalf("❌ Erro ao iniciar servidor: %v", err)
	}
}
