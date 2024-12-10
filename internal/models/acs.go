package models

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/cdriehuys/flight-school/internal/models/queries"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AreaOfOperation struct {
	ID       int32
	ACS      string
	PublicID string
	Name     string
}

func (a AreaOfOperation) FullID() string {
	return fmt.Sprintf("%s.%s", a.ACS, a.PublicID)
}

func areaOfOperationFromModel(m queries.AcsArea) AreaOfOperation {
	return AreaOfOperation{
		ID:       m.ID,
		ACS:      m.AcsID,
		PublicID: m.PublicID,
		Name:     m.Name,
	}
}

type TaskSummary struct {
	ID        int32
	AreaID    int32
	PublicID  string
	Name      string
	Objective string

	FullPublicID string

	Confidence TaskConfidence

	KnowledgeElementCount      int
	RiskManagementElementCount int
	SkillElementCount          int
}

type Task struct {
	ID        int32  `db:"id"`
	PublicID  string `db:"public_id"`
	Name      string `db:"name"`
	Objective string `db:"objective"`

	Area       AreaOfOperation
	Confidence TaskConfidence

	KnowledgeElements      []TaskElement
	RiskManagementElements []TaskElement
	SkillElements          []TaskElement
}

func (t Task) FullPublicID() string {
	return fmt.Sprintf("%s.%s.%s", t.Area.ACS, t.Area.PublicID, t.PublicID)
}

type TaskElementType string

const (
	TaskElementTypeKnowledge      TaskElementType = "K"
	TaskElementTypeRiskManagement TaskElementType = "R"
	TaskElementTypeSkills         TaskElementType = "S"
)

type TaskElement struct {
	ID       int32 `db:"id"`
	TaskID   int32 `db:"task_id"`
	Type     TaskElementType
	PublicID int32  `db:"public_id"`
	Content  string `db:"content"`

	SubElements []SubElement `db:"-"`
}

type SubElement struct {
	ID        int32  `db:"id"`
	ElementID int32  `db:"element_id"`
	PublicID  string `db:"public_id"`
	Content   string `db:"content"`
}

type ConfidenceLevel int16

const (
	ConfidenceLevelLow    ConfidenceLevel = 1
	ConfidenceLevelMedium ConfidenceLevel = 2
	ConfidenceLevelHigh   ConfidenceLevel = 3
)

type ACSModel struct {
	logger *slog.Logger
	db     *pgxpool.Pool
	q      queries.Queries
}

func NewACSModel(logger *slog.Logger, db *pgxpool.Pool) *ACSModel {
	return &ACSModel{logger, db, *queries.New(db)}
}

func (m *ACSModel) GetAreaByID(ctx context.Context, acs string, id string) (AreaOfOperation, error) {
	areaModel, err := m.q.GetAreaByPublicID(ctx, queries.GetAreaByPublicIDParams{
		AcsID:    acs,
		PublicID: id,
	})
	if err != nil {
		return AreaOfOperation{}, fmt.Errorf("failed to retrieve area %s.%s: %v", acs, id, err)
	}

	return areaOfOperationFromModel(areaModel), nil
}

func (m *ACSModel) ListAreasByACS(ctx context.Context, acs string) ([]AreaOfOperation, error) {
	areaModels, err := m.q.ListAreasByACS(ctx, acs)
	if err != nil {
		return nil, fmt.Errorf("failed to list areas for ACS %s: %v", acs, err)
	}

	areas := make([]AreaOfOperation, len(areaModels))
	for i, a := range areaModels {
		areas[i] = areaOfOperationFromModel(a)
	}

	return areas, nil
}

func (m *ACSModel) ListTasksByArea(ctx context.Context, areaID int32) ([]TaskSummary, error) {
	taskModels, err := m.q.ListTasksByArea(ctx, int32(areaID))
	if err != nil {
		return nil, fmt.Errorf("failed to query tasks for area %d: %v", areaID, err)
	}

	tasks := make([]TaskSummary, len(taskModels))
	for i, t := range taskModels {
		tasks[i] = TaskSummary{
			ID:                         t.Task.ID,
			AreaID:                     areaID,
			PublicID:                   t.Task.PublicID,
			Name:                       t.Task.Name,
			Objective:                  t.Task.Objective,
			FullPublicID:               t.FullPublicID,
			Confidence:                 TaskConfidence{int(t.Votes), int(t.MaxVotes)},
			KnowledgeElementCount:      int(t.KnowledgeElementCount),
			RiskManagementElementCount: int(t.RiskElementCount),
			SkillElementCount:          int(t.SkillElementCount),
		}
	}

	return tasks, nil
}

