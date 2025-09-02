# LinkedIn Visible Crawler

Um crawler web para LinkedIn que captura perfis visíveis e envia convites de conexão, com interface web moderna e controle de limites semanais.

## 🚀 Características

- **Interface Web Moderna**: UI responsiva com Tailwind CSS, HTMX e Alpine.js
- **Crawler Chromedp**: Motor robusto baseado no projeto anterior
- **Controle de Limites**: Respeita limite de 200 convites/semana por usuário
- **Sessões em Memória**: Credenciais não são persistidas em disco
- **Streaming em Tempo Real**: SSE para logs e métricas ao vivo
- **Armazenamento CSV**: Dados salvos incrementalmente
- **Multiusuário**: Cada sessão mantém suas próprias credenciais

## 🏗️ Arquitetura

```
.
├─ cmd/web/           # Servidor web principal
├─ internal/
│  ├─ ui/            # Templates HTML e SSE
│  ├─ crawler/       # Motor do crawler (chromedp)
│  ├─ storage/       # Armazenamento CSV e contadores
│  └─ http/          # Handlers e middleware
├─ data/             # Dados persistentes (CSV, uploads)
└─ web/static/       # Arquivos estáticos (se necessário)
```

## 📋 Requisitos

### Opção 1: Docker (Recomendado)
- **Docker** e **Docker Compose**
- **Navegador moderno** para a interface web

### Opção 2: Instalação Local
- **Go 1.22+**
- **Chrome/Chromium** instalado
- **Navegador moderno** para a interface web

## 🛠️ Instalação

### 🐳 Com Docker (Recomendado)

1. **Clone o repositório**
```bash
git clone <repository-url>
cd linkedin-visible-crawler
```

2. **Execute com Docker Compose**
```bash
# Produção
docker-compose up -d

# Desenvolvimento (com hot reload)
docker-compose --profile dev up -d
```

3. **Acesse a interface**
```
http://localhost:8080
```

### 🔧 Instalação Local

1. **Clone o repositório**
```bash
git clone <repository-url>
cd linkedin-visible-crawler
```

2. **Instale as dependências**
```bash
go mod tidy
```

3. **Execute o servidor**
```bash
make run
# ou
go run ./cmd/web
```

4. **Acesse a interface**
```
http://localhost:8080
```

## 🎯 Como Usar

### 1. Configurar Credenciais
- Acesse o primeiro card "🔐 Credenciais do LinkedIn"
- Digite seu email e senha do LinkedIn
- Clique em "Usar nesta sessão"

### 2. Configurar Queries
- **Opção A**: Faça upload de arquivo .txt (uma query por linha)
- **Opção B**: Cole as queries diretamente na textarea
- Exemplo de queries:
  ```
  grupo boticário vendas
  startup tecnologia
  consultor financeiro
  ```

### 3. Executar Crawler
- Configure limites (max cards, max convites por página)
- Clique em "Iniciar Crawler"
- **Importante**: Aguarde 8 segundos para 2FA manual

### 4. Acompanhar Progresso
- **Status ao Vivo**: Contadores e barra de progresso
- **Logs em Tempo Real**: Acompanhe cada ação do crawler
- **Tabela de Convites**: Veja todos os convites enviados

## 📊 Controles e Limites

### Limite Semanal
- **200 convites por usuário por semana**
- Barra de progresso muda de cor conforme aproxima do limite
- Botão "Iniciar Crawler" é desabilitado quando limite é atingido

### Configurações por Página
- **Max Cards**: Quantos perfis capturar por página (padrão: 60)
- **Max Connects**: Quantos convites tentar por página (padrão: 3)

## 🔒 Segurança

- **Credenciais em Memória**: Senhas nunca são salvas em disco
- **Sessões Únicas**: Cada usuário tem sua própria sessão
- **Limpeza Automática**: Sessões expiradas são removidas a cada hora
- **Sanitização**: Todo output HTML é escapado automaticamente

## ⚠️ Avisos Importantes

### Termos de Serviço (ToS)
- **Este é um crawler experimental**
- **Use com responsabilidade**
- **Respeite os ToS do LinkedIn**
- **Não abuse da plataforma**

### Riscos de Bloqueio
- **Limite de 200 convites/semana** é uma diretriz do LinkedIn
- **Pausas entre convites** são recomendadas
- **Modo visível** (não headless) é mais seguro
- **Monitore logs** para detectar bloqueios

