package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/cdriehuys/flight-school/internal/models"
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

	Knowledge      []Element `json:"knowledge"`
	RiskManagement []Element `json:"riskManagement"`
	Skills         []Element `json:"skills"`
}

type Element struct {
	ID          int          `json:"id"`
	Content     string       `json:"content"`
	SubElements []SubElement `json:"subElements"`
}

type SubElement struct {
	Content string `json:"content"`
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

	taskPublicID := fmt.Sprintf("%s.%s", areaID, task.ID)
	log.Printf("%s - Updated task - %s", taskPublicID, task.Name)

	upsertElements := func(elementType models.TaskElementType, elements []Element) error {
		for _, element := range elements {
			if err := upsertElement(ctx, db, taskPublicID, taskPK, elementType, element); err != nil {
				return fmt.Errorf("failed to update element %s.%s%d: %v", taskPublicID, elementType, element.ID, err)
			}
		}

		return nil
	}

	if err := upsertElements(models.TaskElementTypeKnowledge, task.Knowledge); err != nil {
		return err
	}

	if err := upsertElements(models.TaskElementTypeRiskManagement, task.RiskManagement); err != nil {
		return err
	}

	if err := upsertElements(models.TaskElementTypeSkills, task.Skills); err != nil {
		return err
	}

	return nil
}

func upsertElement(
	ctx context.Context,
	db *pgx.Conn,
	taskPublicID string,
	taskPK int,
	elementType models.TaskElementType,
	element Element,
) error {
	query := `INSERT INTO acs_elements (task_id, "type", public_id, content)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (task_id, "type", public_id) DO UPDATE
		SET content = EXCLUDED.content
		RETURNING id`

	var elementPK int
	if err := db.QueryRow(ctx, query, taskPK, elementType, element.ID, element.Content).Scan(&elementPK); err != nil {
		return fmt.Errorf("failed to update task %s.%s%d: %v", taskPublicID, elementType, element.ID, err)
	}

	elementPublicID := fmt.Sprintf("%s.%s%d", taskPublicID, elementType, element.ID)
	log.Printf("%s - Updated element", elementPublicID)

	for i, subElement := range element.SubElements {
		if err := upsertSubelement(ctx, db, elementPublicID, elementPK, i, subElement); err != nil {
			return fmt.Errorf("failed to update subelement: %v", err)
		}
	}

	return nil
}

const subElementAlphabet = "abcdefghijklmnopqrstuvwxyz"

func upsertSubelement(
	ctx context.Context,
	db *pgx.Conn,
	elementPublicID string,
	elementPK int,
	order int,
	subElement SubElement,
) error {
	query := `INSERT INTO acs_subelements (element_id, "order", content)
		VALUES ($1, $2, $3)
		ON CONFLICT (element_id, "order") DO UPDATE
		SET content = EXCLUDED.content`

	subElementName := fmt.Sprintf("%s%c", elementPublicID, subElementAlphabet[order])

	if _, err := db.Exec(ctx, query, elementPK, order, subElement.Content); err != nil {
		return fmt.Errorf("failed to update sub-element %s: %v", subElementName, err)
	}

	log.Printf("%s - Updated sub-element", subElementName)

	return nil
}
