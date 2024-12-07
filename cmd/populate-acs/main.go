package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/jackc/pgx/v5"
)

func main() {
	if err := run(); err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
}

func run() error {
	dsn := flag.String("dsn", "postgres://localhost", "DSN for the database")
	flag.Parse()

	db, err := pgx.Connect(context.Background(), *dsn)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %v", err)
	}

	defer func() {
		if err := db.Close(context.Background()); err != nil {
			log.Println("Error closing database connection:", err)
		}
	}()

	dataFile := flag.Arg(0)
	fmt.Printf("Loading ACS from: %s\n", dataFile)

	file, err := os.Open(dataFile)
	if err != nil {
		return fmt.Errorf("failed to open %s: %v", dataFile, err)
	}

	defer func() {
		if err := file.Close(); err != nil {
			log.Printf("Error closing %s: %v\n", dataFile, err)
		}
	}()

	var acs ACS
	if err := json.NewDecoder(file).Decode(&acs); err != nil {
		return fmt.Errorf("failed to read ACS from %s: %v", dataFile, err)
	}

	return insertACS(context.Background(), db, acs)
}

type ACS struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Areas []Area `json:"areas"`
}

type Area struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Tasks []Task `json:"tasks"`
}

type Task struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Objective string `json:"objective"`
}

func insertACS(ctx context.Context, db *pgx.Conn, acs ACS) error {
	query := `INSERT INTO acs (id, name)
		VALUES ($1, $2)
		ON CONFLICT (id) DO UPDATE
		SET name = EXCLUDED.name`

	if _, err := db.Exec(ctx, query, acs.ID, acs.Name); err != nil {
		return fmt.Errorf("failed to insert ACS %s: %v", acs.ID, err)
	}

	log.Printf("%s - Updated ACS\n", acs.ID)

	for i, area := range acs.Areas {
		if err := upsertArea(ctx, db, acs.ID, i, area); err != nil {
			return fmt.Errorf("failed to update %s.%s: %v", acs.ID, area.ID, err)
		}
	}

	return nil
}

func upsertArea(ctx context.Context, db *pgx.Conn, acsID string, order int, area Area) error {
	query := `INSERT INTO acs_areas (acs_id, public_id, name, "order")
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (acs_id, public_id) DO UPDATE
		SET name = EXCLUDED.name, "order" = EXCLUDED."order"
		RETURNING id`

	var areaPK int
	if err := db.QueryRow(ctx, query, acsID, area.ID, area.Name, order).Scan(&areaPK); err != nil {
		return fmt.Errorf("failed to update area %s.%s: %v", acsID, area.ID, err)
	}

	areaID := fmt.Sprintf("%s.%s", acsID, area.ID)
	log.Printf("%s - Updated area - %s\n", areaID, area.Name)

	for _, task := range area.Tasks {
		if err := upsertTask(ctx, db, areaID, areaPK, task); err != nil {
			return fmt.Errorf("failed to update task %s.%s: %v", areaID, task.ID, err)
		}
	}

	return nil
}

func upsertTask(ctx context.Context, db *pgx.Conn, areaID string, areaPK int, task Task) error {
	query := `INSERT INTO acs_area_tasks (area_id, public_id, name, objective)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (area_id, public_id) DO UPDATE
		SET name = EXCLUDED.name, objective = EXCLUDED.objective
		RETURNING id`

	var taskPK int
	if err := db.QueryRow(ctx, query, areaPK, task.ID, task.Name, task.Objective).Scan(&taskPK); err != nil {
		return fmt.Errorf("failed to update task %s.%s: %v", areaID, task.ID, err)
	}

	log.Printf("%s.%s - Updated task - %s", areaID, task.ID, task.Name)

	return nil
}
