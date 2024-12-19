package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/cdriehuys/flight-school/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func NewPopulateACSCmd(logStream io.Writer) *cobra.Command {
	return &cobra.Command{
		Use:   "populate-acs definition-file",
		Short: "Populate the database with a particular ACS",
		Args:  cobra.ExactArgs(1),
		RunE:  populateACSRunner(logStream),
	}
}

func populateACSRunner(logStream io.Writer) func(*cobra.Command, []string) error {
	return func(c *cobra.Command, args []string) error {
		logger := createLogger(logStream)
		db, err := pgxpool.New(c.Context(), viper.GetString("dsn"))
		if err != nil {
			return fmt.Errorf("failed to open database connection: %v", err)
		}

		model := models.NewACSModel(logger, db)

		acsFileName := args[0]
		acsFile, err := os.Open(acsFileName)
		if err != nil {
			return fmt.Errorf("failed to open %s: %v", acsFileName, err)
		}

		logger.Info("Opened ACS document", "file", acsFileName)

		if err := populateACSFromJSON(c.Context(), model, acsFile); err != nil {
			return fmt.Errorf("failed to populate ACS: %v", err)
		}

		defer func() {
			if err := acsFile.Close(); err != nil {
				log.Printf("Error closing ACS file: %v", err)
			}
		}()

		return nil
	}
}

type acsUpdater interface {
	PopulateACS(ctx context.Context, acs models.ExternalACS) error
}

func populateACSFromJSON(ctx context.Context, model acsUpdater, input io.Reader) error {
	var acs models.ExternalACS
	if err := json.NewDecoder(input).Decode(&acs); err != nil {
		return fmt.Errorf("failed to decode JSON ACS: %v", err)
	}

	if err := model.PopulateACS(ctx, acs); err != nil {
		return fmt.Errorf("failed to update ACS: %v", err)
	}

	return nil
}
