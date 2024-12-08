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

	ACS          string `db:"acs"`
	AreaPublicID string `db:"area_public_id"`
	AreaName     string `db:"area_name"`

	Confidence TaskConfidence `db:"-"`

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

	SubElements []SubElement `db:"-"`
}

type SubElement struct {
	ID        int    `db:"id"`
	ElementID int    `db:"element_id"`
	PublicID  string `db:"public_id"`
	Content   string `db:"content"`
}

type ElementConfidence int

const (
	ElementConfidenceLow    ElementConfidence = 1
	ElementConfidenceMedium ElementConfidence = 2
	ElementConfidenceHigh   ElementConfidence = 3
)

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
	query := `SELECT
			t.id,
			t.area_id,
			t.public_id,
			t.name,
			t.objective,
			a.acs_id AS acs,
			a.public_id AS area_public_id,
			a.name AS area_name
		FROM acs_area_tasks t
			LEFT JOIN acs_areas a ON t.area_id = a.id
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

	confidences, err := m.listTaskConfidences(ctx, taskIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to get confidence for tasks: %v", err)
	}

	for i, task := range tasks {
		elements, hasElements := taskElements[task.ID]
		if hasElements {
			tasks[i].KnowledgeElements = elements[TaskElementTypeKnowledge]
			tasks[i].RiskManagementElements = elements[TaskElementTypeRiskManagement]
			tasks[i].SkillElements = elements[TaskElementTypeSkills]
		}

		tasks[i].Confidence = confidences[task.ID]
	}

	return tasks, nil
}

func (m *ACSModel) GetTaskByArea(ctx context.Context, acs string, areaID string, taskID string) (Task, error) {
	query := `SELECT
			t.id,
			t.area_id,
			t.public_id,
			t.name,
			t.objective,
			a.acs_id AS acs,
			a.public_id AS area_public_id,
			a.name AS area_name
		FROM acs_area_tasks t
			LEFT JOIN acs_areas a ON t.area_id = a.id
		WHERE a.acs_id = $1 AND a.public_id = $2 AND t.public_id = $3`
	rows, _ := m.db.Query(ctx, query, acs, areaID, taskID)

	return m.getTaskFromRows(ctx, rows)
}

func (m *ACSModel) GetTaskByElementID(ctx context.Context, elementID int) (Task, error) {
	query := `SELECT DISTINCT
			t.id,
			t.area_id,
			t.public_id,
			t.name,
			t.objective,
			a.acs_id AS acs,
			a.public_id AS area_public_id,
			a.name AS area_name
		FROM acs_elements e
			LEFT JOIN acs_area_tasks t ON e.task_id = t.id
			LEFT JOIN acs_areas a ON t.area_id = a.id
		WHERE e.id = $1`
	rows, _ := m.db.Query(ctx, query, elementID)

	return m.getTaskFromRows(ctx, rows)
}

func (m *ACSModel) getTaskFromRows(ctx context.Context, rows pgx.Rows) (Task, error) {
	task, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[Task])
	if err != nil {
		return Task{}, fmt.Errorf("failed to collect task: %v", err)
	}

	elements, err := m.listElementsForTasks(ctx, []int{task.ID})
	if err != nil {
		return Task{}, fmt.Errorf("failed to list elements for task: %v", err)
	}

	taskElements, hasElements := elements[task.ID]
	if hasElements {
		task.KnowledgeElements = taskElements[TaskElementTypeKnowledge]
		task.RiskManagementElements = taskElements[TaskElementTypeRiskManagement]
		task.SkillElements = taskElements[TaskElementTypeSkills]
	}

	return task, nil
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

	elementIDs := make([]int, 0)
	elementsByTask := make(map[int]map[TaskElementType][]TaskElement)

	_, err = pgx.CollectRows(rows, func(row pgx.CollectableRow) (struct{}, error) {
		el, err := pgx.RowToStructByName[TaskElement](row)
		if err != nil {
			return struct{}{}, err
		}

		elementIDs = append(elementIDs, el.ID)

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

	subElements, err := m.listSubElements(ctx, elementIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to collect sub-elements: %v", err)
	}

	for taskID, elementTypes := range elementsByTask {
		for elementType, elements := range elementTypes {
			for i, element := range elements {
				s, exists := subElements[element.ID]
				if exists {
					elementsByTask[taskID][elementType][i].SubElements = s
				}
			}
		}
	}

	return elementsByTask, nil
}

func (m *ACSModel) listSubElements(ctx context.Context, elementIDs []int) (map[int][]SubElement, error) {
	query := `SELECT id, element_id, public_id, content
		FROM acs_subelements
		WHERE element_id = ANY ($1)
		ORDER BY public_id ASC`

	rows, err := m.db.Query(ctx, query, elementIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to query sub-elements: %v", err)
	}

	subElements := make(map[int][]SubElement, len(elementIDs))

	_, err = pgx.CollectRows(rows, func(row pgx.CollectableRow) (struct{}, error) {
		subElement, err := pgx.RowToStructByName[SubElement](row)
		if err != nil {
			return struct{}{}, err
		}

		_, alreadySawElement := subElements[subElement.ElementID]
		if !alreadySawElement {
			subElements[subElement.ElementID] = make([]SubElement, 0, 1)
		}

		subElements[subElement.ElementID] = append(subElements[subElement.ElementID], subElement)

		return struct{}{}, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to collect sub-elements: %v", err)
	}

	return subElements, nil
}

func (m *ACSModel) SetElementConfidence(ctx context.Context, elementID int, confidence ElementConfidence) error {
	query := `INSERT INTO element_confidence (element_id, vote)
		VALUES ($1, $2)
		ON CONFLICT (element_id) DO UPDATE
		SET vote = EXCLUDED.vote`

	if _, err := m.db.Exec(ctx, query, elementID, confidence); err != nil {
		return fmt.Errorf("failed to update confidence for element %d: %v", elementID, err)
	}

	m.logger.InfoContext(ctx, "Set element confidence.", "elementID", elementID, "confidence", confidence)

	return nil
}

type TaskConfidence struct {
	Votes    int
	Possible int
}

func (m *ACSModel) GetTaskConfidence(ctx context.Context, taskID int) (TaskConfidence, error) {
	query := `WITH task_elements AS (
			SELECT id FROM acs_elements WHERE task_id = $1
		), max_votes AS (
			SELECT COALESCE(COUNT(*) * 3, 0) AS max_votes FROM task_elements
		)
		SELECT
			COALESCE(SUM(c.vote), 0) AS votes,
			(SELECT max_votes FROM max_votes) AS possible
		FROM element_confidence c
		WHERE c.element_id IN (SELECT id FROM task_elements)
	`

	var confidence TaskConfidence
	if err := m.db.QueryRow(ctx, query, taskID).Scan(&confidence.Votes, &confidence.Possible); err != nil {
		return TaskConfidence{}, fmt.Errorf("failed to get task confidence: %v", err)
	}

	return confidence, nil
}

func (m *ACSModel) listTaskConfidences(ctx context.Context, taskIDs []int) (map[int]TaskConfidence, error) {
	query := `WITH max_votes AS (
			SELECT COALESCE(COUNT(*) * 3, 0) AS max_votes, e.task_id AS task_id
			FROM acs_elements e
			GROUP BY e.task_id
		)
		SELECT
			e.task_id AS task_id,
			COALESCE(SUM(c.vote), 0) AS votes,
			(SELECT max_votes FROM max_votes WHERE task_id = e.task_id) AS possible
		FROM element_confidence c
			RIGHT JOIN acs_elements e ON c.element_id = e.id
		WHERE e.task_id = ANY ($1)
		GROUP BY e.task_id`
	rows, _ := m.db.Query(ctx, query, taskIDs)

	confidenceByTask := make(map[int]TaskConfidence)

	_, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (struct{}, error) {
		var taskID int
		var c TaskConfidence
		if err := row.Scan(&taskID, &c.Votes, &c.Possible); err != nil {
			return struct{}{}, err
		}

		confidenceByTask[taskID] = c

		return struct{}{}, nil
	})

	return confidenceByTask, err
}
