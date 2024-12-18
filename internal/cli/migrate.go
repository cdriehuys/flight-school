package cli

import (
	"fmt"
	"io/fs"
	"log"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/tern/v2/migrate"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func NewMigrateCmd(migrationFS fs.FS) *cobra.Command {
	return &cobra.Command{
		Use:   "migrate",
		Short: "Migrate the database forwards",
		RunE:  migrateRunner(migrationFS),
	}
}

func migrateRunner(migrationFS fs.FS) func(*cobra.Command, []string) error {
	return func(c *cobra.Command, s []string) error {
		dsn := viper.GetString("dsn")
		conn, err := pgx.Connect(c.Context(), dsn)
		if err != nil {
			return fmt.Errorf("failed to connect to database: %v", err)
		}

		defer func() {
			if err := conn.Close(c.Context()); err != nil {
				log.Printf("Error closing database connection: %v", err)
			}
		}()

		migrator, err := migrate.NewMigrator(c.Context(), conn, "public.schema_version")
		if err != nil {
			return fmt.Errorf("failed to build migrator: %v", err)
		}

		if err := migrator.LoadMigrations(migrationFS); err != nil {
			return fmt.Errorf("failed to load migrations: %v", err)
		}

		migrator.OnStart = func(i int32, name string, direction string, sql string) {
			log.Printf("executing %s %s\n%s\n", name, direction, sql)
		}

		if err := migrator.Migrate(c.Context()); err != nil {
			return fmt.Errorf("failed to run migrations: %v", err)
		}

		log.Println("Database successfully migrated")

		return nil
	}
}
