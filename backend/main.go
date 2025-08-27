package main

import (
	"crypto/tls"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"musike-backend/internal/config"
	"musike-backend/internal/database"
	"musike-backend/internal/handlers"
	"musike-backend/internal/middleware"
	"musike-backend/internal/services"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found")
	}

	cfg := config.Load()

	db, err := database.Connect(cfg)
	if err != nil {
		log.Printf("Warning: Failed to connect to database: %v", err)
		log.Println("Continuing without database-dependent features...")
	}
	if db != nil {
		defer db.Close()
	}

	spotifyService := services.NewSpotifyService(cfg)
	authService := services.NewAuthService(cfg)
	analyticsService := services.NewAnalyticsService(cfg, db)

	var trackingService *services.TrackingService
	var trackingHandler *handlers.TrackingHandler
	if db != nil {
		trackingService = services.NewTrackingService(cfg, db)
		trackingHandler = handlers.NewTrackingHandler(trackingService, authService)

		go trackingService.StartPeriodicTracking()
		log.Println("🎵 Spotify tracking service started")
	} else {
		log.Println("⚠️  Database not available - tracking service disabled")
	}

	authHandler := handlers.NewAuthHandler(authService, spotifyService, db, trackingService)
	analyticsHandler := handlers.NewAnalyticsHandler(analyticsService, spotifyService)
	importHandler := handlers.NewImportHandler(db)

	r := gin.Default()

	r.Use(middleware.CORS())
	r.Use(middleware.Logger())

	r.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "Musike Backend API",
			"status":  "running",
			"version": "v1.0.0",
			"endpoints": map[string]string{
				"auth":     "/api/v1/auth/spotify",
				"callback": "/api/v1/auth/callback",
				"frontend": "Run frontend separately on different port",
			},
		})
	})

	r.GET("/callback", authHandler.SpotifyCallback)

	public := r.Group("/api/v1")
	{
		public.GET("/auth/spotify", authHandler.SpotifyAuth)
		public.GET("/auth/callback", authHandler.SpotifyCallback)
		public.POST("/auth/refresh", authHandler.RefreshToken)
	}

	protected := r.Group("/api/v1")
	protected.Use(middleware.Auth(authService))
	{
		protected.GET("/user/profile", analyticsHandler.GetUserProfile)
		protected.GET("/user/top-tracks", analyticsHandler.GetTopTracks)
		protected.GET("/user/top-artists", analyticsHandler.GetTopArtists)
		protected.GET("/user/listening-history", analyticsHandler.GetListeningHistory)
		protected.GET("/user/recently-played", analyticsHandler.GetRecentlyPlayed)
		protected.GET("/user/analytics", analyticsHandler.GetUserAnalytics)
		protected.GET("/user/recommendations", analyticsHandler.GetRecommendations)

		protected.POST("/import/spotify", importHandler.ImportSpotifyData)

		if trackingHandler != nil {
			protected.POST("/tracking/start", trackingHandler.StartTracking)
			protected.POST("/tracking/stop", trackingHandler.StopTracking)
			protected.GET("/tracking/current", trackingHandler.GetCurrentTrack)
			protected.GET("/tracking/status", trackingHandler.GetTrackingStatus)
			protected.GET("/tracking/history", trackingHandler.GetRecentListeningHistory)
		}
	}

	// Rota pública para sync forçado (apenas para debug)
	public.POST("/tracking/force-sync/:userID", func(c *gin.Context) {
		if trackingHandler != nil {
			trackingHandler.ForceFullSync(c)
		} else {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Tracking service not available"})
		}
	})

	if cfg.UseHTTPS {
		log.Printf("🔐 Starting HTTPS server on port %s", cfg.Port)

		tlsConfig := &tls.Config{
			MinVersion: tls.VersionTLS12,
		}

		server := &http.Server{
			Addr:      ":" + cfg.Port,
			Handler:   r,
			TLSConfig: tlsConfig,
		}

		log.Fatal(server.ListenAndServeTLS(cfg.SSLCertPath, cfg.SSLKeyPath))
	} else {
		log.Printf("🌐 Starting HTTP server on port %s", cfg.Port)
		log.Fatal(r.Run(":" + cfg.Port))
	}
}
