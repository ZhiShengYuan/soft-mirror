package main

import (
	"context"
	"html/template"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"file-host/internal/config"
	"file-host/internal/handler"
	"file-host/internal/logger"
	"file-host/internal/middleware"
	"file-host/internal/storage"
	"file-host/web"
)

func main() {
	// Load config
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	// Setup logger
	if err := logger.Setup(cfg.LogLevel, cfg.LogFormat, cfg.LogFile); err != nil {
		slog.Error("failed to setup logger", "error", err)
		os.Exit(1)
	}

	// Create storage
	store, err := storage.New(cfg.DataDir)
	if err != nil {
		slog.Error("failed to initialize storage", "error", err)
		os.Exit(1)
	}

	// Parse embedded templates
	tmpl, err := template.ParseFS(web.Templates, "templates/*.html")
	if err != nil {
		slog.Error("failed to parse templates", "error", err)
		os.Exit(1)
	}

	// Set Gin release mode
	gin.SetMode(gin.ReleaseMode)

	// Create router
	router := gin.New()

	// Set trusted proxies
	if len(cfg.TrustedProxies) > 0 {
		if err := router.SetTrustedProxies(cfg.TrustedProxies); err != nil {
			slog.Error("failed to set trusted proxies", "error", err)
			os.Exit(1)
		}
	} else {
		if err := router.SetTrustedProxies(nil); err != nil {
			slog.Error("failed to disable trusted proxies", "error", err)
			os.Exit(1)
		}
	}

	// Global middleware
	router.Use(middleware.SecurityHeaders())
	router.Use(middleware.RequestLogger())

	// Create HMAC middleware
	hmacMiddleware := middleware.NewHMACMiddleware(cfg.HMACSecret, cfg.HMACMaxDrift)

	// Start nonce cleanup
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	hmacMiddleware.StartCleanup(ctx)

	// Routes
	router.GET("/healthz", handler.Health)
	router.GET("/", handler.Index(store, tmpl))
	router.GET("/programs/:name", handler.ProgramPage(store, tmpl))

	api := router.Group("/api/v1")
	{
		api.GET("/programs/:name/versions", handler.Versions(store))
		api.GET("/programs/:name/download", handler.AutoDownload(store))
		api.GET("/programs/:name/:version/:os/:arch", handler.DirectDownload(store))

		// Authenticated routes
		api.PUT("/programs/:name/:version/:os/:arch",
			hmacMiddleware.Middleware(),
			middleware.MaxBodySize(cfg.MaxUploadSize),
			handler.Upload(store, cfg.MaxUploadSize),
		)
		api.DELETE("/programs/:name/:version/:os/:arch",
			hmacMiddleware.Middleware(),
			handler.DeleteBinary(store),
		)
		api.DELETE("/programs/:name/:version",
			hmacMiddleware.Middleware(),
			handler.DeleteVersion(store),
		)
	}

	// HTTP server with graceful shutdown
	srv := &http.Server{
		Addr:    cfg.ListenAddr,
		Handler: router,
	}

	go func() {
		slog.Info("server starting", "addr", cfg.ListenAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	// Wait for shutdown signal
	<-ctx.Done()
	slog.Info("shutting down server...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("shutdown error", "error", err)
	}
	slog.Info("server stopped")
}
