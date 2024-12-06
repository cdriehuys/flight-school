package models

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AreaOfOperation struct {
	ID   string `db:"id"`
	Name string `db:"name"`
}

type ACSModel struct {
	logger *slog.Logger
	db     *pgxpool.Pool
}

func NewACSModel(logger *slog.Logger, db *pgxpool.Pool) *ACSModel {
	return &ACSModel{logger, db}
}

func (m *ACSModel) GetAreaByID(ctx context.Context, id string) (AreaOfOperation, error) {
	query := `SELECT id, "name" FROM acs_areas WHERE id = $1`

	var area AreaOfOperation
	if err := m.db.QueryRow(ctx, query, id).Scan(&area.ID, &area.Name); err != nil {
		return area, fmt.Errorf("failed to retrieve area with ID %s: %v", id, err)
	}

	return area, nil
}

func (m *ACSModel) ListAreasByACS(ctx context.Context, acs string) ([]AreaOfOperation, error) {
	query := `SELECT id, "name" FROM acs_areas WHERE id LIKE $1 || '.%'`
	rows, err := m.db.Query(ctx, query, acs)
	if err != nil {
		return nil, fmt.Errorf("failed to query areas in for ACS %s: %v", acs, err)
	}

	areas, err := pgx.CollectRows(rows, pgx.RowToStructByName[AreaOfOperation])
	if err != nil {
		return nil, fmt.Errorf("failed to collect areas: %v", err)
	}

	return areas, nil
}
