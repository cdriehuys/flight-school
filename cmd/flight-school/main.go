package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/cdriehuys/flight-school/internal/app"
)

const addr = ":8000"

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
	app := app.New(logger)

	logger.Info("Starting server", "address", addr)

	s := http.Server{
		Addr:    addr,
		Handler: app.Routes(),
	}

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	go func() {
		if err := s.ListenAndServe(); err != nil {
			if !errors.Is(err, http.ErrServerClosed) {
				logger.Error("Unexpected server error.", "error", err)
			}
		}
	}()

	<-interrupt
	signal.Stop(interrupt)

	logger.Info("Received interrupt, shutting down.")

	shutdownContext, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := s.Shutdown(shutdownContext); err != nil {
		logger.Error("Server did not shut down gracefully.", "error", err)
	}

	logger.Info("Shutdown complete.")
}
