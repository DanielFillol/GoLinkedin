package crawler

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
)

type Scraper struct {
	ctx    context.Context
	cancel context.CancelFunc
}

func NewScraper() (*Scraper, error) {
	// Criar contexto com logging suprimido
	ctx, cancel := chromedp.NewContext(context.Background())

	// Configurar chromedp com opções estáveis e suprimir erros de DevTools
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", false),
		chromedp.Flag("disable-blink-features", "AutomationControlled"),
		chromedp.Flag("disable-web-security", true),
		chromedp.Flag("disable-features", "VizDisplayCompositor"),
		chromedp.Flag("disable-logging", true),
		chromedp.Flag("log-level", "0"),
		chromedp.Flag("silent", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("disable-background-timer-throttling", true),
		chromedp.Flag("disable-backgrounding-occluded-windows", true),
		chromedp.Flag("disable-renderer-backgrounding", true),
		chromedp.Flag("disable-field-trial-config", true),
		chromedp.Flag("disable-ipc-flooding-protection", true),
		chromedp.UserAgent("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"),
	)

	allocCtx, cancel := chromedp.NewExecAllocator(ctx, opts...)
	ctx, _ = chromedp.NewContext(allocCtx)

	return &Scraper{
		ctx:    ctx,
		cancel: cancel,
	}, nil
}

func (s *Scraper) Close() {
	s.cancel()
}

func (s *Scraper) Login(email, pass string) error {
	log.Println("Fazendo login no LinkedIn...")

	// Navegar para página de login
	err := chromedp.Run(s.ctx, chromedp.Navigate("https://www.linkedin.com/login"))
	if err != nil {
		return fmt.Errorf("erro ao navegar para login: %v", err)
	}

	// Aguardar campos de login
	err = chromedp.Run(s.ctx, chromedp.WaitVisible(`input[name="session_key"]`))
	if err != nil {
		return fmt.Errorf("campo email não encontrado: %v", err)
	}

	// Preencher credenciais
	err = chromedp.Run(s.ctx,
		chromedp.SendKeys(`input[name="session_key"]`, email),
		chromedp.SendKeys(`input[name="session_password"]`, pass),
		chromedp.Click(`button[type="submit"]`),
	)
	if err != nil {
		return fmt.Errorf("erro ao preencher credenciais: %v", err)
	}

	// Aguardar redirecionamento
	time.Sleep(3 * time.Second)

	log.Println("Login realizado com sucesso")
	return nil
}

func (s *Scraper) OpenSearch(query string) error {
	log.Printf("Abrindo busca: %s", query)

	// Construir URL de busca
	searchURL := fmt.Sprintf("https://www.linkedin.com/search/results/people/?keywords=%s&origin=CLUSTER_EXPANSION", query)

	err := chromedp.Run(s.ctx, chromedp.Navigate(searchURL))
	if err != nil {
		return fmt.Errorf("erro ao navegar para busca: %v", err)
	}

	// Aguardar carregamento da página
	err = chromedp.Run(s.ctx, chromedp.WaitReady("main"))
	if err != nil {
		return fmt.Errorf("página não carregou: %v", err)
	}

	// Fazer 2 scrolls leves para destravar lazy-load
	time.Sleep(1 * time.Second)
	for i := 0; i < 2; i++ {
		err = chromedp.Run(s.ctx, chromedp.Evaluate(`window.scrollBy(0, 300)`, nil))
		if err != nil {
			log.Printf("Aviso: scroll %d falhou: %v", i+1, err)
		}
		time.Sleep(100 * time.Millisecond)
	}

	log.Printf("Busca aberta com sucesso")
	return nil
}

func (s *Scraper) pollProfilesCount() (int, error) {
	var count int

	// Tentar contar perfis usando seletores estáveis
	js := fmt.Sprintf(`
		(() => {
			// Primeiro: contar anchors /in/ dentro de SelCardNew
			const cards = document.querySelectorAll('%s');
			let count = 0;
			cards.forEach(card => {
				const anchors = card.querySelectorAll('a[href*="/in/"]');
				count += anchors.length;
			});
			
			// Fallback: contar todos os anchors /in/ na página
			if (count === 0) {
				const allAnchors = document.querySelectorAll('a[href*="/in/"]');
				count = allAnchors.length;
			}
			
			return count;
		})()
	`, SelCardNew)

	err := chromedp.Run(s.ctx, chromedp.Evaluate(js, &count))
	if err != nil {
		return 0, fmt.Errorf("erro ao contar perfis: %v", err)
	}

	return count, nil
}

