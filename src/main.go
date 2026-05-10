package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/erkannt/rechenschaftspflicht/middlewares"
	"github.com/erkannt/rechenschaftspflicht/services/authentication"
	"github.com/erkannt/rechenschaftspflicht/services/config"
	database "github.com/erkannt/rechenschaftspflicht/services/db"
	"github.com/erkannt/rechenschaftspflicht/services/eventstore"
	"github.com/erkannt/rechenschaftspflicht/services/userstore"
	"github.com/julienschmidt/httprouter"
	sloghttp "github.com/samber/slog-http"
)

func run(
	ctx context.Context,
	stdout io.Writer,
	getenv func(string) string,
) error {
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Setup dependencies
	logger := slog.New(slog.NewJSONHandler(stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	cfg, err := config.LoadFromEnv(getenv)
	if err != nil {
		return fmt.Errorf("could not load config from env: %w", err)
	}

	db, err := database.InitDB(cfg)
	if err != nil {
		return fmt.Errorf("could not init database: %w", err)
	}

	eventStore := eventstore.NewEventStore(db)
	userStore := userstore.NewUserStore(db)
	auth := authentication.New(logger, cfg)

	// Create server
	router := httprouter.New()
	addRoutes(router, cfg, eventStore, userStore, auth)
	requestLogging := sloghttp.New(logger)
	handlerWithMiddlewares := middlewares.TemplCSSWithNonce(middlewares.SecurityHeaders(requestLogging(router)))

	srv := &http.Server{Addr: ":8080", Handler: handlerWithMiddlewares}

	// Start the server
	serverErr := make(chan error, 1)
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErr <- err
		} else {
			serverErr <- nil
		}
	}()
	logger.Info("Server is listening on :8080")

	// Graceful shutdown
	select {
	case <-ctx.Done():
		logger.Info("Shutting down server...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("server forced to shutdown: %w", err)
		}
		logger.Info("Server stopped")
		return nil
	case err := <-serverErr:
		if err != nil {
			return fmt.Errorf("listen and serve: %w", err)
		}
		logger.Info("Server stopped")
		return nil
	}
}

func main() {
	ctx := context.Background()
	if err := run(ctx, os.Stdout, os.Getenv); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}
