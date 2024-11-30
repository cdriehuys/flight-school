package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/cdriehuys/flight-school/html"
	"github.com/cdriehuys/flight-school/internal/app"
)

var debug bool

const addr = ":8000"

func run(logStream io.Writer) error {
	flag.BoolVar(&debug, "debug", false, "enable debug behavior")
	flag.Parse()

	logLevel := slog.LevelInfo
	if debug {
		logLevel = slog.LevelDebug
	}

	logger := slog.New(slog.NewTextHandler(logStream, &slog.HandlerOptions{Level: logLevel}))

	var templateFiles fs.FS
	if debug {
		templateFiles = os.DirFS("./html")
	} else {
		templateFiles = html.Files
	}

	app, err := app.New(logger, templateFiles, &app.Options{Debug: debug, LiveTemplates: debug})
	if err != nil {
		return fmt.Errorf("failed to build app: %v", err)
	}

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

	return nil
}

func main() {
	if err := run(os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}
