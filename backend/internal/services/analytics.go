package services

import (
	"database/sql"
	"fmt"
	"time"

	"golang.org/x/oauth2"
	"musike-backend/internal/config"
)

type AnalyticsService struct {
	config *config.Config
	db     *sql.DB
}

type UserAnalytics struct {
	UserID                 string                `json:"user_id"`
	TotalListeningTime     int64                 `json:"total_listening_time_ms"`
	ActualListeningTime    int64                 `json:"actual_listening_time_ms"`
	TotalPlays             int                   `json:"total_plays"`
	AveragePlayTime        int64                 `json:"average_play_time_ms"`
	AverageTrackPopularity float64               `json:"average_track_popularity"`
	AvgListeningPercentage float64               `json:"avg_listening_percentage"`
	TopGenres              []GenreStats          `json:"top_genres"`
	ListeningPatterns      ListeningPatterns     `json:"listening_patterns"`
	DiversityScore         float64               `json:"diversity_score"`
	RecentActivity         []ActivityPoint       `json:"recent_activity"`
	MonthlyStats           map[string]MonthStats `json:"monthly_stats"`
}

type GenreStats struct {
	Genre      string  `json:"genre"`
	Percentage float64 `json:"percentage"`
	TrackCount int     `json:"track_count"`
	PlayCount  int     `json:"play_count"`
	TotalTime  int64   `json:"total_time_ms"`
}

type ListeningPatterns struct {
	PeakHours    []int              `json:"peak_hours"`
	WeekdayUsage []float64          `json:"weekday_usage"`
	Seasonality  map[string]float64 `json:"seasonality"`
}

type ActivityPoint struct {
	Date        time.Time `json:"date"`
	TrackCount  int       `json:"track_count"`
	UniqueTraks int       `json:"unique_tracks"`
	Duration    int64     `json:"duration_ms"`
}

type MonthStats struct {
	TracksPlayed    int     `json:"tracks_played"`
	UniqueArtists   int     `json:"unique_artists"`
	TopGenre        string  `json:"top_genre"`
	AvgDailyMinutes float64 `json:"avg_daily_minutes"`
}

func NewAnalyticsService(cfg *config.Config, db *sql.DB) *AnalyticsService {
	return &AnalyticsService{
		config: cfg,
		db:     db,
	}
}

