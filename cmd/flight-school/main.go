package main

import (
	"context"
	"errors"
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
	"github.com/cdriehuys/flight-school/internal/cli"
	"github.com/cdriehuys/flight-school/migrations"
	"github.com/cdriehuys/flight-school/static"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const addr = ":8000"

func executeCLI(logStream io.Writer, migrationFS fs.FS) error {
	cmd := &cobra.Command{
		Use:   "flight-school",
		Short: "Run the flight-school web server",
		RunE:  webServerRunner(logStream),
	}

	cmd.Flags().Bool("debug", false, "Enable debug behavior")
	viper.BindPFlag("debug", cmd.Flags().Lookup("debug"))

	cmd.PersistentFlags().String("dsn", "", "DSN for connecting to the database ($FLIGHT_SCHOOL_DSN)")
	viper.BindEnv("dsn", "FLIGHT_SCHOOL_DSN")
	viper.BindPFlag("dsn", cmd.PersistentFlags().Lookup("dsn"))
	viper.SetDefault("dsn", "postgres://localhost")

	cmd.AddCommand(cli.NewMigrateCmd(migrationFS))

	return cmd.Execute()
}

func webServerRunner(logStream io.Writer) func(*cobra.Command, []string) error {
	return func(c *cobra.Command, s []string) error {
		return run(logStream)
	}
}

func run(logStream io.Writer) error {
	debug := viper.GetBool("debug")
	dsn := viper.GetString("dsn")

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

	var staticFiles fs.FS
	if debug {
		staticFiles = os.DirFS("./static")
	} else {
		staticFiles = static.Files
	}

	db, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %v", err)
	}

	defer db.Close()

	app, err := app.New(
		logger,
		templateFiles,
		staticFiles,
		db,
		&app.Options{Debug: debug, LiveTemplates: debug},
	)
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
	logger.Info("Web server stopped.")

	logger.Info("Closing database connection.")
	db.Close()
	logger.Info("Database connection closed.")

	logger.Info("Shutdown complete.")

	return nil
}

func main() {
	if err := executeCLI(os.Stderr, migrations.Files); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}
