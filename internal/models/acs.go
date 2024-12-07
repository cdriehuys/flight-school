package models

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AreaOfOperation struct {
	ID       int    `db:"id"`
	ACS      string `db:"acs_id"`
	PublicID string `db:"public_id"`
	Name     string `db:"name"`
}

func (a AreaOfOperation) FullID() string {
	return fmt.Sprintf("%s.%s", a.ACS, a.PublicID)
}

type Task struct {
	ID        int    `db:"id"`
	AreaID    int    `db:"area_id"`
	PublicID  string `db:"public_id"`
	Name      string `db:"name"`
	Objective string `db:"objective"`
}

type ACSModel struct {
	logger *slog.Logger
	db     *pgxpool.Pool
}

func NewACSModel(logger *slog.Logger, db *pgxpool.Pool) *ACSModel {
	return &ACSModel{logger, db}
}

func (m *ACSModel) GetAreaByID(ctx context.Context, acs string, id string) (AreaOfOperation, error) {
	query := `SELECT id, acs_id, public_id, "name"
		FROM acs_areas
		WHERE acs_id = $1 AND public_id = $2`

	rows, err := m.db.Query(ctx, query, acs, id)
	if err != nil {
		return AreaOfOperation{}, fmt.Errorf("failed to retrieve area with ID %s: %v", id, err)
	}

	return pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[AreaOfOperation])
}

func (m *ACSModel) ListAreasByACS(ctx context.Context, acs string) ([]AreaOfOperation, error) {
	query := `SELECT id, acs_id, public_id, "name"
		FROM acs_areas
		WHERE acs_id = $1
		ORDER BY "order" ASC`
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

func (m *ACSModel) ListTasksByArea(ctx context.Context, areaID int) ([]Task, error) {
	query := `SELECT id, area_id, public_id, name, objective
		FROM acs_area_tasks
		WHERE area_id = $1
		ORDER BY public_id ASC`
	rows, err := m.db.Query(ctx, query, areaID)
	if err != nil {
		return nil, fmt.Errorf("failed to query tasks for area %d: %v", areaID, err)
	}

	return pgx.CollectRows(rows, pgx.RowToStructByName[Task])
}
