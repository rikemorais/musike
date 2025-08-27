package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"musike-backend/internal/config"

	"github.com/lib/pq"
)

type TrackingService struct {
	config         *config.Config
	db             *sql.DB
	httpClient     *http.Client
	activeTracking map[string]*UserTracking
	trackingMutex  sync.RWMutex
	stopChannel    chan bool
}

type UserTracking struct {
	UserID        string
	SpotifyToken  string
	LastTrack     *CurrentlyPlayingTrack
	SessionStart  time.Time
	TotalPlayTime int64
	LastUpdated   time.Time
	IsActive      bool
}

type CurrentlyPlayingTrack struct {
	ID         string           `json:"id"`
	Name       string           `json:"name"`
	Artists    []SpotifyArtist  `json:"artists"`
	Album      SpotifyAlbum     `json:"album"`
	DurationMs int              `json:"duration_ms"`
	ProgressMs int              `json:"progress_ms"`
	IsPlaying  bool             `json:"is_playing"`
	Popularity int              `json:"popularity"`
	PreviewURL string           `json:"preview_url"`
	Context    *PlaybackContext `json:"context"`
}

type PlaybackContext struct {
	Type string `json:"type"` // playlist, album, artist, etc.
	URI  string `json:"uri"`
}

func NewTrackingService(cfg *config.Config, db *sql.DB) *TrackingService {
	return &TrackingService{
		config:         cfg,
		db:             db,
		httpClient:     &http.Client{Timeout: 30 * time.Second},
		activeTracking: make(map[string]*UserTracking),
		stopChannel:    make(chan bool),
	}
}

func (s *TrackingService) StartTracking(userID, spotifyToken string) error {
	s.trackingMutex.Lock()
	defer s.trackingMutex.Unlock()

	log.Printf("Starting tracking for user: %s", userID)

	s.activeTracking[userID] = &UserTracking{
		UserID:       userID,
		SpotifyToken: spotifyToken,
		SessionStart: time.Now(),
		LastUpdated:  time.Now(),
		IsActive:     true,
	}

	return nil
}

func (s *TrackingService) StopTracking(userID string) error {
	s.trackingMutex.Lock()
	defer s.trackingMutex.Unlock()

	if tracking, exists := s.activeTracking[userID]; exists {
		tracking.IsActive = false
		log.Printf("Stopped tracking for user: %s", userID)

		if tracking.LastTrack != nil {
			s.saveListeningSession(tracking)
		}

		delete(s.activeTracking, userID)
	}

	return nil
}