func (s *Scraper) CaptureVisibleAndConnect(maxConnects int) ([]Contact, int) {
	var contacts []Contact
	connectsSent := 0

	log.Printf("Iniciando captura de perfis visíveis...")

	// Polling por perfis com scrolls periódicos
	for poll := 0; poll < 10; poll++ {
		count, err := s.pollProfilesCount()
		if err != nil {
			log.Printf("Erro ao contar perfis (poll %d): %v", poll+1, err)
			continue
		}

		if count > 0 {
			log.Printf("Encontrados %d perfis visíveis", count)
			break
		}

		// Fazer scroll leve a cada 3-4 tentativas
		if poll%3 == 0 {
			err = chromedp.Run(s.ctx, chromedp.Evaluate(`window.scrollBy(0, 200)`, nil))
			if err != nil {
				log.Printf("Aviso: scroll de polling falhou: %v", err)
			}
			time.Sleep(100 * time.Millisecond)
		}

		time.Sleep(500 * time.Millisecond)
	}

	// Capturar perfis visíveis
	js := fmt.Sprintf(`
		(() => {
			const contacts = [];
			const cards = document.querySelectorAll('%s');
			const maxCards = %d;
			
			for (let i = 0; i < Math.min(cards.length, maxCards); i++) {
				const card = cards[i];
				
				// Injetar data-sel para identificação
				card.setAttribute('data-sel', 'card-' + i);
				
				// Buscar link do perfil
				const profileLink = card.querySelector('a[href*="/in/"]');
				if (!profileLink) continue;
				
				// Limpar nome removendo "Ver perfil de" e outros textos
				let name = profileLink.innerText.split('\\n')[0].trim();
				name = name.replace(/Ver perfil de\\s*/i, '').replace(/\\s*Ver perfil.*$/i, '').trim();
				const linkedin = profileLink.href;
				
				// Buscar elementos específicos do LinkedIn
				let title = '';
				let company = '';
				let location = '';
				
				// Buscar título e empresa usando seletores mais específicos
				const titleElements = card.querySelectorAll('span, div, p');
				const relevantTexts = [];
				
				for (const elem of titleElements) {
					const text = elem.textContent.trim();
					if (text && text !== name && text.length > 5 && text.length < 100) {
						// Filtrar status, botões e texto irrelevante
						if (!text.includes('•') && 
							!text.includes('status') && 
							!text.includes('off-line') &&
							!text.includes('online') &&
							!text.includes('Conectar') &&
							!text.includes('Connect') &&
							!text.includes('Mensagem') &&
							!text.includes('Message') &&
							!text.includes('Seguir') &&
							!text.includes('Follow') &&
							!text.includes('Ver perfil') &&
							!text.includes('Conexão de') &&
							!text.includes('Connection') &&
							!text.match(/^\\d+\\s*\\+?\\s*conexões?$/i) &&
							!text.match(/^\\d+\\s*\\+?\\s*connections?$/i)) {
							
							relevantTexts.push(text);
						}
					}
				}
				
				// Separar título e empresa baseado em heurísticas
				if (relevantTexts.length > 0) {
					// Primeiro texto relevante como título
					title = relevantTexts[0];
					
											// Tentar separar título e empresa se contiver "|" ou "-"
		if (title.includes('|') || title.includes('-')) {
			const parts = title.split(/[|-]/).map(p => p.trim()).filter(p => p.length > 0);
			if (parts.length >= 2) {
				// Primeira parte como título, resto como empresa
				title = parts[0];
				company = parts.slice(1).join(' - ');
			}
		}
		
		// Se ainda não tem empresa, tentar extrair de outros elementos
		if (!company) {
			// Buscar elementos que possam conter empresa
			const companyElements = card.querySelectorAll('span[class*="entity-result__primary-subtitle"], span[class*="search-result__info"], div[class*="search-result__info"]');
			for (const elem of companyElements) {
				const text = elem.textContent.trim();
				if (text && text !== name && text !== title && text.length > 3 && text.length < 100) {
					// Filtrar texto irrelevante
					if (!text.includes('•') && 
						!text.includes('status') && 
						!text.includes('off-line') &&
						!text.includes('online') &&
						!text.includes('Conectar') &&
						!text.includes('Connect') &&
						!text.includes('Ver perfil') &&
						!text.includes('Mensagem') &&
						!text.includes('Message') &&
						!text.includes('Seguir') &&
						!text.includes('Follow') &&
						!text.match(/^\d+\s*\+?\s*conexões?$/i) &&
						!text.match(/^\d+\s*\+?\s*seguidores?$/i)) {
						company = text;
						break;
					}
				}
			}
		}
		
		// Se ainda não tem empresa, tentar extrair do título combinado
		if (!company && title) {
			// Se o título contém separadores, tentar separar
			if (title.includes('|') || title.includes('-')) {
				const titleParts = title.split(/[|-]/).map(p => p.trim()).filter(p => p.length > 0);
				if (titleParts.length >= 2) {
					// Primeira parte como título, resto como empresa
					title = titleParts[0];
					company = titleParts.slice(1).join(' - ');
				}
			}
			// Se o título contém palavras-chave de empresa, tentar separar
			else if (title.includes('Grupo') || title.includes('Boticário') || title.includes('Company') || title.includes('Corp')) {
				// Tentar separar por espaços ou outros padrões
				const words = title.split(' ');
				if (words.length >= 3) {
					// Primeiras palavras como título, resto como empresa
					const titleWords = words.slice(0, Math.floor(words.length / 2));
					const companyWords = words.slice(Math.floor(words.length / 2));
					title = titleWords.join(' ');
					company = companyWords.join(' ');
				}
			}
		}
		
		// Se ainda não tem empresa, tentar extrair de outros elementos
		if (!company) {
			// Buscar elementos que possam conter empresa
			const companyElements = card.querySelectorAll('span[class*="entity-result__primary-subtitle"], span[class*="search-result__info"], div[class*="search-result__info"]');
			for (const elem of companyElements) {
				const text = elem.textContent.trim();
				if (text && text !== name && text !== title && text.length > 3 && text.length < 100) {
					// Filtrar texto irrelevante
					if (!text.includes('•') && 
						!text.includes('status') && 
						!text.includes('off-line') &&
						!text.includes('online') &&
						!text.includes('Conectar') &&
						!text.includes('Connect') &&
						!text.includes('Ver perfil') &&
						!text.includes('Mensagem') &&
						!text.includes('Message') &&
						!text.includes('Seguir') &&
						!text.includes('Follow') &&
						!text.match(/^\d+\s*\+?\s*conexões?$/i) &&
						!text.match(/^\d+\s*\+?\s*seguidores?$/i)) {
						company = text;
						break;
					}
				}
			}
		}
				
				// Se ainda não conseguiu separar, tentar heurística mais inteligente
				if (!company && title) {
					const titleLower = title.toLowerCase();
					// Se o título contém palavras-chave de empresa, mover para empresa
					if (titleLower.includes('grupo') || titleLower.includes('boticário') || 
						titleLower.includes('company') || titleLower.includes('corp') ||
						titleLower.includes('ltda') || titleLower.includes('s.a.')) {
						company = title;
						title = '';
					}
				}
					
					// Se não conseguiu separar, usar segundo texto relevante como empresa
					if (!company && relevantTexts.length > 1 && relevantTexts[1] !== title) {
						company = relevantTexts[1];
					}
				}
				
				// Buscar localização (geralmente no final do card)
				const locationElements = card.querySelectorAll('span, div');
				for (const elem of locationElements) {
					const text = elem.textContent.trim();
					if (text && /%s/i.test(text) && text.length < 50) {
						// Filtrar status messages e texto irrelevante
						if (!text.includes('status') && 
							!text.includes('off-line') && 
							!text.includes('online') &&
							!text.includes('O status está') &&
							!text.includes('Ver perfil') &&
							!text.includes('Conectar') &&
							!text.includes('Connect')) {
							location = text;
							break;
						}
					}
				}
				
				// Fallback: se não encontrou título, usar primeira linha relevante
				if (!title) {
					const allText = card.textContent.split('\\n').map(t => t.trim()).filter(t => 
						t && t !== name && t.length > 3 && t.length < 80 &&
						!t.includes('•') && 
						!t.includes('status') && 
						!t.includes('off-line') &&
						!t.includes('online') &&
						!t.includes('Conectar') &&
						!t.includes('Connect') &&
						!t.includes('Ver perfil') &&
						!t.includes('Mensagem') &&
						!t.includes('Message') &&
						!t.includes('Conexão de') &&
						!t.includes('Connection')
					);
					if (allText.length > 0) {
						title = allText[0];
					}
				}
				
				// Fallback: se não encontrou empresa, usar segunda linha relevante
				if (!company && title) {
					const allText = card.textContent.split('\\n').map(t => t.trim()).filter(t => 
						t && t !== name && t !== title && t.length > 3 && t.length < 80 &&
						!t.includes('•') && 
						!t.includes('status') && 
						!t.includes('off-line') &&
						!t.includes('online') &&
						!t.includes('Conectar') &&
						!t.includes('Connect') &&
						!t.includes('Ver perfil') &&
						!t.includes('Mensagem') &&
						!t.includes('Message') &&
						!t.includes('Conexão de') &&
						!t.includes('Connection')
					);
					if (allText.length > 0) {
						company = allText[0];
					}
				}
				
				// Fallback para localização: buscar por padrões específicos de localização
				if (!location) {
					const allText = card.textContent.split('\\n').map(t => t.trim()).filter(t => 
						t && t !== name && t !== title && t !== company && t.length > 3 && t.length < 50 &&
						/%s/i.test(t) &&
						!t.includes('status') && 
						!t.includes('off-line') && 
						!t.includes('online') &&
						!t.includes('O status está') &&
						!t.includes('Ver perfil') &&
						!t.includes('Conectar') &&
						!t.includes('Connect')
					);
					if (allText.length > 0) {
						location = allText[0];
					}
				}
				
				contacts.push({
					name: name,
					title: title,
					company: company,
					location: location,
					linkedin: linkedin
				});
			}
			
			return contacts;
		})()
			`, SelCardNew, 60, RxLocation)

	var rawContacts []map[string]interface{}
	err := chromedp.Run(s.ctx, chromedp.Evaluate(js, &rawContacts))
	if err != nil {
		log.Printf("Erro ao capturar perfis: %v", err)
		return contacts, connectsSent
	}

	// Converter para struct Contact
	for _, raw := range rawContacts {
		contact := Contact{
			Name:     getString(raw, "name"),
			Title:    getString(raw, "title"),
			Company:  getString(raw, "company"),
			Location: getString(raw, "location"),
			LinkedIn: NormalizeProfileURL(getString(raw, "linkedin")),
		}
		contacts = append(contacts, contact)
	}

	log.Printf("Capturados %d perfis visíveis", len(contacts))

	// Tentar conectar nos cards (até maxConnects)
	for i, contact := range contacts {
		if connectsSent >= maxConnects {
			break
		}

		log.Printf("Tentando conectar com %s (%s)", contact.Name, contact.Title)

		// Buscar botão Conectar dentro do card
		js := fmt.Sprintf(`
			(() => {
				const card = document.querySelector('[data-sel="card-%d"]');
				if (!card) return false;
				
				const buttons = card.querySelectorAll('%s');
				for (const btn of buttons) {
					const text = btn.innerText.toLowerCase();
					if (/%s/i.test(text)) {
						btn.click();
						return true;
					}
				}
				return false;
			})()
		`, i, SelButtonsInside, strings.Join(RxConnectLabels, "|"))

		var clicked bool
		err := chromedp.Run(s.ctx, chromedp.Evaluate(js, &clicked))
		if err != nil {
			log.Printf("Erro ao clicar em Conectar: %v", err)
			continue
		}

		if clicked {
			// Aguardar modal e tentar clicar em Enviar
			time.Sleep(1 * time.Second)

			js := `
				(() => {
					const buttons = document.querySelectorAll('button, a');
					for (const btn of buttons) {
						const text = btn.innerText.toLowerCase();
						if (/enviar|send|enviar agora/i.test(text)) {
							btn.click();
							return true;
						}
					}
					return false;
				})()
			`

			var sent bool
			err := chromedp.Run(s.ctx, chromedp.Evaluate(js, &sent))
			if err != nil {
				log.Printf("Erro ao clicar em Enviar: %v", err)
			}

			if sent {
				connectsSent++
				log.Printf("Convite enviado para %s", contact.Name)
			}
		}

		// Jitter entre convites
		time.Sleep(time.Duration(500+time.Now().UnixNano()%1000) * time.Millisecond)
	}

	log.Printf("Convites enviados: %d", connectsSent)
	return contacts, connectsSent
}

func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func jsString(s string) string {
	// Escape básico para injeção JS
	return strings.ReplaceAll(s, `"`, `\"`)
}