### Recomendações
- Use pausas entre execuções
- Não execute 24/7
- Monitore métricas de sucesso
- Tenha backup de dados importantes

## 📁 Estrutura de Dados

### Arquivos CSV
```
data/invites.csv
├─ timestamp
├─ user_email
├─ profile_name
├─ profile_title
├─ company
├─ location
├─ linkedin_url
└─ query
```

### Uploads
```
data/uploads/queries/
└─ <uuid>.txt          # Arquivos de queries temporários
```

## 🐳 Comandos Docker

### Produção
```bash
# Construir e executar
docker-compose up -d

# Ver logs
docker-compose logs -f

# Parar
docker-compose down

# Reconstruir
docker-compose up -d --build
```

### Desenvolvimento
```bash
# Executar com hot reload
docker-compose --profile dev up -d

# Ver logs de desenvolvimento
docker-compose --profile dev logs -f

# Parar desenvolvimento
docker-compose --profile dev down
```

### Comandos Úteis
```bash
# Entrar no container
docker-compose exec linkedin-crawler-web sh

# Ver status dos containers
docker-compose ps

# Limpar volumes
docker-compose down -v
```

## 🚀 Comandos Disponíveis

### Docker
```bash
docker-compose up -d          # Executar em produção
docker-compose --profile dev up -d  # Executar em desenvolvimento
docker-compose down           # Parar containers
docker-compose logs -f        # Ver logs
```

### Local
```bash
make run       # Executar servidor
make build     # Construir binário
make dev       # Modo desenvolvimento
make clean     # Limpar arquivos
make fmt       # Formatar código
make tidy      # Organizar dependências
make help      # Ver todos os comandos
```

## 🔧 Configuração

### Variáveis de Ambiente
```bash
PORT=8080                    # Porta do servidor
CHROME_HEADLESS=false        # Modo headless do Chrome
CHROME_USER_AGENT=...        # User agent personalizado
```

### Arquivo .env
```bash
cp .env.example .env
# Edite conforme necessário
```

## 🐛 Troubleshooting

### Problemas Docker

1. **Container não inicia**
   ```bash
   # Ver logs
   docker-compose logs
   
   # Reconstruir imagem
   docker-compose up -d --build
   ```

2. **Erro de permissão no volume**
   ```bash
   # Verificar permissões do diretório data/
   ls -la data/
   
   # Corrigir permissões
   chmod -R 755 data/
   ```

3. **Porta já em uso**
   ```bash
   # Verificar processos na porta 8080
   lsof -i :8080
   
   # Parar containers
   docker-compose down
   ```

### Problemas Comuns

1. **Chrome não encontrado (Local)**
   - Instale Chrome/Chromium
   - Verifique PATH do sistema

2. **Erro de permissão**
   - Verifique permissões do diretório `data/`
   - Execute com privilégios adequados

3. **Limite semanal atingido**
   - Aguarde até segunda-feira
   - Use conta diferente se necessário

4. **Crawler trava**
   - Verifique logs em tempo real
   - Reinicie o servidor se necessário

### Logs
- **Console**: Logs do servidor
- **Interface Web**: Logs em tempo real via SSE
- **Arquivos**: Dados salvos em CSV

## 📈 Monitoramento

### Métricas em Tempo Real
- Contatos capturados na sessão
- Convites enviados na semana
- Progresso do limite semanal
- Status de execução

### Exportação de Dados
- **CSV**: Download completo de convites
- **Filtros**: Por usuário, data, query
- **Paginação**: Navegação por resultados

## 🤝 Contribuição

1. Fork o projeto
2. Crie uma branch para sua feature
3. Commit suas mudanças
4. Push para a branch
5. Abra um Pull Request

## 📄 Licença

Este projeto é para fins educacionais e experimentais. Use com responsabilidade.

## ⚡ Performance

- **Crawler**: Executa em goroutine separada
- **UI**: Atualizações em tempo real via SSE
- **Storage**: Append-only CSV para performance
- **Sessões**: Limpeza automática de memória

## 🔄 Atualizações

- **Limpeza automática** de sessões expiradas
- **Contadores semanais** resetam automaticamente
- **Logs em tempo real** para debugging
- **Métricas persistentes** via CSV

---

**⚠️ Lembre-se**: Este é um crawler experimental. Use com responsabilidade e sempre respeite os termos de serviço do LinkedIn.
