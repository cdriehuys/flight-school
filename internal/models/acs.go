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

	TaskCount  int
	Confidence Confidence
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

	Confidence Confidence

	KnowledgeElementCount      int
	RiskManagementElementCount int
	SkillElementCount          int
}

type Task struct {
	ID        int32
	PublicID  string
	Name      string
	Objective string
	Note      string

	Area       AreaOfOperation
	Confidence Confidence

	References []string

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
	ID       int32
	TaskID   int32
	Type     TaskElementType
	PublicID int32
	Content  string

	FullPublicID    string
	ConfidenceLevel *ConfidenceLevel

	SubElements []SubElement
}

type SubElement struct {
	ID        int32
	ElementID int32
	Order     int32
	Content   string
}

func (s SubElement) PublicID() string {
	alphabet := "abcdefghijklmnopqrstuvwxyz"

	return string(alphabet[s.Order])
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
		area := areaOfOperationFromModel(a.AcsArea)
		area.TaskCount = int(a.TaskCount)
		area.Confidence = Confidence{Votes: int(a.Votes), Possible: int(a.MaxVotes)}

		areas[i] = area
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
			Confidence:                 Confidence{int(t.Votes), int(t.MaxVotes)},
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
		Note:       row.Task.Note,
		Area:       areaOfOperationFromModel(row.AcsArea),
		Confidence: Confidence{int(row.Votes), int(row.MaxVotes)},
	}

	references, err := m.getTaskReferences(ctx, task.ID)
	if err != nil {
		return Task{}, fmt.Errorf("failed to fetch references: %v", err)
	}

	task.References = references

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
		Note:       row.Task.Note,
		Area:       areaOfOperationFromModel(row.AcsArea),
		Confidence: Confidence{int(row.Votes), int(row.MaxVotes)},
	}

	references, err := m.getTaskReferences(ctx, task.ID)
	if err != nil {
		return Task{}, fmt.Errorf("failed to fetch references: %v", err)
	}

	task.References = references

	return m.addElementsToTask(ctx, task)
}

func (m *ACSModel) getTaskReferences(ctx context.Context, taskID int32) ([]string, error) {
	references, err := m.q.GetTaskReferencesByTaskID(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch references for task %d: %v", taskID, err)
	}

	refValues := make([]string, len(references))
	for i, r := range references {
		refValues[i] = r.Document
	}

	return refValues, nil
}

func (m *ACSModel) addElementsToTask(ctx context.Context, task Task) (Task, error) {
	elements, err := m.listElementsForTask(ctx, task.ID)
	if err != nil {
		return Task{}, fmt.Errorf("failed to list elements for task: %v", err)
	}

	task.KnowledgeElements = elements[TaskElementTypeKnowledge]
	task.RiskManagementElements = elements[TaskElementTypeRiskManagement]
	task.SkillElements = elements[TaskElementTypeSkills]

	return task, nil
}

func (m *ACSModel) listElementsForTask(ctx context.Context, taskID int32) (map[TaskElementType][]TaskElement, error) {
	elements, err := m.q.ListElementsByTaskID(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("failed to list task elements for task %d: %v", taskID, err)
	}

	elementIDs := make([]int32, 0)
	for _, e := range elements {
		elementIDs = append(elementIDs, e.AcsElement.ID)
	}

	subElements, err := m.listSubElements(ctx, elementIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to list sub-elements for elements: %v", err)
	}

	elementsByType := make(map[TaskElementType][]TaskElement)
	for _, e := range elements {
		elementType := taskElementTypeFromModel(e.AcsElement.Type)
		if _, ok := elementsByType[elementType]; !ok {
			elementsByType[elementType] = make([]TaskElement, 0, 1)
		}

		element := TaskElement{
			ID:           e.AcsElement.ID,
			TaskID:       e.AcsElement.TaskID,
			Type:         elementType,
			PublicID:     e.AcsElement.PublicID,
			Content:      e.AcsElement.Content,
			FullPublicID: e.FullPublicID,
			SubElements:  subElements[e.AcsElement.ID],
		}

		if e.ConfidenceVote.Valid {
			level := ConfidenceLevel(e.ConfidenceVote.Int16)
			element.ConfidenceLevel = &level
		}

		elementsByType[elementType] = append(
			elementsByType[elementType],
			element,
		)
	}

	return elementsByType, nil
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
				Order:     s.Order,
				Content:   s.Content,
			},
		)
	}

	return subElementsByElementID, nil
}

func (m *ACSModel) GetElementPublicIDByID(ctx context.Context, elementID int32) (string, error) {
	publicID, err := m.q.GetElementPublicIDByID(ctx, elementID)
	if err != nil {
		return "", fmt.Errorf("failed to retrieve public ID of element %d: %v", elementID, err)
	}

	return publicID, nil
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

func (m *ACSModel) ClearElementConfidence(ctx context.Context, elementID int32) error {
	if err := m.q.ClearElementConfidence(ctx, elementID); err != nil {
		return fmt.Errorf("failed to clear confidence for element %d: %v", elementID, err)
	}

	m.logger.InfoContext(ctx, "Cleared element confidence.", "elementID", elementID)

	return nil
}

type Confidence struct {
	Votes    int
	Possible int
}

func (m *ACSModel) GetTaskConfidence(ctx context.Context, taskID int32) (Confidence, error) {
	result, err := m.q.GetTaskConfidenceByTaskID(ctx, taskID)
	if err != nil {
		return Confidence{}, fmt.Errorf("failed to get task confidence: %v", err)
	}

	confidence := Confidence{Votes: int(result.Votes), Possible: int(result.Possible)}

	return confidence, nil
}
