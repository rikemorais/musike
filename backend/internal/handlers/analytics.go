package handlers

import (
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2"
	"musike-backend/internal/services"
)

type AnalyticsHandler struct {
	analyticsService *services.AnalyticsService
	spotifyService   *services.SpotifyService
}

func NewAnalyticsHandler(analyticsService *services.AnalyticsService, spotifyService *services.SpotifyService) *AnalyticsHandler {
	return &AnalyticsHandler{
		analyticsService: analyticsService,
		spotifyService:   spotifyService,
	}
}

func (h *AnalyticsHandler) GetUserProfile(c *gin.Context) {
	spotifyToken := c.GetHeader("Spotify-Token")
	if spotifyToken == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Spotify token required"})
		return
	}

	token := &oauth2.Token{AccessToken: spotifyToken}

	user, err := h.spotifyService.GetUserProfile(token)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user profile"})
		return
	}

	c.JSON(http.StatusOK, user)
}

func (h *AnalyticsHandler) GetTopTracks(c *gin.Context) {
	spotifyToken := c.GetHeader("Spotify-Token")
	if spotifyToken == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Spotify token required"})
		return
	}

	timeRange := c.DefaultQuery("time_range", "medium_term")
	limitStr := c.DefaultQuery("limit", "20")

	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		limit = 20
	}

	token := &oauth2.Token{AccessToken: spotifyToken}

	tracks, err := h.spotifyService.GetTopTracks(token, timeRange, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get top tracks"})
		return
	}

	c.JSON(http.StatusOK, tracks)
}

func (h *AnalyticsHandler) GetTopArtists(c *gin.Context) {
	spotifyToken := c.GetHeader("Spotify-Token")
	if spotifyToken == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Spotify token required"})
		return
	}

	timeRange := c.DefaultQuery("time_range", "medium_term")
	limitStr := c.DefaultQuery("limit", "20")

	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		limit = 20
	}

	token := &oauth2.Token{AccessToken: spotifyToken}

	artists, err := h.spotifyService.GetTopArtists(token, timeRange, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get top artists"})
		return
	}

	c.JSON(http.StatusOK, artists)
}

func (h *AnalyticsHandler) GetListeningHistory(c *gin.Context) {
	spotifyToken := c.GetHeader("Spotify-Token")
	if spotifyToken == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Spotify token required"})
		return
	}

	limitStr := c.DefaultQuery("limit", "50")

	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		limit = 50
	}

	token := &oauth2.Token{AccessToken: spotifyToken}

	history, err := h.spotifyService.GetRecentlyPlayed(token, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get listening history"})
		return
	}

	c.JSON(http.StatusOK, history)
}

func (h *AnalyticsHandler) GetUserAnalytics(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found"})
		return
	}

	spotifyToken := c.GetHeader("Spotify-Token")
	if spotifyToken == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Spotify token required"})
		return
	}

	// Parâmetros de filtro de tempo
	timeFilter := c.DefaultQuery("time_filter", "6months") // 6months, 1year, alltime

	token := &oauth2.Token{AccessToken: spotifyToken}

	analytics, err := h.analyticsService.GenerateUserAnalytics(userID.(string), timeFilter, h.spotifyService, token)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate analytics"})
		return
	}

	c.JSON(http.StatusOK, analytics)
}

func (h *AnalyticsHandler) GetRecommendations(c *gin.Context) {
	spotifyToken := c.GetHeader("Spotify-Token")
	if spotifyToken == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Spotify token required"})
		return
	}

	seedArtists := []string{"4NHQUGzhtTLFvgF5SZesLK"} // Tame Impala
	seedTracks := []string{"5ghIJDpPoe3CfHMGu71E6T"}  // Blinding Lights

	token := &oauth2.Token{AccessToken: spotifyToken}

	recommendations, err := h.spotifyService.GetRecommendations(token, seedArtists, seedTracks)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get recommendations"})
		return
	}

	c.JSON(http.StatusOK, recommendations)
}

func (h *AnalyticsHandler) GetRecentlyPlayed(c *gin.Context) {
	spotifyToken := c.GetHeader("Spotify-Token")
	if spotifyToken == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Spotify token required"})
		return
	}

	limitStr := c.DefaultQuery("limit", "5")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 || limit > 50 {
		limit = 5
	}

	token := &oauth2.Token{AccessToken: spotifyToken}

	recentTracks, err := h.spotifyService.GetRecentlyPlayed(token, limit)
	if err != nil {
		log.Printf("Error getting recently played tracks: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get recently played tracks"})
		return
	}

	c.JSON(http.StatusOK, recentTracks)
}
