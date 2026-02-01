package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/HerbHall/netvantage/internal/dispatch"
	"github.com/HerbHall/netvantage/internal/gateway"
	"github.com/HerbHall/netvantage/internal/plugin"
	"github.com/HerbHall/netvantage/internal/pulse"
	"github.com/HerbHall/netvantage/internal/recon"
	"github.com/HerbHall/netvantage/internal/server"
	"github.com/HerbHall/netvantage/internal/vault"
	"go.uber.org/zap"
)

func main() {
	configPath := flag.String("config", "", "path to configuration file")
	flag.Parse()

	// Initialize logger
	logger, err := zap.NewProduction()
	if err != nil {
		os.Exit(1)
	}
	defer logger.Sync()

	logger.Info("NetVantage server starting")

	// Load configuration
	config, err := server.LoadConfig(*configPath)
	if err != nil {
		logger.Fatal("failed to load configuration", zap.Error(err))
	}

	// Create plugin registry
	registry := plugin.NewRegistry(logger)

	// Register all plugins (compile-time composition)
	plugins := []plugin.Plugin{
		recon.New(),
		pulse.New(),
		dispatch.New(),
		vault.New(),
		gateway.New(),
	}
	for _, p := range plugins {
		if err := registry.Register(p); err != nil {
			logger.Fatal("failed to register plugin", zap.Error(err))
		}
	}

	// Initialize all plugins
	if err := registry.InitAll(config); err != nil {
		logger.Fatal("failed to initialize plugins", zap.Error(err))
	}

	// Start plugins
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := registry.StartAll(ctx); err != nil {
		logger.Fatal("failed to start plugins", zap.Error(err))
	}

	// Create and start HTTP server
	addr := config.GetString("server.host") + ":" + config.GetString("server.port")
	if addr == ":" {
		addr = "0.0.0.0:8080"
	}
	srv := server.New(addr, registry, logger)

	// Start server in background
	go func() {
		if err := srv.Start(); err != nil {
			logger.Fatal("server error", zap.Error(err))
		}
	}()

	logger.Info("NetVantage server ready", zap.String("addr", addr))

	// Wait for shutdown signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigCh

	logger.Info("received shutdown signal", zap.String("signal", sig.String()))

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	registry.StopAll()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("server shutdown error", zap.Error(err))
	}

	logger.Info("NetVantage server stopped")
}
