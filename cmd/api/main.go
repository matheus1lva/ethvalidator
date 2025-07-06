package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/matheus/eth-validator-api/internal/api/handlers"
	"github.com/matheus/eth-validator-api/internal/api/middleware"
	"github.com/matheus/eth-validator-api/internal/config"
	"github.com/matheus/eth-validator-api/internal/service"
	"github.com/matheus/eth-validator-api/pkg/cache"
	"github.com/matheus/eth-validator-api/pkg/ethereum"
	"github.com/matheus/eth-validator-api/pkg/logger"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	if err := godotenv.Load(); err != nil {
		fmt.Printf("Warning: .env file not found\n")
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	log := logger.New(cfg.LogLevel)

	log.Info().
		Str("version", version).
		Str("commit", commit).
		Str("date", date).
		Msg("starting eth-validator-api")

	ethClient, err := ethereum.NewClient(cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create ethereum client")
	}

	memCache := cache.NewMemoryCache(cfg.Cache.TTL, cfg.Cache.MaxSize)
	defer memCache.Close()

	validatorService, err := service.NewValidatorService(ethClient, log, memCache)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create validator service")
	}

	validatorHandler, err := handlers.NewValidatorHandler(validatorService, log)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create validator handler")
	}

	healthHandler := handlers.NewHealthHandler(version)

	mux := http.NewServeMux()

	mux.HandleFunc("/health", healthHandler.Health)
	mux.HandleFunc("/ready", healthHandler.Ready)

	mux.HandleFunc("/blockreward/", validatorHandler.GetBlockReward)
	mux.HandleFunc("/syncduties/", validatorHandler.GetSyncDuties)

	if cfg.Metrics.Enabled {
		mux.Handle("/metrics", promhttp.Handler())
	}

	handler := middleware.RequestID(
		middleware.Logging(log)(
			middleware.Recovery(log)(
				middleware.Metrics(
					middleware.CORS(
						middleware.Timeout(cfg.Request.Timeout)(mux),
					),
				),
			),
		),
	)

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Info().Str("port", cfg.Port).Msg("starting HTTP server")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("failed to start server")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info().Msg("shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal().Err(err).Msg("server forced to shutdown")
	}

	log.Info().Msg("server exited")
}