func (a *AnalyticsService) GenerateUserAnalytics(userID string, timeFilter string, spotifyService *SpotifyService, token *oauth2.Token) (*UserAnalytics, error) {
	topTracks, err := spotifyService.GetTopTracks(token, "long_term", 50)
	if err != nil {
		return nil, fmt.Errorf("failed to get top tracks: %w", err)
	}

	topArtists, err := spotifyService.GetTopArtists(token, "long_term", 50)
	if err != nil {
		return nil, fmt.Errorf("failed to get top artists: %w", err)
	}

	recentlyPlayed, err := spotifyService.GetRecentlyPlayed(token, 50)
	if err != nil {
		return nil, fmt.Errorf("failed to get recently played: %w", err)
	}

	analytics := &UserAnalytics{
		UserID: userID,
	}

	// Usar dados do banco local para tempo total baseado no filtro
	analytics.TotalListeningTime, err = a.calculateTotalListeningTimeFromDB(userID, timeFilter)
	if err != nil {
		// Fallback para cálculo baseado na API do Spotify se erro no banco
		analytics.TotalListeningTime = a.calculateTotalListeningTime(topTracks.Items)
	}

	// Calcular tempo de escuta real e porcentagem média
	analytics.ActualListeningTime, analytics.AvgListeningPercentage, err = a.calculateActualListeningStats(userID, timeFilter)
	if err != nil {
		// Se não houver dados de tempo de escuta real, usar o tempo total como fallback
		analytics.ActualListeningTime = analytics.TotalListeningTime
		analytics.AvgListeningPercentage = 100.0
	}

	// Calcular total de plays
	analytics.TotalPlays, err = a.calculateTotalPlays(userID, timeFilter)
	if err != nil {
		// Fallback para 0 se não conseguir calcular
		analytics.TotalPlays = 0
	}

	// Calcular tempo médio de play
	analytics.AveragePlayTime, err = a.calculateAveragePlayTime(userID, timeFilter)
	if err != nil {
		// Fallback para 0 se não conseguir calcular
		analytics.AveragePlayTime = 0
	}

	// Calcular popularidade média das tracks
	analytics.AverageTrackPopularity, err = a.calculateAverageTrackPopularity(userID, timeFilter)
	if err != nil {
		// Fallback para 0 se não conseguir calcular
		analytics.AverageTrackPopularity = 0.0
	}

	// Calcular top gêneros do banco de dados local baseado no filtro
	analytics.TopGenres, err = a.analyzeGenresFromDB(userID, timeFilter)
	if err != nil {
		// Fallback para análise baseada na API do Spotify se erro no banco
		analytics.TopGenres = a.analyzeGenres(topArtists.Items)
	}

	// Usar dados do banco local para padrões de escuta baseado no filtro
	analytics.ListeningPatterns, err = a.analyzeListeningPatternsFromDB(userID, timeFilter)
	if err != nil {
		// Fallback para análise baseada na API do Spotify se erro no banco
		analytics.ListeningPatterns = a.analyzeListeningPatterns(recentlyPlayed.Items)
	}

	analytics.DiversityScore = a.calculateDiversityScore(topArtists.Items, topTracks.Items)

	// Usar dados do banco local para atividade recente baseado no filtro
	analytics.RecentActivity, err = a.analyzeRecentActivityFromDB(userID, timeFilter)
	if err != nil {
		// Fallback para análise baseada na API do Spotify se erro no banco
		fmt.Printf("Erro ao buscar atividade recente do DB: %v, usando fallback da API\n", err)
		analytics.RecentActivity = a.analyzeRecentActivity(recentlyPlayed.Items)
	} else {
		fmt.Printf("Atividade recente do DB retornou %d dias de dados\n", len(analytics.RecentActivity))
	}

	analytics.MonthlyStats = a.generateMonthlyStats()

	return analytics, nil
}

func (a *AnalyticsService) calculateTotalListeningTime(tracks []SpotifyTrack) int64 {
	var total int64
	for _, track := range tracks {
		total += int64(track.Duration)
	}
	return total
}

func (a *AnalyticsService) calculateTotalListeningTimeFromDB(userID string, timeFilter string) (int64, error) {
	if a.db == nil {
		return 0, fmt.Errorf("database not available")
	}

	// Determinar o período de filtro baseado no parâmetro
	var startDate time.Time
	now := time.Now()

	switch timeFilter {
	case "6months":
		startDate = now.AddDate(0, -6, 0)
	case "1year":
		startDate = now.AddDate(-1, 0, 0)
	case "alltime":
		startDate = time.Time{} // Data zero = sem filtro
	default:
		startDate = now.AddDate(0, -6, 0) // Default 6 meses
	}

	var query string
	var args []interface{}

	if timeFilter == "alltime" {
		// Para tempo total, usar TODAS as escutas (sem filtro de duração mínima)
		// Usar duração da música quando disponível, senão usar tempo escutado
		query = `
			SELECT COALESCE(SUM(
				CASE 
					WHEN t.duration_ms > 0 THEN t.duration_ms
					ELSE GREATEST(lh.listened_duration_ms, 0)
				END
			), 0) as total_time
			FROM listening_history lh
			JOIN tracks t ON lh.track_id = t.id
			WHERE lh.user_id = $1`
		args = []interface{}{userID}
	} else {
		// Com filtro de data para outros filtros
		query = `
			SELECT COALESCE(SUM(
				CASE 
					WHEN t.duration_ms > 0 THEN t.duration_ms
					ELSE GREATEST(lh.listened_duration_ms, 0)
				END
			), 0) as total_time
			FROM listening_history lh
			JOIN tracks t ON lh.track_id = t.id
			WHERE lh.user_id = $1 AND lh.played_at >= $2`
		args = []interface{}{userID, startDate}
	}

	var totalTime int64
	err := a.db.QueryRow(query, args...).Scan(&totalTime)
	if err != nil {
		return 0, fmt.Errorf("failed to calculate total listening time: %w", err)
	}

	return totalTime, nil
}