func (s *TrackingService) GetCurrentTrack(spotifyToken string) (*CurrentlyPlayingTrack, error) {
	req, err := http.NewRequest("GET", "https://api.spotify.com/v1/me/player/currently-playing", nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+spotifyToken)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 204 {
		return nil, nil
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("spotify API error: %d", resp.StatusCode)
	}

	var response struct {
		Item       *CurrentlyPlayingTrack `json:"item"`
		IsPlaying  bool                   `json:"is_playing"`
		ProgressMs int                    `json:"progress_ms"`
		Context    *PlaybackContext       `json:"context"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}

	if response.Item == nil {
		return nil, nil
	}

	response.Item.IsPlaying = response.IsPlaying
	response.Item.ProgressMs = response.ProgressMs
	response.Item.Context = response.Context

	return response.Item, nil
}

type RecentlyPlayedTrack struct {
	Track    *CurrentlyPlayingTrack `json:"track"`
	PlayedAt string                 `json:"played_at"`
}

type RecentlyPlayedResponseCustom struct {
	Items []RecentlyPlayedTrack `json:"items"`
}

func (s *TrackingService) GetRecentlyPlayed(spotifyToken string, limit int, after int64) (*RecentlyPlayedResponseCustom, error) {
	url := fmt.Sprintf("https://api.spotify.com/v1/me/player/recently-played?limit=%d", limit)
	if after > 0 {
		url += fmt.Sprintf("&after=%d", after)
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+spotifyToken)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("spotify API error: %d", resp.StatusCode)
	}

	var response RecentlyPlayedResponseCustom
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}

	return &response, nil
}

func (s *TrackingService) StartPeriodicTracking() {
	log.Println("Starting periodic tracking service...")

	ticker := time.NewTicker(15 * time.Second) // Verificar a cada 15 segundos para ser mais responsivo
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.updateAllActiveTracking()
			s.syncRecentlyPlayedTracks()
		case <-s.stopChannel:
			log.Println("Stopping periodic tracking service...")
			return
		}
	}
}

func (s *TrackingService) updateAllActiveTracking() {
	s.trackingMutex.RLock()
	activeUsers := make([]*UserTracking, 0, len(s.activeTracking))
	for _, tracking := range s.activeTracking {
		if tracking.IsActive {
			activeUsers = append(activeUsers, tracking)
		}
	}
	s.trackingMutex.RUnlock()

	for _, tracking := range activeUsers {
		s.updateUserTracking(tracking)
	}
}

func (s *TrackingService) updateUserTracking(tracking *UserTracking) {
	currentTrack, err := s.GetCurrentTrack(tracking.SpotifyToken)
	if err != nil {
		log.Printf("Error getting current track for user %s: %v", tracking.UserID, err)
		return
	}

	s.trackingMutex.Lock()
	defer s.trackingMutex.Unlock()

	now := time.Now()

	if currentTrack == nil {
		if tracking.LastTrack != nil {
			s.saveListeningSession(tracking)
			tracking.LastTrack = nil
		}
		tracking.LastUpdated = now
		return
	}

	if tracking.LastTrack == nil || tracking.LastTrack.ID != currentTrack.ID {
		if tracking.LastTrack != nil {
			s.saveListeningSession(tracking)
		}

		tracking.LastTrack = currentTrack
		tracking.SessionStart = now
		tracking.TotalPlayTime = 0

		log.Printf("User %s started playing: %s by %s",
			tracking.UserID, currentTrack.Name,
			strings.Join(getArtistNames(currentTrack.Artists), ", "))
	}

	if currentTrack.IsPlaying {
		timeDiff := now.Sub(tracking.LastUpdated)
		if timeDiff > 0 && timeDiff < 2*time.Minute { // Evitar valores absurdos
			tracking.TotalPlayTime += int64(timeDiff.Milliseconds())
		}
	}

	tracking.LastUpdated = now
}

func (s *TrackingService) saveListeningSession(tracking *UserTracking) {
	if tracking.LastTrack == nil {
		return
	}

	if tracking.TotalPlayTime < 30000 {
		return
	}

	ctx := context.Background()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		log.Printf("Error starting transaction: %v", err)
		return
	}
	defer tx.Rollback()

	// Salvar artistas com detalhes completos
	for _, artist := range tracking.LastTrack.Artists {
		s.saveArtistWithDetails(ctx, tx, tracking.SpotifyToken, artist.ID, artist.Name, tracking.LastTrack.Popularity)
	}

	album := tracking.LastTrack.Album
	imageURL := ""
	if len(album.Images) > 0 {
		imageURL = album.Images[0].URL
	}

	// Handle release date - Spotify sometimes gives just year ("1986") or partial date
	var releaseDate interface{}
	if album.ReleaseDate != "" {
		// If it's just a year (4 digits), convert to date
		if len(album.ReleaseDate) == 4 {
			releaseDate = album.ReleaseDate + "-01-01"
		} else {
			releaseDate = album.ReleaseDate
		}
	} else {
		releaseDate = nil
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO albums (id, name, release_date, image_url, created_at) 
		VALUES ($1, $2, $3, $4, NOW()) 
		ON CONFLICT (id) DO UPDATE SET 
			name = EXCLUDED.name,
			image_url = EXCLUDED.image_url
	`, album.ID, album.Name, releaseDate, imageURL)

	if err != nil {
		log.Printf("Error saving album: %v", err)
		return
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO tracks (id, name, album_id, duration_ms, popularity, preview_url, created_at) 
		VALUES ($1, $2, $3, $4, $5, $6, NOW()) 
		ON CONFLICT (id) DO UPDATE SET 
			name = EXCLUDED.name,
			duration_ms = EXCLUDED.duration_ms,
			popularity = EXCLUDED.popularity,
			preview_url = EXCLUDED.preview_url
	`, tracking.LastTrack.ID, tracking.LastTrack.Name, album.ID,
		tracking.LastTrack.DurationMs, tracking.LastTrack.Popularity, tracking.LastTrack.PreviewURL)

	if err != nil {
		log.Printf("Error saving track: %v", err)
		return
	}

	for _, artist := range tracking.LastTrack.Artists {
		_, err = tx.ExecContext(ctx, `
			INSERT INTO track_artists (track_id, artist_id) 
			VALUES ($1, $2) 
			ON CONFLICT DO NOTHING
		`, tracking.LastTrack.ID, artist.ID)

		if err != nil {
			log.Printf("Error saving track-artist relation: %v", err)
		}
	}

	contextType := ""
	contextURI := ""
	if tracking.LastTrack.Context != nil {
		contextType = tracking.LastTrack.Context.Type
		contextURI = tracking.LastTrack.Context.URI
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO listening_history (user_id, track_id, played_at, context_type, context_uri, created_at) 
		VALUES ($1, $2, $3, $4, $5, NOW())
	`, tracking.UserID, tracking.LastTrack.ID, tracking.SessionStart, contextType, contextURI)

	if err != nil {
		log.Printf("Error saving listening history: %v", err)
		return
	}

	if err = tx.Commit(); err != nil {
		log.Printf("Error committing transaction: %v", err)
		return
	}

	log.Printf("Saved listening session for user %s: %s (%.1f seconds)",
		tracking.UserID, tracking.LastTrack.Name, float64(tracking.TotalPlayTime)/1000)
}

