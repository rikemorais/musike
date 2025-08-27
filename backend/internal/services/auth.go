package services

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/spotify"
	"musike-backend/internal/config"
)

type AuthService struct {
	config      *config.Config
	oauthConfig *oauth2.Config
}

type Claims struct {
	UserID string `json:"user_id"`
	jwt.RegisteredClaims
}

func NewAuthService(cfg *config.Config) *AuthService {
	// Validar configurações críticas
	if cfg.SpotifyClientID == "" {
		log.Fatal("SPOTIFY_CLIENT_ID is required")
	}
	if cfg.SpotifyClientSecret == "" {
		log.Fatal("SPOTIFY_CLIENT_SECRET is required")
	}
	if cfg.SpotifyRedirectURL == "" {
		log.Fatal("SPOTIFY_REDIRECT_URL is required")
	}

	log.Printf("Initializing Spotify OAuth with Client ID: %s", cfg.SpotifyClientID[:8]+"...")
	log.Printf("Redirect URL configured: %s", cfg.SpotifyRedirectURL)

	oauthConfig := &oauth2.Config{
		ClientID:     cfg.SpotifyClientID,
		ClientSecret: cfg.SpotifyClientSecret,
		RedirectURL:  cfg.SpotifyRedirectURL,
		Scopes: []string{
			"user-read-private",
			"user-read-email",
			"user-top-read",
			"user-read-recently-played",
			"user-library-read",
			"playlist-read-private",
			"user-read-playback-state",
			"user-read-currently-playing",
		},
		Endpoint: spotify.Endpoint,
	}

	return &AuthService{
		config:      cfg,
		oauthConfig: oauthConfig,
	}
}

func (a *AuthService) GetAuthURL(state string) string {
	authURL := a.oauthConfig.AuthCodeURL(state, oauth2.AccessTypeOffline)
	log.Printf("Generated auth URL: %s", authURL)
	return authURL
}

func (a *AuthService) ExchangeCode(code string) (*oauth2.Token, error) {
	log.Printf("Exchanging authorization code for token...")

	token, err := a.oauthConfig.Exchange(context.Background(), code)
	if err != nil {
		log.Printf("Token exchange failed: %v", err)
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}

	log.Printf("Token exchange successful. Expires at: %v", token.Expiry)
	return token, nil
}

func (a *AuthService) GenerateJWT(userID string) (string, error) {
	claims := &Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(a.config.JWTSecret))
}

func (a *AuthService) ValidateToken(tokenString string) (string, error) {
	claims := &Claims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(a.config.JWTSecret), nil
	})

	if err != nil {
		return "", err
	}

	if !token.Valid {
		return "", fmt.Errorf("invalid token")
	}

	return claims.UserID, nil
}

func (a *AuthService) RefreshSpotifyToken(refreshToken string) (*oauth2.Token, error) {
	token := &oauth2.Token{
		RefreshToken: refreshToken,
	}

	tokenSource := a.oauthConfig.TokenSource(context.Background(), token)
	return tokenSource.Token()
}