func (a *AnalyticsService) calculateActualListeningStats(userID string, timeFilter string) (int64, float64, error) {
	if a.db == nil {
		return 0, 0.0, fmt.Errorf("database not available")
	}

	// Determinar o período de filtro baseado no parâmetro
	var startDate time.Time
	now := time.Now()

	switch timeFilter {
	case "6months":
		startDate = now.AddDate(0, -6, 0)
	case "1year":
		startDate = now.AddDate(-1, 0, 0)
	case "alltime":
		startDate = time.Time{} // Data zero = sem filtro
	default:
		startDate = now.AddDate(0, -6, 0) // Default 6 meses
	}

	var query string
	var args []interface{}

	if timeFilter == "alltime" {
		query = `
			SELECT 
				COALESCE(SUM(lh.listened_duration_ms), 0) as actual_time,
				COALESCE(AVG(lh.listening_percentage), 0) as avg_percentage,
				COUNT(*) as total_tracks
			FROM listening_history lh
			WHERE lh.user_id = $1 AND lh.listened_duration_ms > 0`
		args = []interface{}{userID}
	} else {
		query = `
			SELECT 
				COALESCE(SUM(lh.listened_duration_ms), 0) as actual_time,
				COALESCE(AVG(lh.listening_percentage), 0) as avg_percentage,
				COUNT(*) as total_tracks
			FROM listening_history lh
			WHERE lh.user_id = $1 AND lh.played_at >= $2 AND lh.listened_duration_ms > 0`
		args = []interface{}{userID, startDate}
	}

	var actualTime int64
	var avgPercentage float64
	var totalTracks int

	err := a.db.QueryRow(query, args...).Scan(&actualTime, &avgPercentage, &totalTracks)
	if err != nil {
		return 0, 0.0, fmt.Errorf("failed to calculate actual listening stats: %w", err)
	}

	// Se não há dados de tempo escutado, retornar erro para usar fallback
	if totalTracks == 0 {
		return 0, 0.0, fmt.Errorf("no listening duration data available")
	}

	return actualTime, avgPercentage, nil
}

func (a *AnalyticsService) calculateTotalPlays(userID string, timeFilter string) (int, error) {
	if a.db == nil {
		return 0, fmt.Errorf("database not available")
	}

	// Determinar o período de filtro baseado no parâmetro
	var startDate time.Time
	now := time.Now()

	switch timeFilter {
	case "6months":
		startDate = now.AddDate(0, -6, 0)
	case "1year":
		startDate = now.AddDate(-1, 0, 0)
	case "alltime":
		startDate = time.Time{} // Data zero = sem filtro
	default:
		startDate = now.AddDate(0, -6, 0) // Default 6 meses
	}

	var query string
	var args []interface{}

	if timeFilter == "alltime" {
		query = `
			SELECT COUNT(*) as total_plays
			FROM listening_history lh
			WHERE lh.user_id = $1`
		args = []interface{}{userID}
	} else {
		query = `
			SELECT COUNT(*) as total_plays
			FROM listening_history lh
			WHERE lh.user_id = $1 AND lh.played_at >= $2`
		args = []interface{}{userID, startDate}
	}

	var totalPlays int
	err := a.db.QueryRow(query, args...).Scan(&totalPlays)
	if err != nil {
		return 0, fmt.Errorf("failed to calculate total plays: %w", err)
	}

	return totalPlays, nil
}

func (a *AnalyticsService) calculateAveragePlayTime(userID string, timeFilter string) (int64, error) {
	if a.db == nil {
		return 0, fmt.Errorf("database not available")
	}

	// Determinar o período de filtro baseado no parâmetro
	var startDate time.Time
	now := time.Now()

	switch timeFilter {
	case "6months":
		startDate = now.AddDate(0, -6, 0)
	case "1year":
		startDate = now.AddDate(-1, 0, 0)
	case "alltime":
		startDate = time.Time{} // Data zero = sem filtro
	default:
		startDate = now.AddDate(0, -6, 0) // Default 6 meses
	}

	var query string
	var args []interface{}

	if timeFilter == "alltime" {
		query = `
			SELECT COALESCE(AVG(lh.listened_duration_ms), 0) as avg_play_time
			FROM listening_history lh
			WHERE lh.user_id = $1 AND lh.listened_duration_ms > 0`
		args = []interface{}{userID}
	} else {
		query = `
			SELECT COALESCE(AVG(lh.listened_duration_ms), 0) as avg_play_time
			FROM listening_history lh
			WHERE lh.user_id = $1 AND lh.played_at >= $2 AND lh.listened_duration_ms > 0`
		args = []interface{}{userID, startDate}
	}

	var avgPlayTime float64
	err := a.db.QueryRow(query, args...).Scan(&avgPlayTime)
	if err != nil {
		return 0, fmt.Errorf("failed to calculate average play time: %w", err)
	}

	return int64(avgPlayTime), nil
}

