package handlers

import (
	"archive/zip"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lib/pq"
)

type ImportHandler struct {
	db *sql.DB
}

type SpotifyStreamingData struct {
	Timestamp        string `json:"ts"`
	Username         string `json:"username"`
	Platform         string `json:"platform"`
	MsPlayed         int    `json:"ms_played"`
	ConnCountry      string `json:"conn_country"`
	IPAddrDecrypted  string `json:"ip_addr_decrypted"`
	TrackName        string `json:"master_metadata_track_name"`
	ArtistName       string `json:"master_metadata_album_artist_name"`
	AlbumName        string `json:"master_metadata_album_album_name"`
	SpotifyTrackURI  string `json:"spotify_track_uri"`
	ReasonStart      string `json:"reason_start"`
	ReasonEnd        string `json:"reason_end"`
	Shuffle          bool   `json:"shuffle"`
	Skipped          *bool  `json:"skipped"`
	Offline          bool   `json:"offline"`
	OfflineTimestamp string `json:"offline_timestamp"`
	IncognitoMode    bool   `json:"incognito_mode"`
}

type ImportResult struct {
	ProcessedFiles  int           `json:"processed_files"`
	ProcessedTracks int           `json:"processed_tracks"`
	Errors          []string      `json:"errors"`
	Status          string        `json:"status"`
	ProcessingTime  time.Duration `json:"processing_time_ms"`
	ImportSummary   ImportSummary `json:"summary"`
}

type ImportSummary struct {
	TotalStreams    int           `json:"total_streams"`
	UniqueArtists   int           `json:"unique_artists"`
	UniqueTracks    int           `json:"unique_tracks"`
	TotalListenTime int64         `json:"total_listen_time_ms"`
	DateRange       DateRange     `json:"date_range"`
	TopArtists      []ArtistCount `json:"top_artists"`
	TopTracks       []TrackCount  `json:"top_tracks"`
}

type DateRange struct {
	From string `json:"from"`
	To   string `json:"to"`
}

type ArtistCount struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

type TrackCount struct {
	Name   string `json:"name"`
	Artist string `json:"artist"`
	Count  int    `json:"count"`
}

func NewImportHandler(db *sql.DB) *ImportHandler {
	return &ImportHandler{
		db: db,
	}
}

func (h *ImportHandler) ImportSpotifyData(c *gin.Context) {
	startTime := time.Now()

	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	log.Printf("Starting Spotify data import for user: %s", userID)

	form, err := c.MultipartForm()
	if err != nil {
		log.Printf("Failed to parse multipart form: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse form data"})
		return
	}

	files := form.File["files"]
	if len(files) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No files provided"})
		return
	}

	log.Printf("Processing %d files for import", len(files))

	result := &ImportResult{
		Status: "processing",
		Errors: make([]string, 0),
	}

	var allStreamingData []SpotifyStreamingData

	for _, fileHeader := range files {
		log.Printf("Processing file: %s (%.2f MB)", fileHeader.Filename, float64(fileHeader.Size)/1024/1024)

		file, err := fileHeader.Open()
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Failed to open file %s: %v", fileHeader.Filename, err))
			continue
		}
		defer file.Close()

		var fileData []SpotifyStreamingData

		if strings.HasSuffix(fileHeader.Filename, ".zip") {
			fileData, err = h.processZipFile(file, fileHeader.Size)
		} else if strings.HasSuffix(fileHeader.Filename, ".json") {
			fileData, err = h.processJSONFile(file)
		} else {
			result.Errors = append(result.Errors, fmt.Sprintf("Unsupported file format: %s", fileHeader.Filename))
			continue
		}

		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Failed to process file %s: %v", fileHeader.Filename, err))
			continue
		}

		allStreamingData = append(allStreamingData, fileData...)
		result.ProcessedFiles++
	}

	result.ProcessedTracks = len(allStreamingData)
	result.ImportSummary = h.generateSummary(allStreamingData)
	result.ProcessingTime = time.Since(startTime)

	if len(result.Errors) > 0 {
		result.Status = "completed_with_errors"
	} else {
		result.Status = "completed"
	}

	log.Printf("Import completed for user %s: %d files, %d tracks processed in %v",
		userID, result.ProcessedFiles, result.ProcessedTracks, result.ProcessingTime)

	c.JSON(http.StatusOK, result)
}

