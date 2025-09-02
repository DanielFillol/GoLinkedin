package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/your-org/linkedin-visible-crawler/internal/crawler"
)

func main() {
	// Carregar .env (dev)
	if _, err := os.Stat(".env"); err == nil {
		if err := godotenv.Load(); err != nil {
			log.Printf("Aviso: erro ao carregar .env: %v", err)
		}
	}

	// Flags
	query := flag.String("query", "", "Query de busca (pode ser repetida)")
	queriesFile := flag.String("queries-file", "", "Arquivo com queries (uma por linha)")
	headless := flag.Bool("headless", true, "Executar em modo headless")
	maxCards := flag.Int("max-cards", 60, "Máximo de cards para ler")
	maxConnects := flag.Int("max-connects", 3, "Máximo de convites por página")
	csvOut := flag.String("csv-out", "", "Arquivo CSV de saída (opcional)")
	flag.Parse()

	// Credenciais
	email := os.Getenv("LINKEDIN_EMAIL")
	password := os.Getenv("LINKEDIN_PASSWORD")
	if email == "" || password == "" {
		log.Fatal("LINKEDIN_EMAIL e LINKEDIN_PASSWORD devem estar definidos (env/.env)")
	}

	// Coletar queries
	var queries []string
	if *query != "" {
		queries = append(queries, *query)
	}
	if *queriesFile != "" {
		content, err := os.ReadFile(*queriesFile)
		if err != nil {
			log.Fatalf("Erro ao ler arquivo de queries: %v", err)
		}
		for _, line := range strings.Split(string(content), "\n") {
			if s := strings.TrimSpace(line); s != "" {
				queries = append(queries, s)
			}
		}
	}
	if len(queries) == 0 {
		log.Fatal("Nenhuma query especificada. Use --query ou --queries-file")
	}

	// Config nova (RunConfig)
	cfg := crawler.RunConfig{
		MaxCardsRead:       *maxCards,
		MaxConnectsPerPage: *maxConnects,
		Queries:            queries,
		Headless:           *headless,
	}

	if *csvOut == "" {
		*csvOut = fmt.Sprintf("linkedin_visible_%s.csv", time.Now().Format("20060102_150405"))
	}

	log.Printf("Iniciando crawler: %d queries | headless=%v | maxCards=%d | maxConnects=%d",
		len(queries), cfg.Headless, cfg.MaxCardsRead, cfg.MaxConnectsPerPage)

	creds := crawler.Creds{Email: email, Password: password}

	// Agregar contatos para salvar CSV ao final
	var capturedAll []crawler.Contact
	var invitesTotal int

	callbacks := crawler.Callbacks{
		OnCaptured: func(c crawler.Contact) {
			capturedAll = append(capturedAll, c)
			log.Printf("📇 Capturado: %s | %s | %s | %s", c.Name, c.Title, c.Company, c.LinkedIn)
		},
		OnInviteSent: func(c crawler.Contact) {
			invitesTotal++
			log.Printf("🤝 Convite enviado: %s | %s | %s", c.Name, c.Title, c.LinkedIn)
		},
		OnLog: func(line string) {
			log.Println(line)
		},
	}

	// Executa o engine (login + 2FA aguardado de forma robusta + queries)
	engine := crawler.NewEngine()
	if err := engine.Run(cfg, creds, callbacks); err != nil {
		log.Fatalf("Erro no crawler: %v", err)
	}

	// Dedup e salvar CSV
	unique := crawler.RemoveDup(capturedAll)
	if err := crawler.SaveCSV(*csvOut, unique); err != nil {
		log.Fatalf("Erro ao salvar CSV: %v", err)
	}

	log.Printf("\n=== RESUMO ===")
	log.Printf("Total capturados: %d", len(capturedAll))
	log.Printf("Únicos: %d", len(unique))
	log.Printf("Convites enviados: %d", invitesTotal)
	log.Printf("CSV salvo em: %s", *csvOut)
	log.Println("✅ Crawler concluído com sucesso")
}