func (a *AnalyticsService) calculateAverageTrackPopularity(userID string, timeFilter string) (float64, error) {
	if a.db == nil {
		return 0.0, fmt.Errorf("database not available")
	}

	// Determinar o período de filtro baseado no parâmetro
	var startDate time.Time
	now := time.Now()

	switch timeFilter {
	case "6months":
		startDate = now.AddDate(0, -6, 0)
	case "1year":
		startDate = now.AddDate(-1, 0, 0)
	case "alltime":
		startDate = time.Time{} // Data zero = sem filtro
	default:
		startDate = now.AddDate(0, -6, 0) // Default 6 meses
	}

	var query string
	var args []interface{}

	if timeFilter == "alltime" {
		query = `
			SELECT COALESCE(AVG(t.popularity), 0) as avg_popularity
			FROM listening_history lh
			JOIN tracks t ON lh.track_id = t.id
			WHERE lh.user_id = $1 AND t.popularity > 0`
		args = []interface{}{userID}
	} else {
		query = `
			SELECT COALESCE(AVG(t.popularity), 0) as avg_popularity
			FROM listening_history lh
			JOIN tracks t ON lh.track_id = t.id
			WHERE lh.user_id = $1 AND lh.played_at >= $2 AND t.popularity > 0`
		args = []interface{}{userID, startDate}
	}

	var avgPopularity float64
	err := a.db.QueryRow(query, args...).Scan(&avgPopularity)
	if err != nil {
		return 0.0, fmt.Errorf("failed to calculate average track popularity: %w", err)
	}

	return avgPopularity, nil
}

func (a *AnalyticsService) analyzeGenresFromDB(userID string, timeFilter string) ([]GenreStats, error) {
	if a.db == nil {
		return nil, fmt.Errorf("database not available")
	}

	// Determinar o período de filtro baseado no parâmetro
	var startDate time.Time
	now := time.Now()

	switch timeFilter {
	case "6months":
		startDate = now.AddDate(0, -6, 0)
	case "1year":
		startDate = now.AddDate(-1, 0, 0)
	case "alltime":
		startDate = time.Time{} // Data zero = sem filtro
	default:
		startDate = now.AddDate(0, -6, 0) // Default 6 meses
	}

	var query string
	var args []interface{}

	if timeFilter == "alltime" {
		query = `
			SELECT 
				UNNEST(string_to_array(TRIM(BOTH '[]"' FROM a.genres::text), '","')) as genre,
				COUNT(*) as play_count,
				COUNT(DISTINCT t.id) as track_count,
				COALESCE(SUM(lh.listened_duration_ms), 0) as total_time
			FROM listening_history lh
			JOIN tracks t ON lh.track_id = t.id
			JOIN artists a ON t.artist_id = a.id
			WHERE lh.user_id = $1 AND a.genres IS NOT NULL AND a.genres != '[]'
			GROUP BY genre
			ORDER BY play_count DESC
			LIMIT 10`
		args = []interface{}{userID}
	} else {
		query = `
			SELECT 
				UNNEST(string_to_array(TRIM(BOTH '[]"' FROM a.genres::text), '","')) as genre,
				COUNT(*) as play_count,
				COUNT(DISTINCT t.id) as track_count,
				COALESCE(SUM(lh.listened_duration_ms), 0) as total_time
			FROM listening_history lh
			JOIN tracks t ON lh.track_id = t.id
			JOIN artists a ON t.artist_id = a.id
			WHERE lh.user_id = $1 AND lh.played_at >= $2 AND a.genres IS NOT NULL AND a.genres != '[]'
			GROUP BY genre
			ORDER BY play_count DESC
			LIMIT 10`
		args = []interface{}{userID, startDate}
	}

	rows, err := a.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query genres from database: %w", err)
	}
	defer rows.Close()

	var genreStats []GenreStats
	var totalPlays int

	// Primeiro, calcular o total de plays para percentagens
	for rows.Next() {
		var genre string
		var playCount, trackCount int
		var totalTime int64
		if err := rows.Scan(&genre, &playCount, &trackCount, &totalTime); err != nil {
			continue
		}
		totalPlays += playCount
		genreStats = append(genreStats, GenreStats{
			Genre:      genre,
			TrackCount: trackCount,
			PlayCount:  playCount,
			TotalTime:  totalTime,
		})
	}

	// Calcular percentagens baseadas em play_count, não track_count
	for i := range genreStats {
		if totalPlays > 0 {
			genreStats[i].Percentage = float64(genreStats[i].PlayCount) / float64(totalPlays) * 100
		}
	}

	return genreStats, nil
}

