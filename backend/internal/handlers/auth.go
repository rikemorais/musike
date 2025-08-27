package handlers

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"musike-backend/internal/services"
)

type AuthHandler struct {
	authService     *services.AuthService
	spotifyService  *services.SpotifyService
	trackingService *services.TrackingService
	db              *sql.DB
	processedCodes  map[string]bool
	codesMutex      sync.RWMutex
}

func NewAuthHandler(authService *services.AuthService, spotifyService *services.SpotifyService, db *sql.DB, trackingService *services.TrackingService) *AuthHandler {
	return &AuthHandler{
		authService:     authService,
		spotifyService:  spotifyService,
		trackingService: trackingService,
		db:              db,
		processedCodes:  make(map[string]bool),
		codesMutex:      sync.RWMutex{},
	}
}

func (h *AuthHandler) SpotifyAuth(c *gin.Context) {
	state := c.Query("state")
	if state == "" {
		state = "musike-" + strconv.FormatInt(time.Now().Unix(), 10) // State único baseado em timestamp
	}

	authURL := h.authService.GetAuthURL(state)

	log.Printf("Generated Spotify auth URL: %s", authURL)

	c.JSON(http.StatusOK, gin.H{
		"auth_url": authURL,
		"state":    state,
	})
}

func (h *AuthHandler) SpotifyCallback(c *gin.Context) {
	code := c.Query("code")
	error := c.Query("error")
	state := c.Query("state")

	log.Printf("Spotify callback received - Code: %s, Error: %s, State: %s",
		code != "", error, state)

	if code != "" {
		h.codesMutex.Lock()
		if h.processedCodes[code] {
			h.codesMutex.Unlock()
			log.Printf("Code already processed, ignoring duplicate request")
			c.JSON(http.StatusOK, gin.H{
				"message": "Already processed",
				"status":  "duplicate_request_ignored",
			})
			return
		}
		h.processedCodes[code] = true
		h.codesMutex.Unlock()
	}

	if error != "" {
		log.Printf("Spotify auth error: %s", error)
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Spotify authorization failed: " + error,
			"details": "User denied access or authorization failed",
		})
		return
	}

	if code == "" {
		log.Printf("No authorization code received from Spotify")
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Authorization code not provided",
			"details": "The callback did not include a valid authorization code",
		})
		return
	}

	log.Printf("Exchanging code for token...")
	token, err := h.authService.ExchangeCode(code)
	if err != nil {
		log.Printf("Failed to exchange code for token: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to exchange code for token",
			"details": err.Error(),
		})
		return
	}

	log.Printf("Getting user profile from Spotify...")
	user, err := h.spotifyService.GetUserProfile(token)
	if err != nil {
		log.Printf("Failed to get user profile: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get user profile",
			"details": err.Error(),
		})
		return
	}

	// Create or get user from database
	dbUserID, err := h.createOrGetUser(user)
	if err != nil {
		log.Printf("Failed to create/get user in database: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to save user data",
			"details": err.Error(),
		})
		return
	}

	log.Printf("Generating JWT token for user: %s (DB ID: %s)", user.ID, dbUserID)
	jwtToken, err := h.authService.GenerateJWT(dbUserID)
	if err != nil {
		log.Printf("Failed to generate JWT token: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to generate JWT token",
			"details": err.Error(),
		})
		return
	}

	log.Printf("Authentication successful for user: %s (%s)", user.DisplayName, user.ID)

	// Auto-start tracking for the user
	if h.trackingService != nil {
		err = h.trackingService.StartTracking(dbUserID, token.AccessToken)
		if err != nil {
			log.Printf("Warning: Failed to auto-start tracking for user %s: %v", dbUserID, err)
		} else {
			log.Printf("Auto-started tracking for user: %s", dbUserID)
		}
	}

	frontendURL := "http://localhost:3001/callback"
	redirectURL := frontendURL + "?access_token=" + jwtToken + "&spotify_token=" + token.AccessToken + "&user_id=" + user.ID

	c.Redirect(http.StatusFound, redirectURL)
}

func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var request struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	token, err := h.authService.RefreshSpotifyToken(request.RefreshToken)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to refresh token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token": token.AccessToken,
		"expires_in":   token.Expiry,
	})
}

func (h *AuthHandler) createOrGetUser(spotifyUser *services.SpotifyUser) (string, error) {
	if h.db == nil {
		return "", fmt.Errorf("database not available")
	}

	ctx := context.Background()

	// Try to get existing user first
	var dbUserID string
	err := h.db.QueryRowContext(ctx, `
		SELECT id FROM users WHERE spotify_id = $1
	`, spotifyUser.ID).Scan(&dbUserID)

	if err == sql.ErrNoRows {
		// User doesn't exist, create new one
		followers := spotifyUser.Followers.Total

		var imageURL string
		if len(spotifyUser.Images) > 0 {
			imageURL = spotifyUser.Images[0].URL
		}

		err = h.db.QueryRowContext(ctx, `
			INSERT INTO users (spotify_id, display_name, email, country, followers_count, profile_image_url) 
			VALUES ($1, $2, $3, $4, $5, $6) 
			RETURNING id
		`, spotifyUser.ID, spotifyUser.DisplayName, spotifyUser.Email, spotifyUser.Country, followers, imageURL).Scan(&dbUserID)

		if err != nil {
			return "", fmt.Errorf("failed to create user: %v", err)
		}

		log.Printf("Created new user in database: %s -> %s", spotifyUser.ID, dbUserID)
	} else if err != nil {
		return "", fmt.Errorf("failed to query user: %v", err)
	} else {
		log.Printf("Found existing user in database: %s -> %s", spotifyUser.ID, dbUserID)
	}

	return dbUserID, nil
}
