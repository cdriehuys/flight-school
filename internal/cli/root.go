package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/cdriehuys/flight-school/html"
	"github.com/cdriehuys/flight-school/internal/app"
	"github.com/cdriehuys/flight-school/static"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func NewRootCmd(logStream io.Writer, acsDocs fs.FS, migrationFS fs.FS) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "flight-school",
		Short: "Run the flight-school web server",
		RunE:  webServerRunner(logStream),
	}

	cmd.PersistentFlags().Bool("debug", false, "Enable debug logging")
	viper.BindPFlag("debug", cmd.PersistentFlags().Lookup("debug"))

	cmd.PersistentFlags().String("dsn", "", "DSN for connecting to the database ($FLIGHT_SCHOOL_DSN)")
	viper.BindEnv("dsn", "FLIGHT_SCHOOL_DSN")
	viper.BindPFlag("dsn", cmd.PersistentFlags().Lookup("dsn"))
	viper.SetDefault("dsn", "postgres://localhost")

	cmd.Flags().String("static-dir", "", "Use static files from this path instead of the embedded files")
	viper.BindPFlag("static-dir", cmd.Flags().Lookup("static-dir"))

	cmd.Flags().String("template-dir", "", "Use templates from this path instead of the embedded files")
	viper.BindPFlag("template-dir", cmd.Flags().Lookup("template-dir"))

	cmd.AddCommand(
		newMigrateCmd(logStream, acsDocs, migrationFS),
		newPopulateACSCmd(logStream),
	)

	return cmd
}

func webServerRunner(logStream io.Writer) func(*cobra.Command, []string) error {
	return func(c *cobra.Command, s []string) error {
		return run(logStream)
	}
}

const addr = ":8000"

func run(logStream io.Writer) error {
	debug := viper.GetBool("debug")
	dsn := viper.GetString("dsn")

	logger := createLogger(logStream)

	appOpts := app.Options{Debug: debug}

	templateDir := viper.GetString("template-dir")

	var templateFiles fs.FS
	if templateDir != "" {
		templateFiles = os.DirFS(templateDir)
		appOpts.LiveTemplates = true
		logger.Info("Using live templates", "templateDir", templateDir)
	} else {
		templateFiles = html.Files
	}

	staticDir := viper.GetString("static-dir")

	var staticFiles fs.FS
	if staticDir != "" {
		staticFiles = os.DirFS(staticDir)
		logger.Info("Using live static files", "staticDir", staticDir)
	} else {
		staticFiles = static.Files
	}

	db, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %v", err)
	}

	defer db.Close()

	app, err := app.New(logger, templateFiles, staticFiles, db, &appOpts)
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