func (s *TrackingService) syncRecentlyPlayedTracks() {
	s.trackingMutex.RLock()
	activeUsers := make([]*UserTracking, 0, len(s.activeTracking))
	for _, tracking := range s.activeTracking {
		if tracking.IsActive {
			activeUsers = append(activeUsers, tracking)
		}
	}
	s.trackingMutex.RUnlock()

	for _, tracking := range activeUsers {
		s.syncUserRecentlyPlayed(tracking)
	}
}

func (s *TrackingService) syncUserRecentlyPlayed(tracking *UserTracking) {
	// Pegar última entrada no banco para este usuário para saber onde parar
	ctx := context.Background()
	var lastPlayedAt time.Time
	err := s.db.QueryRowContext(ctx, `
		SELECT MAX(played_at) FROM listening_history WHERE user_id = $1
	`, tracking.UserID).Scan(&lastPlayedAt)

	if err != nil && err != sql.ErrNoRows {
		log.Printf("Error getting last played time for user %s: %v", tracking.UserID, err)
		return
	}

	// Converter para timestamp Unix em milissegundos
	var afterTimestamp int64 = 0
	if !lastPlayedAt.IsZero() {
		afterTimestamp = lastPlayedAt.UnixMilli()
	}

	recent, err := s.GetRecentlyPlayed(tracking.SpotifyToken, 50, afterTimestamp)
	if err != nil {
		log.Printf("Error getting recently played for user %s: %v", tracking.UserID, err)
		return
	}

	// Processar músicas mais antigas primeiro (ordem cronológica)
	for i := len(recent.Items) - 1; i >= 0; i-- {
		item := recent.Items[i]
		s.saveRecentlyPlayedTrack(tracking.UserID, tracking.SpotifyToken, &item)
	}
}

