package cli

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"

	"github.com/cdriehuys/flight-school/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/tern/v2/migrate"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func newMigrateCmd(logStream io.Writer, acsDocs fs.FS, migrationFS fs.FS) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Migrate the database forwards",
		RunE:  migrateRunner(logStream, acsDocs, migrationFS),
	}

	cmd.Flags().String("acs-dir", "", "Use ACS documents from this path instead of the embedded files")
	viper.BindPFlag("acs-dir", cmd.Flags().Lookup("acs-dir"))

	cmd.Flags().Bool("populate-acs", false, "Populate the database after migrating it")
	viper.BindPFlag("populate-acs", cmd.Flags().Lookup("populate-acs"))

	return cmd
}

func migrateRunner(logStream io.Writer, acsDocs fs.FS, migrationFS fs.FS) func(*cobra.Command, []string) error {
	return func(cli *cobra.Command, s []string) error {
		logger := createLogger(logStream)

		dsn := viper.GetString("dsn")
		pool, err := pgxpool.New(cli.Context(), dsn)
		if err != nil {
			return fmt.Errorf("failed to connect to database: %v", err)
		}

		defer pool.Close()

		err = pool.AcquireFunc(cli.Context(), func(c *pgxpool.Conn) error {
			migrator, err := migrate.NewMigrator(cli.Context(), c.Conn(), "public.schema_version")
			if err != nil {
				return fmt.Errorf("failed to build migrator: %v", err)
			}

			if err := migrator.LoadMigrations(migrationFS); err != nil {
				return fmt.Errorf("failed to load migrations: %v", err)
			}

			migrator.OnStart = func(i int32, name string, direction string, sql string) {
				logger.Info("Executing migration", "name", name, "direction", direction)
				logger.Debug("Migration contents", "sql", sql)
			}

			if err := migrator.Migrate(cli.Context()); err != nil {
				return fmt.Errorf("failed to run migrations: %v", err)
			}

			logger.Info("Database migrations completed successfully")

			return nil
		})

		if err != nil {
			return err
		}

		if viper.GetBool("populate-acs") {
			acsDir := viper.GetString("acs-dir")

			var docs fs.FS
			if acsDir == "" {
				docs = acsDocs
			} else {
				docs = os.DirFS(acsDir)
			}

			if err := loadACSDefinitions(cli.Context(), logger, pool, docs); err != nil {
				return err
			}
		}

		return nil
	}
}

func loadACSDefinitions(ctx context.Context, logger *slog.Logger, db *pgxpool.Pool, acsDocuments fs.FS) error {
	model := models.NewACSModel(logger, db)

	documents, err := fs.Glob(acsDocuments, "*.json")
	if err != nil {
		return fmt.Errorf("failed to find ACS documents: %v", err)
	}

	for _, doc := range documents {
		if err := loadACSDefinition(ctx, logger, model, acsDocuments, doc); err != nil {
			return err
		}
	}

	return nil
}

func loadACSDefinition(ctx context.Context, logger *slog.Logger, model acsUpdater, files fs.FS, name string) error {
	file, err := files.Open(name)
	if err != nil {
		return fmt.Errorf("failed to open %s: %v", name, err)
	}

	defer func() {
		if err := file.Close(); err != nil {
			logger.ErrorContext(ctx, "Failed to close ACS document", "document", name)
		}
	}()

	if err := populateACSFromJSON(ctx, model, file); err != nil {
		return err
	}

	logger.InfoContext(ctx, "Populated ACS definition", "document", name)

	return nil
}
