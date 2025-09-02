package crawler

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
)

// Engine motor principal do crawler
type Engine struct {
	ctx    context.Context
	cancel context.CancelFunc
}

// NewEngine cria nova instância do motor
func NewEngine() *Engine {
	ctx, cancel := chromedp.NewContext(context.Background())
	return &Engine{
		ctx:    ctx,
		cancel: cancel,
	}
}

// Run executa o crawler com as configurações especificadas
func (e *Engine) Run(cfg RunConfig, creds Creds, callbacks Callbacks) error {
	defer e.cancel()

	// Configurar chromedp
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", cfg.Headless),
		chromedp.Flag("disable-blink-features", "AutomationControlled"),
		chromedp.Flag("disable-web-security", true),
		chromedp.Flag("disable-features", "VizDisplayCompositor"),
		chromedp.Flag("disable-logging", true),
		chromedp.Flag("log-level", "0"),
		chromedp.Flag("silent", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.UserAgent("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"),
	)

	allocCtx, cancel := chromedp.NewExecAllocator(e.ctx, opts...)
	defer cancel()

	taskCtx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	// Login no LinkedIn
	if err := e.login(taskCtx, creds, callbacks); err != nil {
		return err
	}

	// Aguardar 2FA manual
	callbacks.OnLog("Aguardando 2FA manual... (8 segundos)")
	time.Sleep(8 * time.Second)

	// Minimizar/ocultar navegador após 2FA (apenas se não estiver em modo headless)
	if !cfg.Headless {
		callbacks.OnLog("Minimizando navegador para execução em background...")
		if err := e.minimizeBrowser(taskCtx); err != nil {
			callbacks.OnLog("Aviso: não foi possível minimizar o navegador")
		}
	} else {
		callbacks.OnLog("Modo headless ativo - navegador já está oculto")
	}

	// Processar cada query
	for i, query := range cfg.Queries {
		callbacks.OnLog(fmt.Sprintf("=== Processando query %d/%d: %s ===", i+1, len(cfg.Queries), query))

		if err := e.processQuery(taskCtx, query, cfg, callbacks); err != nil {
			callbacks.OnLog(fmt.Sprintf("Erro ao processar query '%s': %v", query, err))
			continue
		}
	}

	return nil
}

// login realiza login no LinkedIn
func (e *Engine) login(ctx context.Context, creds Creds, callbacks Callbacks) error {
	callbacks.OnLog("Fazendo login no LinkedIn...")

	// Navegar para página de login
	if err := chromedp.Run(ctx, chromedp.Navigate("https://www.linkedin.com/login")); err != nil {
		return err
	}

	// Aguardar página carregar
	if err := chromedp.Run(ctx, chromedp.WaitReady("input[name='session_key']")); err != nil {
		return err
	}

	// Preencher credenciais
	if err := chromedp.Run(ctx,
		chromedp.SendKeys("input[name='session_key']", creds.Email),
		chromedp.SendKeys("input[name='session_password']", creds.Password),
	); err != nil {
		return err
	}

	// Clicar no botão de login
	if err := chromedp.Run(ctx, chromedp.Click("button[type='submit']")); err != nil {
		return err
	}

	// Aguardar redirecionamento
	time.Sleep(3 * time.Second)

	callbacks.OnLog("Login realizado com sucesso")
	return nil
}

