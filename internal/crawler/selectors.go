package crawler

var (
	// seletores estáveis na busca
	SelCardNew     = `div[data-view-name="search-entity-result-universal-template"]` // wrapper do card
	SelAnyProfileA = `a[href*="/in/"]`                                               // link de perfil

	// botões
	SelButtonsInside = `button, a`

	// padrões de texto (sem PCRE, serão usados com flag /i no JavaScript)
	RxConnectLabels = []string{`^conectar$`, `^connect$`}
	RxSendLabels    = []string{`^enviar$`, `^send$`, `^enviar agora$`}

	// heurística de localização (UF/BR)
	RxLocation = `,\s*[A-Z]{2}\b|Brasil|Brazil|SP|RJ|CE|PE|PR|SC|RS|MG|BA|DF|GO|ES|AM|PA`
)