func (a *AnalyticsService) analyzeListeningPatternsFromDB(userID string, timeFilter string) (ListeningPatterns, error) {
	if a.db == nil {
		return ListeningPatterns{}, fmt.Errorf("database not available")
	}

	// Determinar o período de filtro baseado no parâmetro
	var startDate time.Time
	now := time.Now()

	switch timeFilter {
	case "6months":
		startDate = now.AddDate(0, -6, 0)
	case "1year":
		startDate = now.AddDate(-1, 0, 0)
	case "alltime":
		startDate = time.Time{} // Data zero = sem filtro
	default:
		startDate = now.AddDate(0, -6, 0) // Default 6 meses
	}

	var query string
	var args []interface{}

	if timeFilter == "alltime" {
		// Sem filtro de data para 'alltime'
		query = `
			SELECT 
				EXTRACT(HOUR FROM lh.played_at) as hour,
				EXTRACT(DOW FROM lh.played_at) as weekday,
				COUNT(*) as count
			FROM listening_history lh
			WHERE lh.user_id = $1
			GROUP BY EXTRACT(HOUR FROM lh.played_at), EXTRACT(DOW FROM lh.played_at)
			ORDER BY hour, weekday`
		args = []interface{}{userID}
	} else {
		// Com filtro de data para outros filtros
		query = `
			SELECT 
				EXTRACT(HOUR FROM lh.played_at) as hour,
				EXTRACT(DOW FROM lh.played_at) as weekday,
				COUNT(*) as count
			FROM listening_history lh
			WHERE lh.user_id = $1 AND lh.played_at >= $2
			GROUP BY EXTRACT(HOUR FROM lh.played_at), EXTRACT(DOW FROM lh.played_at)
			ORDER BY hour, weekday`
		args = []interface{}{userID, startDate}
	}

	rows, err := a.db.Query(query, args...)
	if err != nil {
		return ListeningPatterns{}, fmt.Errorf("failed to query listening patterns: %w", err)
	}
	defer rows.Close()

	// Inicializar contadores
	hourCounts := make([]int, 24)
	weekdayCounts := make([]int, 7)

	// Processar resultados
	for rows.Next() {
		var hour, weekday, count int
		if err := rows.Scan(&hour, &weekday, &count); err != nil {
			continue
		}

		if hour >= 0 && hour < 24 {
			hourCounts[hour] += count
		}
		if weekday >= 0 && weekday < 7 {
			weekdayCounts[weekday] += count
		}
	}

	// Calcular horários de pico
	var peakHours []int
	maxCount := 0
	for _, count := range hourCounts {
		if count > maxCount {
			maxCount = count
		}
	}

	for i, count := range hourCounts {
		if count >= int(float64(maxCount)*0.7) { // 70% do pico
			peakHours = append(peakHours, i)
		}
	}

	// Calcular percentuais dos dias da semana
	total := 0
	for _, count := range weekdayCounts {
		total += count
	}

	weekdayUsage := make([]float64, 7)
	for i, count := range weekdayCounts {
		if total > 0 {
			weekdayUsage[i] = float64(count) / float64(total) * 100
		}
	}

	return ListeningPatterns{
		PeakHours:    peakHours,
		WeekdayUsage: weekdayUsage,
		Seasonality: map[string]float64{
			"spring": 25.0,
			"summer": 30.0,
			"autumn": 25.0,
			"winter": 20.0,
		},
	}, nil
}