func (h *ImportHandler) processZipFile(file multipart.File, size int64) ([]SpotifyStreamingData, error) {
	var allData []SpotifyStreamingData

	buffer := make([]byte, size)
	_, err := file.Read(buffer)
	if err != nil {
		return nil, fmt.Errorf("failed to read zip file: %v", err)
	}

	zipReader, err := zip.NewReader(strings.NewReader(string(buffer)), size)
	if err != nil {
		return nil, fmt.Errorf("failed to open zip file: %v", err)
	}

	for _, zipFile := range zipReader.File {
		if !strings.HasSuffix(zipFile.Name, ".json") {
			continue
		}

		log.Printf("Processing JSON file from ZIP: %s", zipFile.Name)

		reader, err := zipFile.Open()
		if err != nil {
			log.Printf("Failed to open file %s in ZIP: %v", zipFile.Name, err)
			continue
		}

		jsonData, err := h.processJSONFile(reader)
		reader.Close()

		if err != nil {
			log.Printf("Failed to process JSON file %s: %v", zipFile.Name, err)
			continue
		}

		allData = append(allData, jsonData...)
	}

	return allData, nil
}

func (h *ImportHandler) processJSONFile(file io.Reader) ([]SpotifyStreamingData, error) {
	var data []SpotifyStreamingData

	decoder := json.NewDecoder(file)
	err := decoder.Decode(&data)
	if err != nil {
		return nil, fmt.Errorf("failed to decode JSON: %v", err)
	}

	validData := make([]SpotifyStreamingData, 0)
	for _, stream := range data {
		if stream.MsPlayed >= 30000 && stream.TrackName != "" && stream.ArtistName != "" {
			validData = append(validData, stream)
		}
	}

	log.Printf("Processed JSON: %d total entries, %d valid streams", len(data), len(validData))
	return validData, nil
}

func (h *ImportHandler) generateSummary(data []SpotifyStreamingData) ImportSummary {
	if len(data) == 0 {
		return ImportSummary{}
	}

	artistCounts := make(map[string]int)
	trackCounts := make(map[string]TrackCount)

	var totalListenTime int64
	var earliestDate, latestDate string

	for i, stream := range data {
		artistCounts[stream.ArtistName]++

		trackKey := fmt.Sprintf("%s - %s", stream.TrackName, stream.ArtistName)
		if existing, exists := trackCounts[trackKey]; exists {
			existing.Count++
			trackCounts[trackKey] = existing
		} else {
			trackCounts[trackKey] = TrackCount{
				Name:   stream.TrackName,
				Artist: stream.ArtistName,
				Count:  1,
			}
		}

		totalListenTime += int64(stream.MsPlayed)

		if i == 0 {
			earliestDate = stream.Timestamp
			latestDate = stream.Timestamp
		} else {
			if stream.Timestamp < earliestDate {
				earliestDate = stream.Timestamp
			}
			if stream.Timestamp > latestDate {
				latestDate = stream.Timestamp
			}
		}
	}

	topArtists := make([]ArtistCount, 0)
	for artist, count := range artistCounts {
		topArtists = append(topArtists, ArtistCount{Name: artist, Count: count})
	}

	for i := 0; i < len(topArtists)-1; i++ {
		for j := i + 1; j < len(topArtists); j++ {
			if topArtists[j].Count > topArtists[i].Count {
				topArtists[i], topArtists[j] = topArtists[j], topArtists[i]
			}
		}
	}
	if len(topArtists) > 10 {
		topArtists = topArtists[:10]
	}

	topTracks := make([]TrackCount, 0)
	for _, track := range trackCounts {
		topTracks = append(topTracks, track)
	}

	for i := 0; i < len(topTracks)-1; i++ {
		for j := i + 1; j < len(topTracks); j++ {
			if topTracks[j].Count > topTracks[i].Count {
				topTracks[i], topTracks[j] = topTracks[j], topTracks[i]
			}
		}
	}
	if len(topTracks) > 10 {
		topTracks = topTracks[:10]
	}

	return ImportSummary{
		TotalStreams:    len(data),
		UniqueArtists:   len(artistCounts),
		UniqueTracks:    len(trackCounts),
		TotalListenTime: totalListenTime,
		DateRange: DateRange{
			From: earliestDate,
			To:   latestDate,
		},
		TopArtists: topArtists,
		TopTracks:  topTracks,
	}
}

