package ui

import (
	"fmt"
	"html/template"
	"strings"

	"github.com/your-org/linkedin-visible-crawler/internal/crawler"
)

// Templates contém todos os templates HTML
type Templates struct {
	home     *template.Template
	invites  *template.Template
	partials map[string]*template.Template
}

// NewTemplates cria nova instância dos templates
func NewTemplates() *Templates {
	tmpl := &Templates{
		partials: make(map[string]*template.Template),
	}

	// Template principal
	tmpl.home = template.Must(template.New("home").Parse(homeTemplate))

	// Template de convites
	tmpl.invites = template.Must(template.New("invites").Parse(invitesTemplate))

	// Partials
	tmpl.partials["invites-table"] = template.Must(template.New("invites-table").Parse(invitesTablePartial))
	tmpl.partials["progress-bar"] = template.Must(template.New("progress-bar").Parse(progressBarPartial))

	return tmpl
}

// RenderHome renderiza a página principal
func (t *Templates) RenderHome(data map[string]interface{}) (string, error) {
	var buf strings.Builder
	if err := t.home.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// RenderInvites renderiza a tabela de convites
func (t *Templates) RenderInvites(invites []crawler.InviteRecord, total int, page int, pageSize int) (string, error) {
	data := map[string]interface{}{
		"Invites":    invites,
		"Total":      total,
		"Page":       page,
		"PageSize":   pageSize,
		"TotalPages": (total + pageSize - 1) / pageSize,
	}

	var buf strings.Builder
	if err := t.invites.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// RenderPartial renderiza um partial específico
func (t *Templates) RenderPartial(name string, data interface{}) (string, error) {
	partial, exists := t.partials[name]
	if !exists {
		return "", fmt.Errorf("partial '%s' não encontrado", name)
	}

	var buf strings.Builder
	if err := partial.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// Template da página principal
const homeTemplate = `<!DOCTYPE html>
<html lang="pt-BR">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>LinkedIn Visible Crawler</title>
    <script src="https://cdn.tailwindcss.com"></script>
    <script src="https://unpkg.com/htmx.org@1.9.10"></script>
    <script src="https://unpkg.com/alpinejs@3.x.x/dist/cdn.min.js" defer></script>
    <script>
        tailwind.config = {
            theme: {
                extend: {
                    colors: {
                        'linkedin': '#0077B5'
                    }
                }
            }
        }
    </script>
</head>
<body class="bg-gray-50 min-h-screen">
    <!-- Topbar -->
    <nav class="bg-linkedin text-white shadow-lg">
        <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
            <div class="flex justify-between h-16">
                <div class="flex items-center">
                    <h1 class="text-xl font-bold">LinkedIn Visible Crawler</h1>
                </div>
                <div class="flex items-center space-x-4">
                    <span class="text-sm opacity-75">Crawler de perfis visíveis</span>
                </div>
            </div>
        </div>
    </nav>

    <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        <div class="grid grid-cols-1 lg:grid-cols-3 gap-8">
            
            <!-- Card 1: Credenciais do LinkedIn -->
            <div class="bg-white rounded-lg shadow-md p-6">
                <h2 class="text-lg font-semibold text-gray-900 mb-4">🔐 Credenciais do LinkedIn</h2>
                
                <form hx-post="/session/creds" hx-target="#credentials-status" hx-swap="innerHTML">
                    <div class="space-y-4">
                        <div>
                            <label class="block text-sm font-medium text-gray-700">Email</label>
                            <input type="email" name="linkedin_email" required
                                   class="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-linkedin focus:ring-linkedin">
                        </div>
                        <div>
                            <label class="block text-sm font-medium text-gray-700">Senha</label>
                            <input type="password" name="linkedin_password" required
                                   class="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-linkedin focus:ring-linkedin">
                        </div>
                        <button type="submit" 
                                class="w-full bg-linkedin text-white py-2 px-4 rounded-md hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-linkedin focus:ring-offset-2">
                            Usar nesta sessão
                        </button>
                    </div>
                </form>
                
                <div id="credentials-status" class="mt-4">
                    <!-- Status será atualizado via HTMX -->
                </div>
            </div>

            <!-- Card 2: Queries -->
            <div class="bg-white rounded-lg shadow-md p-6">
                <h2 class="text-lg font-semibold text-gray-900 mb-4">📝 Queries (.txt)</h2>
                
                <!-- Upload de arquivo -->
                <form hx-post="/upload/queries" hx-encoding="multipart/form-data" hx-target="#queries-status" hx-swap="innerHTML" class="mb-4">
                    <div class="space-y-4">
                        <div>
                            <label class="block text-sm font-medium text-gray-700">Arquivo .txt</label>
                            <input type="file" name="queries_file" accept=".txt" required
                                   class="mt-1 block w-full text-sm text-gray-500 file:mr-4 file:py-2 file:px-4 file:rounded-md file:border-0 file:text-sm file:font-semibold file:bg-linkedin file:text-white hover:file:bg-blue-700">
                        </div>
                        <button type="submit" 
                                class="w-full bg-green-600 text-white py-2 px-4 rounded-md hover:bg-green-700 focus:outline-none focus:ring-2 focus:ring-green-500 focus:ring-offset-2">
                            Enviar arquivo
                        </button>
                    </div>
                </form>

                <!-- Ou colar texto -->
                <div class="border-t pt-4">
                    <label class="block text-sm font-medium text-gray-700 mb-2">Ou cole as queries (uma por linha)</label>
                    <form hx-post="/upload/queries-text" hx-target="#queries-status" hx-swap="innerHTML">
                        <textarea name="queries_text" rows="4" placeholder="grupo boticário vendas&#10;startup tecnologia&#10;consultor financeiro"
                                  class="block w-full rounded-md border-gray-300 shadow-sm focus:border-linkedin focus:ring-linkedin"></textarea>
                        <button type="submit" 
                                class="mt-2 w-full bg-blue-600 text-white py-2 px-4 rounded-md hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2">
                            Usar texto
                        </button>
                    </form>
                </div>
                
                <div id="queries-status" class="mt-4">
                    <!-- Status será atualizado via HTMX -->
                </div>
            </div>

            <!-- Card 3: Execução -->
            <div class="bg-white rounded-lg shadow-md p-6">
                <h2 class="text-lg font-semibold text-gray-900 mb-4">🚀 Execução</h2>
                
                <form hx-post="/run" hx-target="#execution-status" hx-swap="innerHTML">
                    <div class="space-y-4">
                        <div>
                            <label class="block text-sm font-medium text-gray-700">Max Cards por página</label>
                            <input type="number" name="max_cards" value="60" min="1" max="100"
                                   class="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-linkedin focus:ring-linkedin">
                        </div>
                        <div>
                            <label class="block text-sm font-medium text-gray-700">Max Convites por página</label>
                            <input type="number" name="max_connects" value="10" min="1" max="10"
                                   class="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-linkedin focus:ring-linkedin">
                        </div>
                        
                        <!-- Opção de modo headless -->
                        <div class="flex items-center">
                            <input type="checkbox" name="headless_mode" id="headless_mode" 
                                   class="h-4 w-4 text-linkedin focus:ring-linkedin border-gray-300 rounded">
                            <label for="headless_mode" class="ml-2 block text-sm text-gray-700">
                                Modo Headless (navegador oculto)
                            </label>
                        </div>
                        
                        <button type="submit" id="start-crawler"
                                class="w-full bg-red-600 text-white py-2 px-4 rounded-md hover:bg-red-700 focus:outline-none focus:ring-2 focus:ring-red-500 focus:ring-offset-2 disabled:opacity-50 disabled:cursor-not-allowed">
                            Iniciar Crawler
                        </button>
                    </div>
                </form>
                
                <div id="execution-status" class="mt-4">
                    <!-- Status será atualizado via HTMX -->
                </div>
            </div>
        </div>

        <!-- Área de Status ao Vivo -->
        <div class="mt-8 bg-white rounded-lg shadow-md p-6">
            <h2 class="text-lg font-semibold text-gray-900 mb-4">📊 Status ao Vivo</h2>
            
            <div class="grid grid-cols-1 md:grid-cols-3 gap-6 mb-6">
                <!-- Contatos capturados (sessão) -->
                <div class="text-center">
                    <div class="text-2xl font-bold text-blue-600" id="captured-count">0</div>
                    <div class="text-sm text-gray-600">Contatos capturados</div>
                </div>
                
                <!-- Convites enviados (semana) -->
                <div class="text-center">
                    <div class="text-2xl font-bold text-green-600" id="invites-week">0</div>
                    <div class="text-sm text-gray-600">Convites esta semana</div>
                </div>
                
                <!-- Limite semanal -->
                <div class="text-center">
                    <div class="text-2xl font-bold text-gray-600">200</div>
                    <div class="text-sm text-gray-600">Limite semanal</div>
                </div>
            </div>

            <!-- Barra de progresso -->
            <div class="mb-6">
                <div class="flex justify-between text-sm text-gray-600 mb-2">
                    <span>Progresso semanal</span>
                    <span id="progress-text">0 / 200</span>
                </div>
                <div class="w-full bg-gray-200 rounded-full h-2.5">
                    <div id="progress-bar" class="bg-green-600 h-2.5 rounded-full transition-all duration-300" style="width: 0%"></div>
                </div>
            </div>

            <!-- Log em tempo real -->
            <div class="bg-gray-900 text-green-400 p-4 rounded-lg font-mono text-sm h-64 overflow-y-auto" id="live-log">
                <div class="text-gray-500">Aguardando logs...</div>
            </div>
        </div>

        <!-- Tabela de Convites -->
        <div class="mt-8 bg-white rounded-lg shadow-md p-6">
            <div class="flex justify-between items-center mb-4">
                <h2 class="text-lg font-semibold text-gray-900">📋 Convites Enviados</h2>
                <a href="/export/invites.csv" 
                   class="bg-linkedin text-white py-2 px-4 rounded-md hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-linkedin focus:ring-offset-2">
                    Exportar CSV
                </a>
            </div>
            
            <div id="invites-table" hx-get="/invites" hx-trigger="load">
                <!-- Tabela será carregada via HTMX -->
            </div>
        </div>
    </div>

    <!-- SSE para atualizações em tempo real -->
    <script>
        // Conectar ao SSE
        const eventSource = new EventSource('/events');
        
        eventSource.onopen = function(event) {
            console.log('✅ Conexão SSE estabelecida');
            addLogLine('🔗 Conectado ao servidor - aguardando logs...');
        };
        
        eventSource.onmessage = function(event) {
            console.log('Evento SSE recebido:', event.data); // Debug
            try {
                const data = JSON.parse(event.data);
                console.log('Dados parseados:', data); // Debug
                
                switch(data.type) {
                    case 'metrics':
                        updateMetrics(data.data);
                        break;
                    case 'invite':
                        addInviteToTable(data.data);
                        break;
                    case 'log':
                        addLogLine(data.data.line);
                        break;
                    case 'error':
                        addLogLine('❌ ' + data.data.message);
                        break;
                    default:
                        console.log('Tipo de evento desconhecido:', data.type);
                }
            } catch (error) {
                console.error('Erro ao processar evento SSE:', error);
                addLogLine('❌ Erro ao processar evento: ' + error.message);
            }
        };

        eventSource.onerror = function(event) {
            console.error('❌ Erro na conexão SSE:', event);
            addLogLine('❌ Erro na conexão com o servidor');
        };

        // Atualizar métricas
        function updateMetrics(data) {
            document.getElementById('captured-count').textContent = data.captured_session;
            document.getElementById('invites-week').textContent = data.invites_week;
            
            const progressBar = document.getElementById('progress-bar');
            const progressText = document.getElementById('progress-text');
            const percentage = (data.invites_week / data.invites_limit) * 100;
            
            progressBar.style.width = percentage + '%';
            progressText.textContent = data.invites_week + ' / ' + data.invites_limit;
            
            // Mudar cor baseado no limite
            if (percentage >= 90) {
                progressBar.className = 'bg-red-600 h-2.5 rounded-full transition-all duration-300';
            } else if (percentage >= 75) {
                progressBar.className = 'bg-yellow-600 h-2.5 rounded-full transition-all duration-300';
            } else {
                progressBar.className = 'bg-green-600 h-2.5 rounded-full transition-all duration-300';
            }
            
            // Desabilitar botão se limite atingido
            const startButton = document.getElementById('start-crawler');
            if (data.invites_week >= data.invites_limit) {
                startButton.disabled = true;
                startButton.textContent = 'Limite semanal atingido';
            } else {
                startButton.disabled = false;
                startButton.textContent = 'Iniciar Crawler';
            }
        }

        // Adicionar linha de log
        function addLogLine(line) {
            console.log('Log recebido:', line); // Debug
            const logContainer = document.getElementById('live-log');
            if (!logContainer) {
                console.error('Container de log não encontrado');
                return;
            }
            
            const logLine = document.createElement('div');
            logLine.textContent = '[' + new Date().toLocaleTimeString() + '] ' + line;
            logContainer.appendChild(logLine);
            
            // Auto-scroll
            logContainer.scrollTop = logContainer.scrollHeight;
            
            // Limitar número de linhas para performance
            while (logContainer.children.length > 100) {
                logContainer.removeChild(logContainer.firstChild);
            }
        }

        // Adicionar convite à tabela
        function addInviteToTable(data) {
            // Recarregar tabela via HTMX
            htmx.trigger('#invites-table', 'refresh');
            // Também forçar atualização manual
            setTimeout(() => {
                htmx.ajax('GET', '/invites', {target: '#invites-table'});
            }, 500);
        }

        // Carregar métricas iniciais
        function loadInitialMetrics() {
            fetch('/metrics')
                .then(response => response.json())
                .then(data => {
                    console.log('Métricas iniciais carregadas:', data);
                    updateMetrics(data);
                })
                .catch(error => {
                    console.error('Erro ao carregar métricas iniciais:', error);
                });
        }

        // Carregar métricas periodicamente (a cada 10 segundos)
        function startMetricsRefresh() {
            setInterval(() => {
                fetch('/metrics')
                    .then(response => response.json())
                    .then(data => {
                        updateMetrics(data);
                    })
                    .catch(error => {
                        console.error('Erro ao atualizar métricas:', error);
                    });
            }, 10000); // 10 segundos
        }

        // Verificar status inicial
        document.addEventListener('DOMContentLoaded', function() {
            // Carregar métricas iniciais
            loadInitialMetrics();
            
            // Iniciar refresh automático das métricas
            startMetricsRefresh();
            
            // Verificar se há credenciais e queries configuradas
            htmx.trigger('#credentials-status', 'check-status');
            htmx.trigger('#queries-status', 'check-status');
            
            // Atualizar tabela automaticamente a cada 5 segundos
            setInterval(() => {
                htmx.ajax('GET', '/invites', {target: '#invites-table'});
            }, 5000);
        });
    </script>
</body>
</html>`

// Template da tabela de convites
const invitesTemplate = `{{if .Invites}}
<div class="overflow-x-auto">
    <table class="min-w-full divide-y divide-gray-200">
        <thead class="bg-gray-50">
            <tr>
                <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Data/Hora</th>
                <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Usuário</th>
                <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Nome</th>
                <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Cargo</th>
                <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Empresa</th>
                <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Localização</th>
                <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Query</th>
            </tr>
        </thead>
        <tbody class="bg-white divide-y divide-gray-200">
            {{range .Invites}}
            <tr>
                <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-900">{{.Timestamp.Format "02/01/2006 15:04"}}</td>
                <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-900">{{.UserEmail}}</td>
                <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-900">{{.ProfileName}}</td>
                <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-900">{{.ProfileTitle}}</td>
                <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-900">{{.Company}}</td>
                <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-900">{{.Location}}</td>
                <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-900">{{.Query}}</td>
            </tr>
            {{end}}
        </tbody>
    </table>
</div>

<!-- Paginação -->
{{if gt .TotalPages 1}}
<div class="mt-4 flex justify-between items-center">
    		<div class="text-sm text-gray-700">
			Mostrando {{.Page}} a {{.PageSize}} de {{.Total}} resultados
		</div>
		<div class="flex space-x-2">
			{{if gt .Page 0}}
			<button hx-get="/invites?page={{.Page}}" hx-target="#invites-table" 
					class="px-3 py-2 border border-gray-300 rounded-md text-sm font-medium text-gray-700 bg-white hover:bg-gray-50">
				Anterior
			</button>
			{{end}}
			{{if lt .Page .TotalPages}}
			<button hx-get="/invites?page={{.Page}}" hx-target="#invites-table" 
					class="px-3 py-2 border border-gray-300 rounded-md text-sm font-medium text-gray-700 bg-white hover:bg-gray-50">
				Próxima
			</button>
			{{end}}
		</div>
</div>
{{end}}

{{else}}
<div class="text-center py-8 text-gray-500">
    <p>Nenhum convite enviado ainda.</p>
    <p class="text-sm">Execute o crawler para começar a capturar perfis e enviar convites.</p>
</div>
{{end}}`

// Partial da tabela de convites
const invitesTablePartial = `{{template "invites-table" .}}`

// Partial da barra de progresso
const progressBarPartial = `{{template "progress-bar" .}}`