func (a *AnalyticsService) analyzeGenres(artists []SpotifyArtist) []GenreStats {
	genreCount := make(map[string]int)
	totalTracks := len(artists)

	for _, artist := range artists {
		for _, genre := range artist.Genres {
			genreCount[genre]++
		}
	}

	var genreStats []GenreStats
	for genre, count := range genreCount {
		percentage := float64(count) / float64(totalTracks) * 100
		genreStats = append(genreStats, GenreStats{
			Genre:      genre,
			Percentage: percentage,
			TrackCount: count,
		})
	}

	return genreStats
}

func (a *AnalyticsService) analyzeListeningPatterns(recentTracks []struct {
	Track    SpotifyTrack `json:"track"`
	PlayedAt time.Time    `json:"played_at"`
}) ListeningPatterns {
	hourCounts := make([]int, 24)
	weekdayCounts := make([]int, 7)

	for _, item := range recentTracks {
		hour := item.PlayedAt.Hour()
		weekday := int(item.PlayedAt.Weekday())

		hourCounts[hour]++
		weekdayCounts[weekday]++
	}

	var peakHours []int
	maxCount := 0
	for _, count := range hourCounts {
		if count > maxCount {
			maxCount = count
		}
	}

	for i, count := range hourCounts {
		if count >= int(float64(maxCount)*0.7) { // 70% do pico
			peakHours = append(peakHours, i)
		}
	}

	total := 0
	for _, count := range weekdayCounts {
		total += count
	}

	weekdayUsage := make([]float64, 7)
	for i, count := range weekdayCounts {
		if total > 0 {
			weekdayUsage[i] = float64(count) / float64(total) * 100
		}
	}

	return ListeningPatterns{
		PeakHours:    peakHours,
		WeekdayUsage: weekdayUsage,
		Seasonality: map[string]float64{
			"spring": 25.0,
			"summer": 30.0,
			"autumn": 25.0,
			"winter": 20.0,
		},
	}
}

func (a *AnalyticsService) calculateDiversityScore(artists []SpotifyArtist, tracks []SpotifyTrack) float64 {
	uniqueGenres := make(map[string]bool)
	uniqueArtists := make(map[string]bool)

	for _, artist := range artists {
		uniqueArtists[artist.ID] = true
		for _, genre := range artist.Genres {
			uniqueGenres[genre] = true
		}
	}

	genreScore := float64(len(uniqueGenres)) / 10.0   // Normalizado para 10 gêneros
	artistScore := float64(len(uniqueArtists)) / 50.0 // Normalizado para 50 artistas

	if genreScore > 1.0 {
		genreScore = 1.0
	}
	if artistScore > 1.0 {
		artistScore = 1.0
	}

	return (genreScore + artistScore) / 2.0 * 100 // 0-100 score
}

