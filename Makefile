# Makefile para LinkedIn Visible Crawler

.PHONY: run build clean fmt tidy test help \
        dev install clean-data \
        docker-up docker-down docker-logs docker-rebuild

# Comandos principais
run: ## Executar o servidor web local (sem Docker)
	go run ./cmd/web

build: ## Construir binário local
	go build -o bin/web ./cmd/web

clean: ## Limpar arquivos gerados
	rm -rf bin/
	rm -rf data/
	go clean

fmt: ## Formatar código Go
	go fmt ./...

tidy: ## Organizar dependências
	go mod tidy

test: ## Executar testes
	go test ./...

# Comandos de desenvolvimento
dev: ## Executar em modo desenvolvimento
	PORT=8080 go run ./cmd/web

install: ## Instalar dependências
	go mod download

# Comandos de limpeza de dados
clean-data: ## Limpar dados (CSV e uploads)
	rm -rf data/

# Integração com Docker
docker-up: ## Subir containers em background
	docker compose up -d

docker-down: ## Derrubar containers
	docker compose down

docker-logs: ## Ver logs em tempo real
	docker compose logs -f --tail=200 linkedin-crawler-web

docker-rebuild: ## Reconstruir imagem e subir com logs
	docker compose down && docker compose up -d --build && docker compose logs -f --tail=200 linkedin-crawler-web

# Ajuda
help: ## Mostrar esta ajuda
	@echo "LinkedIn Visible Crawler - Comandos disponíveis:"
	@echo ""
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)
	@echo ""
	@echo "Exemplos:"
	@echo "  make run           - Executar servidor local na porta 8080"
	@echo "  make docker-up     - Subir containers com Docker Compose"
	@echo "  make docker-logs   - Acompanhar logs do container"
	@echo "  make docker-rebuild- Rebuild da imagem + logs"
