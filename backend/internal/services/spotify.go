package services

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"musike-backend/internal/config"

	"golang.org/x/oauth2"
)

type SpotifyService struct {
	config *config.Config
	client *http.Client
}

type SpotifyUser struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
	Email       string `json:"email"`
	Country     string `json:"country"`
	Followers   struct {
		Total int `json:"total"`
	} `json:"followers"`
	Images []struct {
		URL string `json:"url"`
	} `json:"images"`
}

type SpotifyTrack struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Artists     []SpotifyArtist `json:"artists"`
	Album       SpotifyAlbum    `json:"album"`
	Duration    int             `json:"duration_ms"`
	Popularity  int             `json:"popularity"`
	PreviewURL  string          `json:"preview_url"`
	ExternalIDs struct {
		ISRC string `json:"isrc"`
	} `json:"external_ids"`
}

type SpotifyArtist struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	Genres     []string `json:"genres"`
	Popularity int      `json:"popularity"`
	Images     []struct {
		URL string `json:"url"`
	} `json:"images"`
}

type SpotifyAlbum struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	ReleaseDate string `json:"release_date"`
	Images      []struct {
		URL string `json:"url"`
	} `json:"images"`
}

type TopTracksResponse struct {
	Items []SpotifyTrack `json:"items"`
	Total int            `json:"total"`
}

type TopArtistsResponse struct {
	Items []SpotifyArtist `json:"items"`
	Total int             `json:"total"`
}

type RecentlyPlayedResponse struct {
	Items []struct {
		Track    SpotifyTrack `json:"track"`
		PlayedAt time.Time    `json:"played_at"`
	} `json:"items"`
}

func NewSpotifyService(cfg *config.Config) *SpotifyService {
	return &SpotifyService{
		config: cfg,
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

func (s *SpotifyService) GetUserProfile(token *oauth2.Token) (*SpotifyUser, error) {
	req, err := http.NewRequest("GET", "https://api.spotify.com/v1/me", nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+token.AccessToken)

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("spotify API error: %d", resp.StatusCode)
	}

	var user SpotifyUser
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, err
	}

	return &user, nil
}

func (s *SpotifyService) GetTopTracks(token *oauth2.Token, timeRange string, limit int) (*TopTracksResponse, error) {
	params := url.Values{}
	params.Set("time_range", timeRange)
	params.Set("limit", strconv.Itoa(limit))

	apiURL := "https://api.spotify.com/v1/me/top/tracks?" + params.Encode()

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+token.AccessToken)

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("spotify API error: %d", resp.StatusCode)
	}

	var tracks TopTracksResponse
	if err := json.NewDecoder(resp.Body).Decode(&tracks); err != nil {
		return nil, err
	}

	return &tracks, nil
}

func (s *SpotifyService) GetTopArtists(token *oauth2.Token, timeRange string, limit int) (*TopArtistsResponse, error) {
	params := url.Values{}
	params.Set("time_range", timeRange)
	params.Set("limit", strconv.Itoa(limit))

	apiURL := "https://api.spotify.com/v1/me/top/artists?" + params.Encode()

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+token.AccessToken)

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("spotify API error: %d", resp.StatusCode)
	}

	var artists TopArtistsResponse
	if err := json.NewDecoder(resp.Body).Decode(&artists); err != nil {
		return nil, err
	}

	return &artists, nil
}

func (s *SpotifyService) GetRecentlyPlayed(token *oauth2.Token, limit int) (*RecentlyPlayedResponse, error) {
	params := url.Values{}
	params.Set("limit", strconv.Itoa(limit))

	apiURL := "https://api.spotify.com/v1/me/player/recently-played?" + params.Encode()

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+token.AccessToken)

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("spotify API error: %d", resp.StatusCode)
	}

	var recent RecentlyPlayedResponse
	if err := json.NewDecoder(resp.Body).Decode(&recent); err != nil {
		return nil, err
	}

	return &recent, nil
}

func (s *SpotifyService) GetRecommendations(token *oauth2.Token, seedArtists, seedTracks []string) (map[string]interface{}, error) {
	params := url.Values{}

	if len(seedArtists) > 0 {
		params.Set("seed_artists", fmt.Sprintf("%v", seedArtists))
	}
	if len(seedTracks) > 0 {
		params.Set("seed_tracks", fmt.Sprintf("%v", seedTracks))
	}
	params.Set("limit", "20")

	apiURL := "https://api.spotify.com/v1/recommendations?" + params.Encode()

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+token.AccessToken)

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	return result, nil
}
