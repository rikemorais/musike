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

func (s *TrackingService) StartPeriodicTracking() {
	log.Println("Starting periodic tracking service...")

	ticker := time.NewTicker(30 * time.Second) // Verificar a cada 30 segundos
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.updateAllActiveTracking()
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

	for _, artist := range tracking.LastTrack.Artists {
		_, err = tx.ExecContext(ctx, `
			INSERT INTO artists (id, name, popularity, created_at) 
			VALUES ($1, $2, $3, NOW()) 
			ON CONFLICT (id) DO UPDATE SET 
				name = EXCLUDED.name,
				popularity = EXCLUDED.popularity
		`, artist.ID, artist.Name, tracking.LastTrack.Popularity)

		if err != nil {
			log.Printf("Error saving artist: %v", err)
			continue
		}
	}

	album := tracking.LastTrack.Album
	imageURL := ""
	if len(album.Images) > 0 {
		imageURL = album.Images[0].URL
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO albums (id, name, release_date, image_url, created_at) 
		VALUES ($1, $2, $3, $4, NOW()) 
		ON CONFLICT (id) DO UPDATE SET 
			name = EXCLUDED.name,
			image_url = EXCLUDED.image_url
	`, album.ID, album.Name, album.ReleaseDate, imageURL)

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
