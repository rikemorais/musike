package handlers

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"musike-backend/internal/services"
)

type TrackingHandler struct {
	trackingService *services.TrackingService
	authService     *services.AuthService
}

func NewTrackingHandler(trackingService *services.TrackingService, authService *services.AuthService) *TrackingHandler {
	return &TrackingHandler{
		trackingService: trackingService,
		authService:     authService,
	}
}

func (h *TrackingHandler) StartTracking(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	spotifyToken := c.GetHeader("Spotify-Token")
	if spotifyToken == "" {
		spotifyToken = c.PostForm("spotify_token")
		if spotifyToken == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Spotify token required"})
			return
		}
	}

	err := h.trackingService.StartTracking(userID.(string), spotifyToken)
	if err != nil {
		log.Printf("Error starting tracking for user %s: %v", userID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start tracking"})
		return
	}

	log.Printf("Started tracking for user: %s", userID)
	c.JSON(http.StatusOK, gin.H{
		"message": "Tracking started successfully",
		"status":  "active",
	})
}

func (h *TrackingHandler) StopTracking(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	err := h.trackingService.StopTracking(userID.(string))
	if err != nil {
		log.Printf("Error stopping tracking for user %s: %v", userID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to stop tracking"})
		return
	}

	log.Printf("Stopped tracking for user: %s", userID)
	c.JSON(http.StatusOK, gin.H{
		"message": "Tracking stopped successfully",
		"status":  "inactive",
	})
}

func (h *TrackingHandler) GetCurrentTrack(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	spotifyToken := c.GetHeader("Spotify-Token")
	if spotifyToken == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Spotify token required in header"})
		return
	}

	currentTrack, err := h.trackingService.GetCurrentTrack(spotifyToken)
	if err != nil {
		log.Printf("Error getting current track for user %s: %v", userID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get current track"})
		return
	}

	if currentTrack == nil {
		c.JSON(http.StatusOK, gin.H{
			"message": "No track currently playing",
			"track":   nil,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"track": currentTrack,
	})
}

func (h *TrackingHandler) GetTrackingStatus(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"active_users": h.trackingService.GetActiveTrackingCount(),
		"status":       "running",
	})
}

func (h *TrackingHandler) GetRecentListeningHistory(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user_id": userID,
		"history": []interface{}{},
		"message": "Listening history tracking is now active. Check back in a few minutes to see your recent tracks.",
	})
}

func (h *TrackingHandler) ForceFullSync(c *gin.Context) {
	userID := c.Param("userID")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "User ID is required"})
		return
	}

	log.Printf("Force full sync requested for user: %s", userID)

	err := h.trackingService.ForceFullSync(userID)
	if err != nil {
		log.Printf("Error during force full sync for user %s: %v", userID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to sync tracks: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Full sync completed successfully",
		"user_id": userID,
	})
}
