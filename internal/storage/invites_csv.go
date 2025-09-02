package storage

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/your-org/linkedin-visible-crawler/internal/crawler"
)

// InviteStorage gerencia o armazenamento de convites em CSV
type InviteStorage struct {
	filePath string
	writer   *csv.Writer
	file     *os.File
}

// NewInviteStorage cria nova instância do storage
func NewInviteStorage() *InviteStorage {
	// Criar diretório data se não existir
	dataDir := "data"
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		panic(fmt.Sprintf("Erro ao criar diretório data: %v", err))
	}

	filePath := filepath.Join(dataDir, "invites.csv")
	return &InviteStorage{
		filePath: filePath,
	}
}

// AppendInvite adiciona um novo convite ao CSV
func (s *InviteStorage) AppendInvite(record crawler.InviteRecord) error {
	// Abrir arquivo para append (criar se não existir)
	file, err := os.OpenFile(s.filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("erro ao abrir arquivo CSV: %v", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Se arquivo está vazio, escrever cabeçalho
	fileInfo, err := file.Stat()
	if err != nil {
		return fmt.Errorf("erro ao obter info do arquivo: %v", err)
	}

	if fileInfo.Size() == 0 {
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
		if err := writer.Write(header); err != nil {
			return fmt.Errorf("erro ao escrever cabeçalho: %v", err)
		}
	}

	// Escrever registro
	row := []string{
		record.Timestamp.Format(time.RFC3339),
		record.UserEmail,
		record.ProfileName,
		record.ProfileTitle,
		record.Company,
		record.Location,
		record.LinkedInURL,
		record.Query,
	}

	if err := writer.Write(row); err != nil {
		return fmt.Errorf("erro ao escrever registro: %v", err)
	}

	return nil
}

// ListInvites lista convites com paginação
func (s *InviteStorage) ListInvites(page, size int) ([]crawler.InviteRecord, int, error) {
	// Abrir arquivo para leitura
	file, err := os.Open(s.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return []crawler.InviteRecord{}, 0, nil
		}
		return nil, 0, fmt.Errorf("erro ao abrir arquivo CSV: %v", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, 0, fmt.Errorf("erro ao ler CSV: %v", err)
	}

	// Pular cabeçalho
	if len(records) == 0 {
		return []crawler.InviteRecord{}, 0, nil
	}

	// Converter registros
	var invites []crawler.InviteRecord
	for _, record := range records[1:] { // Pular cabeçalho
		if len(record) < 8 {
			continue // Pular linhas inválidas
		}

		timestamp, err := time.Parse(time.RFC3339, record[0])
		if err != nil {
			continue // Pular linhas com timestamp inválido
		}

		invite := crawler.InviteRecord{
			Timestamp:    timestamp,
			UserEmail:    record[1],
			ProfileName:  record[2],
			ProfileTitle: record[3],
			Company:      record[4],
			Location:     record[5],
			LinkedInURL:  record[6],
			Query:        record[7],
		}

		invites = append(invites, invite)
	}

	total := len(invites)

	// Aplicar paginação
	start := page * size
	end := start + size

	if start >= total {
		return []crawler.InviteRecord{}, total, nil
	}

	if end > total {
		end = total
	}

	return invites[start:end], total, nil
}

// GetTotalCount retorna o total de convites
func (s *InviteStorage) GetTotalCount() (int, error) {
	file, err := os.Open(s.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, fmt.Errorf("erro ao abrir arquivo CSV: %v", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return 0, fmt.Errorf("erro ao ler CSV: %v", err)
	}

	// Retornar total (menos cabeçalho)
	return len(records) - 1, nil
}
