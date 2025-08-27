package handlers

import (
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"musike-backend/internal/services"
)

type AuthHandler struct {
	authService    *services.AuthService
	spotifyService *services.SpotifyService
	processedCodes map[string]bool
	codesMutex     sync.RWMutex
}

func NewAuthHandler(authService *services.AuthService, spotifyService *services.SpotifyService) *AuthHandler {
	return &AuthHandler{
		authService:    authService,
		spotifyService: spotifyService,
		processedCodes: make(map[string]bool),
		codesMutex:     sync.RWMutex{},
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

	// Verificar se o código já foi processado
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

	// Trocar código por token
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

	// Buscar perfil do usuário no Spotify
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

	// Gerar JWT token
	log.Printf("Generating JWT token for user: %s", user.ID)
	jwtToken, err := h.authService.GenerateJWT(user.ID)
	if err != nil {
		log.Printf("Failed to generate JWT token: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to generate JWT token",
			"details": err.Error(),
		})
		return
	}

	// Aqui você salvaria o token do Spotify no banco/Redis para uso posterior
	// Para este exemplo, vamos retornar tudo

	log.Printf("Authentication successful for user: %s (%s)", user.DisplayName, user.ID)

	c.JSON(http.StatusOK, gin.H{
		"access_token":  jwtToken,
		"user":          user,
		"spotify_token": token.AccessToken,
		"expires_in":    token.Expiry,
		"message":       "Authentication successful",
	})
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
