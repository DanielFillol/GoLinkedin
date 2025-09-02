package storage

import (
	"fmt"
	"time"
)

// WeeklyCounter gerencia contadores semanais de convites
type WeeklyCounter struct {
	storage *InviteStorage
}

// NewWeeklyCounter cria nova instância do contador
func NewWeeklyCounter(storage *InviteStorage) *WeeklyCounter {
	return &WeeklyCounter{
		storage: storage,
	}
}

// CountThisWeek conta convites da semana atual para um usuário
func (wc *WeeklyCounter) CountThisWeek(userEmail string) (int, error) {
	// Obter início da semana (segunda-feira)
	now := time.Now()
	weekStart := getWeekStart(now)
	weekEnd := weekStart.AddDate(0, 0, 7)

	// Listar todos os convites
	invites, _, err := wc.storage.ListInvites(0, 10000) // Buscar todos
	if err != nil {
		return 0, fmt.Errorf("erro ao listar convites: %v", err)
	}

	// Contar convites da semana
	count := 0
	for _, invite := range invites {
		if invite.UserEmail == userEmail &&
			invite.Timestamp.After(weekStart) &&
			invite.Timestamp.Before(weekEnd) {
			count++
		}
	}

	return count, nil
}

// CanSendInvite verifica se usuário pode enviar convite (limite 200/semana)
func (wc *WeeklyCounter) CanSendInvite(userEmail string) (bool, int, error) {
	count, err := wc.CountThisWeek(userEmail)
	if err != nil {
		return false, 0, err
	}

	return count < 200, count, nil
}

// getWeekStart retorna o início da semana (segunda-feira)
func getWeekStart(t time.Time) time.Time {
	weekday := int(t.Weekday())
	if weekday == 0 { // Domingo
		weekday = 7
	}

	// Calcular dias até segunda-feira
	daysToMonday := weekday - 1

	// Retornar início da semana (00:00:00)
	return time.Date(t.Year(), t.Month(), t.Day()-daysToMonday, 0, 0, 0, 0, t.Location())
}

// GetWeeklyStats retorna estatísticas da semana para um usuário
func (wc *WeeklyCounter) GetWeeklyStats(userEmail string) (map[string]interface{}, error) {
	count, err := wc.CountThisWeek(userEmail)
	if err != nil {
		return nil, err
	}

	limit := 200
	remaining := limit - count
	percentage := float64(count) / float64(limit) * 100

	return map[string]interface{}{
		"count":      count,
		"limit":      limit,
		"remaining":  remaining,
		"percentage": percentage,
		"can_send":   count < limit,
	}, nil
}