func (m *ACSModel) GetTaskByArea(ctx context.Context, acs string, areaID string, taskID string) (Task, error) {
	row, err := m.q.GetTaskByPublicID(ctx, queries.GetTaskByPublicIDParams{
		Acs:    acs,
		AreaID: areaID,
		TaskID: taskID,
	})
	if err != nil {
		return Task{}, fmt.Errorf("failed to retrieve task %s.%s.%s: %v", acs, areaID, taskID, err)
	}

	task := Task{
		ID:         row.Task.ID,
		PublicID:   row.Task.PublicID,
		Name:       row.Task.Name,
		Objective:  row.Task.Objective,
		Area:       areaOfOperationFromModel(row.AcsArea),
		Confidence: TaskConfidence{int(row.Votes), int(row.MaxVotes)},
	}

	return m.addElementsToTask(ctx, task)
}

func (m *ACSModel) GetTaskByElementID(ctx context.Context, elementID int32) (Task, error) {
	row, err := m.q.GetTaskByElementID(ctx, elementID)
	if err != nil {
		return Task{}, fmt.Errorf("failed to retrieve parent task for element %d: %v", elementID, err)
	}

	task := Task{
		ID:         row.Task.ID,
		PublicID:   row.Task.PublicID,
		Name:       row.Task.Name,
		Objective:  row.Task.Objective,
		Area:       areaOfOperationFromModel(row.AcsArea),
		Confidence: TaskConfidence{int(row.Votes), int(row.MaxVotes)},
	}

	return m.addElementsToTask(ctx, task)
}

func (m *ACSModel) addElementsToTask(ctx context.Context, task Task) (Task, error) {
	elements, err := m.listElementsForTasks(ctx, []int32{task.ID})
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

func (m *ACSModel) listElementsForTasks(ctx context.Context, taskIDs []int32) (map[int32]map[TaskElementType][]TaskElement, error) {
	elements, err := m.q.ListElementsByTaskIDs(ctx, taskIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to list task elements: %v", err)
	}

	elementIDs := make([]int32, 0)
	for _, e := range elements {
		elementIDs = append(elementIDs, e.ID)
	}

	subElements, err := m.listSubElements(ctx, elementIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to list sub-elements for elements: %v", err)
	}

	elementsByTask := make(map[int32]map[TaskElementType][]TaskElement)
	for _, e := range elements {
		if _, ok := elementsByTask[e.TaskID]; !ok {
			elementsByTask[e.TaskID] = make(map[TaskElementType][]TaskElement)
		}

		elementType := taskElementTypeFromModel(e.Type)
		if _, ok := elementsByTask[e.TaskID][elementType]; !ok {
			elementsByTask[e.TaskID][elementType] = make([]TaskElement, 0, 1)
		}

		element := TaskElement{
			ID:          e.ID,
			TaskID:      e.TaskID,
			Type:        elementType,
			PublicID:    e.PublicID,
			Content:     e.Content,
			SubElements: subElements[e.ID],
		}

		elementsByTask[e.TaskID][elementType] = append(
			elementsByTask[e.TaskID][elementType],
			element,
		)
	}

	return elementsByTask, nil
}

func taskElementTypeFromModel(elementType queries.AcsElementType) TaskElementType {
	switch elementType {
	case queries.AcsElementTypeK:
		return TaskElementTypeKnowledge

	case queries.AcsElementTypeR:
		return TaskElementTypeRiskManagement

	case queries.AcsElementTypeS:
		return TaskElementTypeSkills
	}

	panic(fmt.Sprintf("Unknown element type: %s", elementType))
}

func (m *ACSModel) listSubElements(ctx context.Context, elementIDs []int32) (map[int32][]SubElement, error) {
	subElements, err := m.q.ListSubElementsByElementIDs(ctx, elementIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to query sub-elements: %v", err)
	}

	subElementsByElementID := make(map[int32][]SubElement)
	for _, s := range subElements {
		if _, ok := subElementsByElementID[s.ElementID]; !ok {
			subElementsByElementID[s.ElementID] = make([]SubElement, 0, 1)
		}

		subElementsByElementID[s.ElementID] = append(
			subElementsByElementID[s.ElementID],
			SubElement{
				ID:        s.ID,
				ElementID: s.ElementID,
				PublicID:  s.PublicID,
				Content:   s.Content,
			},
		)
	}

	return subElementsByElementID, nil
}

func (m *ACSModel) SetElementConfidence(ctx context.Context, elementID int32, confidence ConfidenceLevel) error {
	params := queries.SetElementConfidenceParams{
		ElementID: elementID,
		Vote:      int16(confidence),
	}
	if err := m.q.SetElementConfidence(ctx, params); err != nil {
		return fmt.Errorf("failed to update confidence for element %d: %v", elementID, err)
	}

	m.logger.InfoContext(ctx, "Set element confidence.", "elementID", elementID, "confidence", confidence)

	return nil
}

type TaskConfidence struct {
	Votes    int
	Possible int
}

func (m *ACSModel) GetTaskConfidence(ctx context.Context, taskID int32) (TaskConfidence, error) {
	result, err := m.q.GetTaskConfidenceByTaskID(ctx, taskID)
	if err != nil {
		return TaskConfidence{}, fmt.Errorf("failed to get task confidence: %v", err)
	}

	confidence := TaskConfidence{Votes: int(result.Votes), Possible: int(result.Possible)}

	return confidence, nil
}