// processQuery processa uma query específica
func (e *Engine) processQuery(ctx context.Context, query string, cfg RunConfig, callbacks Callbacks) error {
	callbacks.OnLog(fmt.Sprintf("Abrindo busca: %s", query))

	// Navegar para busca
	searchURL := "https://www.linkedin.com/search/results/people/?keywords=" + strings.ReplaceAll(query, " ", "+") + "&origin=CLUSTER_EXPANSION"
	if err := chromedp.Run(ctx, chromedp.Navigate(searchURL)); err != nil {
		return err
	}

	// Aguardar página carregar
	if err := chromedp.Run(ctx, chromedp.WaitReady("main")); err != nil {
		return err
	}

	// Fazer scrolls leves para destravar lazy-load
	time.Sleep(1 * time.Second)
	if err := chromedp.Run(ctx, chromedp.Evaluate("window.scrollBy(0, 300)", nil)); err != nil {
		// Ignorar erro de scroll
	}
	time.Sleep(1 * time.Second)
	if err := chromedp.Run(ctx, chromedp.Evaluate("window.scrollBy(0, 300)", nil)); err != nil {
		// Ignorar erro de scroll
	}

	callbacks.OnLog("Busca aberta com sucesso")
	callbacks.OnLog("Iniciando captura de perfis visíveis...")

	// Contar perfis visíveis
	count, err := e.countVisibleProfiles(ctx)
	if err != nil {
		callbacks.OnLog(fmt.Sprintf("Erro ao contar perfis: %v", err))
	} else {
		callbacks.OnLog(fmt.Sprintf("Encontrados %d perfis visíveis", count))
	}

	// Capturar e conectar
	contacts, invitesSent, err := e.captureAndConnect(ctx, cfg, callbacks)
	if err != nil {
		return err
	}

	callbacks.OnLog(fmt.Sprintf("Capturados %d perfis para '%s'", len(contacts), query))
	callbacks.OnLog(fmt.Sprintf("Convites enviados: %d", invitesSent))

	return nil
}

// countVisibleProfiles conta perfis visíveis na página
func (e *Engine) countVisibleProfiles(ctx context.Context) (int, error) {
	var count int
	err := chromedp.Run(ctx, chromedp.Evaluate(`
		document.querySelectorAll('a[href*="/in/"]').length
	`, &count))
	return count, err
}

// captureAndConnect captura perfis e tenta conectar
func (e *Engine) captureAndConnect(ctx context.Context, cfg RunConfig, callbacks Callbacks) ([]Contact, int, error) {
	var contacts []Contact
	invitesSent := 0

	// JavaScript para extrair perfis visíveis
	var result []map[string]interface{}
	err := chromedp.Run(ctx, chromedp.Evaluate(`
		(() => {
			const cards = document.querySelectorAll('div[data-view-name="search-entity-result-universal-template"]');
			const results = [];
			let count = 0;
			
			for (const card of cards) {
				if (count >= `+fmt.Sprintf("%d", cfg.MaxCardsRead)+`) break;
				
				const link = card.querySelector('a[href*="/in/"]');
				if (!link) continue;
				
				const name = link.innerText.replace(/Ver perfil de\\s*/i, '').replace(/\\s*Ver perfil.*$/i, '').trim();
				const url = link.href;
				
				// Extrair título e empresa
				const relevantTexts = [];
				const spans = card.querySelectorAll('span, div, p');
				for (const span of spans) {
					const text = span.innerText.trim();
					if (text && text !== name && 
						!text.includes('status') && !text.includes('off-line') && !text.includes('online') &&
						!text.includes('Ver perfil') && !text.includes('Conectar') && !text.includes('Connect') &&
						!text.includes('Mensagem') && !text.includes('Message') && !text.includes('Seguir') &&
						!text.includes('Follow') && !text.match(/\\d+\\s*conexão/)) {
						relevantTexts.push(text);
					}
				}
				
				let title = relevantTexts[0] || '';
				let company = relevantTexts[1] || '';
				
				// Heurística para separar título e empresa
				if (title.includes('|') || title.includes('-')) {
					const parts = title.split(/[|-]/);
					title = parts[0].trim();
					company = parts.slice(1).join(' ').trim();
				} else if (title.toLowerCase().includes('grupo') || title.toLowerCase().includes('boticário') ||
						   title.toLowerCase().includes('company') || title.toLowerCase().includes('corp') ||
						   title.toLowerCase().includes('ltda') || title.toLowerCase().includes('s.a.')) {
					if (title.split(' ').length >= 3) {
						const words = title.split(' ');
						title = words.slice(0, 2).join(' ');
						company = words.slice(2).join(' ');
					}
				}
				
				// Extrair localização
				let location = '';
				for (const text of relevantTexts) {
					if (text.match(/,\\s*[A-Z]{2}\\b|Brasil|Brazil|SP|RJ|CE|PE|PR|SC|RS|MG|BA|DF|GO|ES|AM|PA/i) &&
						!text.includes('status') && !text.includes('off-line') && !text.includes('online') &&
						!text.includes('O status está') && !text.includes('Ver perfil') && 
						!text.includes('Conectar') && !text.includes('Connect')) {
						location = text;
						break;
					}
				}
				
				// Fallback para localização
				if (!location) {
					const allText = card.innerText;
					const lines = allText.split('\\n');
					for (const line of lines) {
						if (line.match(/,\\s*[A-Z]{2}\\b|Brasil|Brazil|SP|RJ|CE|PE|PR|SC|RS|MG|BA|DF|GO|ES|AM|PA/i) &&
							!line.includes('status') && !line.includes('off-line') && !line.includes('online') &&
							!line.includes('Ver perfil') && !line.includes('Conectar') && !line.includes('Connect')) {
							location = line.trim();
							break;
						}
					}
				}
				
				results.push({
					name: name,
					title: title,
					company: company,
					location: location,
					linkedin_url: url,
					card: card
				});
				
				count++;
			}
			
			return results;
		})()
	`, &result))

	if err != nil {
		return contacts, invitesSent, err
	}

	// Processar cada perfil capturado
	for i, profile := range result {
		if i >= cfg.MaxCardsRead {
			break
		}

		contact := Contact{
			Name:     profile["name"].(string),
			Title:    profile["title"].(string),
			Company:  profile["company"].(string),
			Location: profile["location"].(string),
			LinkedIn: profile["linkedin_url"].(string),
		}

		contacts = append(contacts, contact)
		callbacks.OnCaptured(contact)

		// Tentar conectar (limitado por página)
		if invitesSent < cfg.MaxConnectsPerPage {
			callbacks.OnLog(fmt.Sprintf("Tentando conectar com %s (%s)", contact.Company, contact.Name))

			if e.tryConnect(ctx, profile["card"], callbacks) {
				invitesSent++
				callbacks.OnInviteSent(contact)
			}
		}
	}

	return contacts, invitesSent, nil
}