func (a *AnalyticsService) analyzeRecentActivityFromDB(userID string, timeFilter string) ([]ActivityPoint, error) {
	if a.db == nil {
		return nil, fmt.Errorf("database not available")
	}

	// Para atividade recente, sempre mostrar os últimos 7 dias independente do filtro
	now := time.Now()

	// Criar um mapa para armazenar atividades por data
	activityMap := make(map[string]ActivityPoint)

	// Gerar todos os últimos 7 dias (incluindo hoje)
	fmt.Printf("Gerando datas para os últimos 7 dias:\n")
	for i := 6; i >= 0; i-- {
		date := now.AddDate(0, 0, -i)
		dateStr := date.Format("2006-01-02")
		fmt.Printf("Data gerada: %s\n", dateStr)
		activityMap[dateStr] = ActivityPoint{
			Date:        date,
			TrackCount:  0,
			UniqueTraks: 0,
			Duration:    0,
		}
	}

	startDate := now.AddDate(0, 0, -7)
	fmt.Printf("Consultando dados desde: %s\n", startDate.Format("2006-01-02 15:04:05"))

	// Primeiro, verificar se há dados na tabela
	var totalRows int
	countQuery := `SELECT COUNT(*) FROM listening_history WHERE user_id = $1`
	countErr := a.db.QueryRow(countQuery, userID).Scan(&totalRows)
	if countErr != nil {
		fmt.Printf("Erro ao contar registros: %v\n", countErr)
	} else {
		fmt.Printf("Total de registros na listening_history para user %s: %d\n", userID, totalRows)
	}

	query := `
		SELECT 
			TO_CHAR(lh.played_at, 'YYYY-MM-DD') as date,
			COUNT(*) as track_count,
			COUNT(DISTINCT lh.track_id) as unique_tracks,
			COALESCE(SUM(lh.listened_duration_ms), 0) as duration_ms
		FROM listening_history lh
		WHERE lh.user_id = $1 AND lh.played_at >= $2
		GROUP BY TO_CHAR(lh.played_at, 'YYYY-MM-DD')
		ORDER BY date DESC`

	rows, err := a.db.Query(query, userID, startDate)
	if err != nil {
		return nil, fmt.Errorf("failed to query recent activity: %w", err)
	}
	defer rows.Close()

	// Atualizar o mapa com dados reais
	dataCount := 0
	for rows.Next() {
		var dateStr string
		var trackCount, uniqueTracks int
		var durationMs int64

		if err := rows.Scan(&dateStr, &trackCount, &uniqueTracks, &durationMs); err != nil {
			continue
		}

		dataCount++
		fmt.Printf("Dados encontrados para %s: %d tracks, %d únicos, %d ms\n", dateStr, trackCount, uniqueTracks, durationMs)

		// Só incluir se a data está nos últimos 7 dias
		if activityPoint, exists := activityMap[dateStr]; exists {
			activityPoint.TrackCount = trackCount
			activityPoint.UniqueTraks = uniqueTracks
			activityPoint.Duration = durationMs
			activityMap[dateStr] = activityPoint
		}
	}

	fmt.Printf("Total de registros encontrados na query: %d\n", dataCount)

	// Converter mapa para slice ordenado (mais recente primeiro)
	var activity []ActivityPoint
	for i := 6; i >= 0; i-- {
		date := now.AddDate(0, 0, -i)
		dateStr := date.Format("2006-01-02")
		activityPoint := activityMap[dateStr]
		fmt.Printf("Data final %s: %d tracks, %d únicos, %d ms\n",
			dateStr, activityPoint.TrackCount, activityPoint.UniqueTraks, activityPoint.Duration)
		activity = append(activity, activityPoint)
	}

	return activity, nil
}

func (a *AnalyticsService) analyzeRecentActivity(recentTracks []struct {
	Track    SpotifyTrack `json:"track"`
	PlayedAt time.Time    `json:"played_at"`
}) []ActivityPoint {
	dailyActivity := make(map[string]*ActivityPoint)

	for _, item := range recentTracks {
		date := item.PlayedAt.Format("2006-01-02")

		if _, exists := dailyActivity[date]; !exists {
			parsedDate, _ := time.Parse("2006-01-02", date)
			dailyActivity[date] = &ActivityPoint{
				Date:        parsedDate,
				TrackCount:  0,
				UniqueTraks: 0,
				Duration:    0,
			}
		}

		dailyActivity[date].TrackCount++
		dailyActivity[date].Duration += int64(item.Track.Duration)
	}

	var activity []ActivityPoint
	for _, point := range dailyActivity {
		point.UniqueTraks = point.TrackCount // Simplificado - em produção seria mais preciso
		activity = append(activity, *point)
	}

	return activity
}

func (a *AnalyticsService) generateMonthlyStats() map[string]MonthStats {
	return map[string]MonthStats{
		"2025-08": {
			TracksPlayed:    1250,
			UniqueArtists:   85,
			TopGenre:        "pop",
			AvgDailyMinutes: 125.5,
		},
		"2025-07": {
			TracksPlayed:    1100,
			UniqueArtists:   78,
			TopGenre:        "rock",
			AvgDailyMinutes: 110.2,
		},
	}
}
