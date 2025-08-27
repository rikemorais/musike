package config

import (
	"os"
)

type Config struct {
	SpotifyClientID     string
	SpotifyClientSecret string
	SpotifyRedirectURL  string
	JWTSecret           string
	DatabaseURL         string
	RedisURL            string
	Port                string
	SSLCertPath         string
	SSLKeyPath          string
	UseHTTPS            bool
}

func Load() *Config {
	return &Config{
		SpotifyClientID:     getEnv("SPOTIFY_CLIENT_ID", ""),
		SpotifyClientSecret: getEnv("SPOTIFY_CLIENT_SECRET", ""),
		SpotifyRedirectURL:  getEnv("SPOTIFY_REDIRECT_URL", "https://localhost:3000/callback"),
		JWTSecret:           getEnv("JWT_SECRET", "your-secret-key"),
		DatabaseURL:         getEnv("DATABASE_URL", "postgres://user:password@localhost/musike?sslmode=disable"),
		RedisURL:            getEnv("REDIS_URL", "redis://localhost:6379"),
		Port:                getEnv("PORT", "8080"),
		SSLCertPath:         getEnv("SSL_CERT_PATH", "./certs/cert.pem"),
		SSLKeyPath:          getEnv("SSL_KEY_PATH", "./certs/key.pem"),
		UseHTTPS:            getEnv("USE_HTTPS", "true") == "true",
	}
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