func (h *ImportHandler) saveToDatabase(userID string, data []SpotifyStreamingData) error {
	if h.db == nil {
		return fmt.Errorf("database connection is nil")
	}

	log.Printf("Starting database save for user %s with %d records", userID, len(data))

	tx, err := h.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %v", err)
	}
	defer tx.Rollback()

	artistsInserted := make(map[string]bool)
	albumsInserted := make(map[string]bool)
	tracksInserted := make(map[string]bool)

	insertArtistStmt, err := tx.Prepare(`
		INSERT INTO artists (id, name, genres, popularity, image_url) 
		VALUES ($1, $2, $3, $4, $5) 
		ON CONFLICT (id) DO NOTHING
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare artist statement: %v", err)
	}
	defer insertArtistStmt.Close()

	insertAlbumStmt, err := tx.Prepare(`
		INSERT INTO albums (id, name, release_date, image_url) 
		VALUES ($1, $2, $3, $4) 
		ON CONFLICT (id) DO NOTHING
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare album statement: %v", err)
	}
	defer insertAlbumStmt.Close()

	insertTrackStmt, err := tx.Prepare(`
		INSERT INTO tracks (id, name, album_id, duration_ms, popularity, preview_url, isrc) 
		VALUES ($1, $2, $3, $4, $5, $6, $7) 
		ON CONFLICT (id) DO NOTHING
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare track statement: %v", err)
	}
	defer insertTrackStmt.Close()

	insertTrackArtistStmt, err := tx.Prepare(`
		INSERT INTO track_artists (track_id, artist_id) 
		VALUES ($1, $2) 
		ON CONFLICT (track_id, artist_id) DO NOTHING
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare track_artists statement: %v", err)
	}
	defer insertTrackArtistStmt.Close()

	insertListeningHistoryStmt, err := tx.Prepare(`
		INSERT INTO listening_history (user_id, track_id, played_at, context_type, context_uri) 
		VALUES ($1, $2, $3, $4, $5)
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare listening_history statement: %v", err)
	}
	defer insertListeningHistoryStmt.Close()

	for _, stream := range data {
		trackID := h.extractTrackIDFromURI(stream.SpotifyTrackURI)
		artistID := h.generateArtistID(stream.ArtistName)
		albumID := h.generateAlbumID(stream.AlbumName)

		playedAt, err := time.Parse("2006-01-02 15:04", stream.Timestamp)
		if err != nil {
			log.Printf("Failed to parse timestamp %s: %v", stream.Timestamp, err)
			continue
		}

		if !artistsInserted[artistID] && stream.ArtistName != "" {
			_, err = insertArtistStmt.Exec(artistID, stream.ArtistName, pq.Array([]string{}), 0, nil)
			if err != nil {
				log.Printf("Failed to insert artist %s: %v", stream.ArtistName, err)
			} else {
				artistsInserted[artistID] = true
			}
		}

		if !albumsInserted[albumID] && stream.AlbumName != "" {
			_, err = insertAlbumStmt.Exec(albumID, stream.AlbumName, nil, nil)
			if err != nil {
				log.Printf("Failed to insert album %s: %v", stream.AlbumName, err)
			} else {
				albumsInserted[albumID] = true
			}
		}

		if !tracksInserted[trackID] && stream.TrackName != "" {
			var albumIDForTrack *string
			if stream.AlbumName != "" {
				albumIDForTrack = &albumID
			}

			_, err = insertTrackStmt.Exec(trackID, stream.TrackName, albumIDForTrack, 0, 0, nil, nil)
			if err != nil {
				log.Printf("Failed to insert track %s: %v", stream.TrackName, err)
			} else {
				tracksInserted[trackID] = true
			}
		}

		if trackID != "" && artistID != "" {
			_, err = insertTrackArtistStmt.Exec(trackID, artistID)
			if err != nil {
				log.Printf("Failed to insert track-artist relationship: %v", err)
			}
		}

		if trackID != "" {
			contextType := "unknown"
			if stream.ReasonStart != "" {
				contextType = stream.ReasonStart
			}

			_, err = insertListeningHistoryStmt.Exec(userID, trackID, playedAt, contextType, stream.SpotifyTrackURI)
			if err != nil {
				log.Printf("Failed to insert listening history: %v", err)
			}
		}
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %v", err)
	}

	log.Printf("Successfully saved data to database: %d artists, %d albums, %d tracks, %d history records",
		len(artistsInserted), len(albumsInserted), len(tracksInserted), len(data))

	return nil
}

func (h *ImportHandler) extractTrackIDFromURI(uri string) string {
	if uri == "" {
		return ""
	}

	parts := strings.Split(uri, ":")
	if len(parts) >= 3 && parts[0] == "spotify" && parts[1] == "track" {
		return parts[2]
	}

	return ""
}

func (h *ImportHandler) generateArtistID(artistName string) string {
	if artistName == "" {
		return ""
	}

	return fmt.Sprintf("artist_%s", strings.ReplaceAll(strings.ToLower(artistName), " ", "_"))
}

func (h *ImportHandler) generateAlbumID(albumName string) string {
	if albumName == "" {
		return ""
	}

	return fmt.Sprintf("album_%s", strings.ReplaceAll(strings.ToLower(albumName), " ", "_"))
}
