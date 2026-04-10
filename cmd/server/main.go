package main

import (
	"context"
	"encoding/json"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	greenapi "green-api-client"
	"green-api-client/internal/handler"
	"green-api-client/internal/middleware"
	"green-api-client/pkg/logger/sl"
	"green-api-client/pkg/logger/slogpretty"
)

func main() {
	os.Exit(run())
}

func run() int {
	env := getEnv("ENV", "local")
	port := getEnv("PORT", "8080")

	log := slogpretty.SetupLogger(env)

	h := handler.New(log)

	mux := http.NewServeMux()

	// API-роуты
	mux.HandleFunc("POST /api/getSettings", h.GetSettings)
	mux.HandleFunc("POST /api/getStateInstance", h.GetStateInstance)
	mux.HandleFunc("POST /api/sendMessage", h.SendMessage)
	mux.HandleFunc("POST /api/sendFileByUrl", h.SendFileByURL)

	// healthcheck
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	// статика из embed.FS
	staticContent, err := fs.Sub(greenapi.StaticFS, "static")
	if err != nil {
		log.Error("failed to create static sub-FS", sl.Err(err))
		return 1
	}
	mux.Handle("/", http.FileServer(http.FS(staticContent)))

	// rate-limiter middleware поверх всего mux
	rateLimitHandler := middleware.RateLimit(10, 20)(mux)

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: rateLimitHandler,
	}

	// запуск HTTP-сервера в горутине
	serverErr := make(chan error, 1)
	go func() {
		log.Info("server starting", slog.String("port", port), slog.String("env", env))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("HTTP server error", sl.Err(err))
			serverErr <- err
		}
	}()

	// graceful shutdown по SIGINT/SIGTERM
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-quit:
		log.Info("shutdown signal received")
	case <-serverErr:
		return 1
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Error("server forced to shutdown", sl.Err(err))
		return 1
	}

	log.Info("server stopped")
	return 0
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