func (s *TrackingService) saveRecentlyPlayedTrack(userID, spotifyToken string, recentTrack *RecentlyPlayedTrack) {
	if recentTrack.Track == nil {
		return
	}

	// Parse do timestamp
	playedAt, err := time.Parse(time.RFC3339, recentTrack.PlayedAt)
	if err != nil {
		log.Printf("Error parsing played_at time: %v", err)
		return
	}

	// Verificar se já existe no banco
	ctx := context.Background()
	var count int
	err = s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM listening_history 
		WHERE user_id = $1 AND track_id = $2 AND played_at = $3
	`, userID, recentTrack.Track.ID, playedAt).Scan(&count)

	if err != nil {
		log.Printf("Error checking existing track: %v", err)
		return
	}

	if count > 0 {
		return // Já existe
	}

	track := recentTrack.Track

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		log.Printf("Error starting transaction: %v", err)
		return
	}
	defer tx.Rollback()

	// Salvar artistas com detalhes completos
	for _, artist := range track.Artists {
		s.saveArtistWithDetails(ctx, tx, spotifyToken, artist.ID, artist.Name, track.Popularity)
	}

	// Salvar álbum
	album := track.Album
	imageURL := ""
	if len(album.Images) > 0 {
		imageURL = album.Images[0].URL
	}

	var releaseDate interface{}
	if album.ReleaseDate != "" {
		if len(album.ReleaseDate) == 4 {
			releaseDate = album.ReleaseDate + "-01-01"
		} else {
			releaseDate = album.ReleaseDate
		}
	} else {
		releaseDate = nil
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO albums (id, name, release_date, image_url, created_at) 
		VALUES ($1, $2, $3, $4, NOW()) 
		ON CONFLICT (id) DO UPDATE SET 
			name = EXCLUDED.name,
			image_url = EXCLUDED.image_url
	`, album.ID, album.Name, releaseDate, imageURL)

	if err != nil {
		log.Printf("Error saving album: %v", err)
		return
	}

	// Salvar track
	_, err = tx.ExecContext(ctx, `
		INSERT INTO tracks (id, name, album_id, duration_ms, popularity, preview_url, created_at) 
		VALUES ($1, $2, $3, $4, $5, $6, NOW()) 
		ON CONFLICT (id) DO UPDATE SET 
			name = EXCLUDED.name,
			duration_ms = EXCLUDED.duration_ms,
			popularity = EXCLUDED.popularity,
			preview_url = EXCLUDED.preview_url
	`, track.ID, track.Name, album.ID, track.DurationMs, track.Popularity, track.PreviewURL)

	if err != nil {
		log.Printf("Error saving track: %v", err)
		return
	}

	// Salvar relações track-artist
	for _, artist := range track.Artists {
		_, err = tx.ExecContext(ctx, `
			INSERT INTO track_artists (track_id, artist_id) 
			VALUES ($1, $2) 
			ON CONFLICT DO NOTHING
		`, track.ID, artist.ID)

		if err != nil {
			log.Printf("Error saving track-artist relation: %v", err)
		}
	}

	// Salvar histórico
	contextType := ""
	contextURI := ""
	if track.Context != nil {
		contextType = track.Context.Type
		contextURI = track.Context.URI
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO listening_history (user_id, track_id, played_at, context_type, context_uri, created_at) 
		VALUES ($1, $2, $3, $4, $5, NOW())
	`, userID, track.ID, playedAt, contextType, contextURI)

	if err != nil {
		log.Printf("Error saving listening history: %v", err)
		return
	}

	if err = tx.Commit(); err != nil {
		log.Printf("Error committing transaction: %v", err)
		return
	}

	log.Printf("Synced recently played track for user %s: %s by %s (played at %s)",
		userID, track.Name, strings.Join(getArtistNames(track.Artists), ", "), playedAt.Format("15:04:05"))
}

func (s *TrackingService) GetArtistDetails(spotifyToken, artistID string) (*SpotifyArtist, error) {
	url := fmt.Sprintf("https://api.spotify.com/v1/artists/%s", artistID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+spotifyToken)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("spotify API error: %d", resp.StatusCode)
	}

	var artist SpotifyArtist
	if err := json.NewDecoder(resp.Body).Decode(&artist); err != nil {
		return nil, err
	}

	return &artist, nil
}

func (s *TrackingService) saveArtistWithDetails(ctx context.Context, tx *sql.Tx, spotifyToken, artistID, artistName string, popularity int) {
	log.Printf("Checking artist details for %s (%s)", artistName, artistID)

	// Primeiro verificar se o artista já existe com gêneros
	var genresCount sql.NullInt32
	err := tx.QueryRowContext(ctx, `
		SELECT array_length(genres, 1) FROM artists WHERE id = $1
	`, artistID).Scan(&genresCount)

	if err == nil && genresCount.Valid && genresCount.Int32 > 0 {
		log.Printf("Artist %s already has %d genres, skipping enrichment", artistName, genresCount.Int32)
		// Artista já existe com gêneros, apenas atualizar popularidade se necessário
		_, err = tx.ExecContext(ctx, `
			UPDATE artists SET popularity = $2 WHERE id = $1 AND popularity != $2
		`, artistID, popularity)
		if err != nil {
			log.Printf("Error updating artist popularity: %v", err)
		}
		return
	}

	log.Printf("Artist %s needs genre enrichment, fetching from Spotify...", artistName)

	// Buscar detalhes completos do artista no Spotify
	artistDetails, err := s.GetArtistDetails(spotifyToken, artistID)
	if err != nil {
		log.Printf("Error getting artist details for %s: %v", artistID, err)
		// Fallback: salvar com dados básicos
		_, err = tx.ExecContext(ctx, `
			INSERT INTO artists (id, name, popularity, created_at) 
			VALUES ($1, $2, $3, NOW()) 
			ON CONFLICT (id) DO UPDATE SET 
				name = EXCLUDED.name,
				popularity = EXCLUDED.popularity
		`, artistID, artistName, popularity)
		return
	}

	// Imagem do artista
	var imageURL string
	if len(artistDetails.Images) > 0 {
		imageURL = artistDetails.Images[0].URL
	}

	// Preparar gêneros como array PostgreSQL usando lib/pq
	var genresArray pq.StringArray
	if len(artistDetails.Genres) > 0 {
		genresArray = pq.StringArray(artistDetails.Genres)
	} else {
		genresArray = pq.StringArray{}
	}

	// Salvar artista com todos os detalhes
	_, err = tx.ExecContext(ctx, `
		INSERT INTO artists (id, name, genres, popularity, image_url, created_at) 
		VALUES ($1, $2, $3, $4, $5, NOW()) 
		ON CONFLICT (id) DO UPDATE SET 
			name = EXCLUDED.name,
			genres = EXCLUDED.genres,
			popularity = EXCLUDED.popularity,
			image_url = EXCLUDED.image_url
	`, artistID, artistDetails.Name, genresArray, artistDetails.Popularity, imageURL)

	if err != nil {
		log.Printf("Error saving artist with details: %v", err)
	} else {
		log.Printf("Saved artist details for %s: %v", artistDetails.Name, artistDetails.Genres)
	}
}

func (s *TrackingService) StopPeriodicTracking() {
	s.stopChannel <- true
}

func (s *TrackingService) GetActiveTrackingCount() int {
	s.trackingMutex.RLock()
	defer s.trackingMutex.RUnlock()

	count := 0
	for _, tracking := range s.activeTracking {
		if tracking.IsActive {
			count++
		}
	}
	return count
}

func getArtistNames(artists []SpotifyArtist) []string {
	names := make([]string, len(artists))
	for i, artist := range artists {
		names[i] = artist.Name
	}
	return names
}
