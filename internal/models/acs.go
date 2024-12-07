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

	KnowledgeElements      []TaskElement `db:"-"`
	RiskManagementElements []TaskElement `db:"-"`
	SkillElements          []TaskElement `db:"-"`
}

type TaskElementType string

const (
	TaskElementTypeKnowledge      TaskElementType = "K"
	TaskElementTypeRiskManagement TaskElementType = "R"
	TaskElementTypeSkills         TaskElementType = "S"
)

type TaskElement struct {
	ID       int `db:"id"`
	TaskID   int `db:"task_id"`
	Type     TaskElementType
	PublicID int    `db:"public_id"`
	Content  string `db:"content"`
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

	tasks, err := pgx.CollectRows(rows, pgx.RowToStructByName[Task])
	if err != nil {
		return nil, fmt.Errorf("failed to collect tasks: %v", err)
	}

	taskIDs := make([]int, len(tasks))
	for _, task := range tasks {
		taskIDs = append(taskIDs, task.ID)
	}

	taskElements, err := m.listElementsForTasks(ctx, taskIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to list elements for tasks: %v", err)
	}

	for i, task := range tasks {
		elements, hasElements := taskElements[task.ID]
		if !hasElements {
			continue
		}

		tasks[i].KnowledgeElements = elements[TaskElementTypeKnowledge]
		tasks[i].RiskManagementElements = elements[TaskElementTypeRiskManagement]
		tasks[i].SkillElements = elements[TaskElementTypeSkills]
	}

	return tasks, nil
}

func (m *ACSModel) listElementsForTasks(ctx context.Context, taskIDs []int) (map[int]map[TaskElementType][]TaskElement, error) {
	query := `SELECT id, task_id, "type", public_id, content
		FROM acs_elements
		WHERE task_id = ANY ($1)
		ORDER BY public_id ASC`

	rows, err := m.db.Query(ctx, query, taskIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to query task elements: %v", err)
	}

	elementsByTask := make(map[int]map[TaskElementType][]TaskElement)

	_, err = pgx.CollectRows(rows, func(row pgx.CollectableRow) (struct{}, error) {
		el, err := pgx.RowToStructByName[TaskElement](row)
		if err != nil {
			return struct{}{}, err
		}

		_, taskExists := elementsByTask[el.TaskID]
		if !taskExists {
			elementsByTask[el.TaskID] = make(map[TaskElementType][]TaskElement)
		}

		_, elementTypeExists := elementsByTask[el.TaskID][el.Type]
		if !elementTypeExists {
			elementsByTask[el.TaskID][el.Type] = make([]TaskElement, 0, 1)
		}

		elementsByTask[el.TaskID][el.Type] = append(elementsByTask[el.TaskID][el.Type], el)

		return struct{}{}, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to collect task elements: %v", err)
	}

	return elementsByTask, nil
}