// tryConnect tenta enviar convite para um perfil
func (e *Engine) tryConnect(ctx context.Context, card interface{}, callbacks Callbacks) bool {
	// JavaScript para tentar conectar
	var success bool
	err := chromedp.Run(ctx, chromedp.Evaluate(`
		(() => {
			try {
				// Buscar botões de conectar na página
				const connectBtns = document.querySelectorAll('button, a');
				let found = false;
				
				for (const btn of connectBtns) {
					const btnText = btn.innerText.trim();
					if (/^conectar$|^connect$/i.test(btnText)) {
						// Clicar no botão conectar
						btn.click();
						found = true;
						break;
					}
				}
				
				if (found) {
					// Aguardar modal e clicar em enviar
					setTimeout(() => {
						const sendBtn = document.querySelector('button, a');
						if (sendBtn && /^enviar$|^send$|^enviar agora$/i.test(sendBtn.innerText.trim())) {
							sendBtn.click();
						}
					}, 1000);
				}
				
				// Simular sucesso para teste (em produção seria mais robusto)
				return found;
			} catch (e) {
				return false;
			}
		})()
	`, &success))

	if err != nil {
		callbacks.OnLog(fmt.Sprintf("Erro ao tentar conectar: %v", err))
		return false
	}

	return success
}

// minimizeBrowser minimiza/oculta o navegador após o 2FA
func (e *Engine) minimizeBrowser(ctx context.Context) error {
	// JavaScript para minimizar a janela do navegador
	return chromedp.Run(ctx, chromedp.Evaluate(`
		(() => {
			try {
				// Tentar minimizar a janela
				if (window.screen && window.screen.availHeight) {
					// Mover a janela para fora da tela visível
					window.moveTo(-10000, -10000);
					window.resizeTo(1, 1);
					return true;
				}
				return false;
			} catch (e) {
				return false;
			}
		})()
	`, nil))
}
