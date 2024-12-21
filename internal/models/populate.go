package models

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/cdriehuys/flight-school/internal/models/queries"
	"github.com/jackc/pgx/v5"
)

type ExternalACS struct {
	ID    string         `json:"id"`
	Name  string         `json:"name"`
	Areas []ExternalArea `json:"areas"`
}

type ExternalArea struct {
	ID    string         `json:"id"`
	Name  string         `json:"name"`
	Tasks []ExternalTask `json:"tasks"`
}

type ExternalTask struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Objective string `json:"objective"`
	Note      string `json:"note"`

	References []string `json:"references"`

	Knowledge      []ExternalElement `json:"knowledge"`
	RiskManagement []ExternalElement `json:"riskManagement"`
	Skills         []ExternalElement `json:"skills"`
}

type ExternalElement struct {
	ID          int32                `json:"id"`
	Content     string               `json:"content"`
	SubElements []ExternalSubElement `json:"subElements"`
}

type ExternalSubElement struct {
	Content string `json:"content"`
}

func (m *ACSModel) PopulateACS(ctx context.Context, acs ExternalACS) error {
	tx, err := m.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to start transaction: %v", err)
	}

	defer func() {
		if err := tx.Rollback(ctx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
			m.logger.Error("Failed to rollback ACS population transaction.", "error", err)
		}
	}()

	q := queries.New(tx)
	acsModel, err := q.UpsertACS(ctx, queries.UpsertACSParams{
		ID:   acs.ID,
		Name: acs.Name,
	})
	if err != nil {
		return fmt.Errorf("failed to insert ACS: %v", err)
	}

	logger := m.logger.With("acs", acsModel.ID)
	logger.InfoContext(ctx, "Updated ACS")

	knownAreas := make([]int32, len(acs.Areas))
	for i, area := range acs.Areas {
		areaModel, err := m.upsertArea(ctx, logger, q, acsModel.ID, int32(i), area)
		if err != nil {
			return fmt.Errorf("failed to update area %s: %v", area.ID, err)
		}

		knownAreas[i] = areaModel.ID
	}

	unknownAreaCount, err := q.ClearUnknownAreas(ctx, queries.ClearUnknownAreasParams{
		AcsID:    acsModel.ID,
		KnownIds: knownAreas,
	})
	if err != nil {
		return fmt.Errorf("failed to remove unknown areas: %v", err)
	}

	if unknownAreaCount == 0 {
		logger.DebugContext(ctx, "No unknown areas to remove.")
	} else {
		logger.InfoContext(ctx, "Removed unknown areas.", "count", unknownAreaCount)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit ACS update: %v", err)
	}

	return nil
}

func (m *ACSModel) upsertArea(
	ctx context.Context,
	logger *slog.Logger,
	q *queries.Queries,
	acs string,
	order int32,
	area ExternalArea,
) (queries.AcsArea, error) {
	areaModel, err := q.UpsertArea(ctx, queries.UpsertAreaParams{
		AcsID:    acs,
		PublicID: area.ID,
		Name:     area.Name,
		Order:    order,
	})
	if err != nil {
		return queries.AcsArea{}, fmt.Errorf("failed to upsert ACS area: %v", err)
	}

	logger = logger.With("area", areaModel.PublicID)
	logger.InfoContext(ctx, "Updated ACS area")

	knownTasks := make([]int32, len(area.Tasks))
	for i, task := range area.Tasks {
		taskModel, err := m.upsertTask(ctx, logger, q, areaModel.ID, task)
		if err != nil {
			return queries.AcsArea{}, fmt.Errorf("failed to update task for area: %v", err)
		}

		knownTasks[i] = taskModel.ID
	}

	unknownTaskCount, err := q.ClearUnknownTasks(ctx, queries.ClearUnknownTasksParams{
		AreaID:   areaModel.ID,
		KnownIds: knownTasks,
	})
	if err != nil {
		return queries.AcsArea{}, fmt.Errorf("failed to clear unknown tasks: %v", err)
	}

	if unknownTaskCount == 0 {
		logger.DebugContext(ctx, "No extra tasks deleted.")
	} else {
		logger.InfoContext(ctx, "Extra tasks deleted", "count", unknownTaskCount)
	}

	return areaModel, nil
}

func (m *ACSModel) upsertTask(
	ctx context.Context,
	logger *slog.Logger,
	q *queries.Queries,
	areaID int32,
	task ExternalTask,
) (queries.Task, error) {
	taskModel, err := q.UpsertTask(ctx, queries.UpsertTaskParams{
		AreaID:    areaID,
		PublicID:  task.ID,
		Name:      task.Name,
		Objective: task.Objective,
		Note:      task.Note,
	})
	if err != nil {
		return queries.Task{}, fmt.Errorf("failed to upsert task: %v", err)
	}

	logger = logger.With("task", taskModel.PublicID)
	logger.InfoContext(ctx, "Updated task")

	knownReferences := make([]int32, len(task.References))
	for i, reference := range task.References {
		referenceModel, err := m.upsertTaskReference(ctx, logger, q, taskModel.ID, int32(i), reference)
		if err != nil {
			return queries.Task{}, err
		}

		knownReferences[i] = referenceModel.ID
	}

	unknownReferenceCount, err := q.ClearUnknownTaskReferences(ctx, queries.ClearUnknownTaskReferencesParams{
		TaskID:   taskModel.ID,
		KnownIds: knownReferences,
	})
	if err != nil {
		return queries.Task{}, fmt.Errorf("failed to clear unknown references: %v", err)
	}

	if unknownReferenceCount == 0 {
		logger.DebugContext(ctx, "No extra task references to remove")
	} else {
		logger.InfoContext(ctx, "Removed extra task references", "count", unknownReferenceCount)
	}

	knownElements := make([]int32, 0, len(task.Knowledge)+len(task.RiskManagement)+len(task.Skills))
	upsertElements := func(elementType TaskElementType, elements []ExternalElement) error {
		for _, e := range elements {
			elementModel, err := m.upsertElement(ctx, logger, q, taskModel.ID, elementType, e)
			if err != nil {
				return err
			}

			knownElements = append(knownElements, elementModel.ID)
		}

		return nil
	}

	if err := upsertElements(TaskElementTypeKnowledge, task.Knowledge); err != nil {
		return queries.Task{}, err
	}

	if err := upsertElements(TaskElementTypeRiskManagement, task.RiskManagement); err != nil {
		return queries.Task{}, err
	}

	if err := upsertElements(TaskElementTypeSkills, task.Skills); err != nil {
		return queries.Task{}, err
	}

	unknownElementCount, err := q.ClearUnknownTaskElements(ctx, queries.ClearUnknownTaskElementsParams{
		TaskID:   taskModel.ID,
		KnownIds: knownElements,
	})
	if err != nil {
		return queries.Task{}, fmt.Errorf("failed to remove unknown elements: %v", err)
	}

	if unknownElementCount == 0 {
		logger.DebugContext(ctx, "No extra task elements to remove")
	} else {
		logger.InfoContext(ctx, "Removed extra task elements", "count", unknownElementCount)
	}

	return taskModel, nil
}

func (m *ACSModel) upsertTaskReference(
	ctx context.Context,
	logger *slog.Logger,
	q *queries.Queries,
	taskID int32,
	order int32,
	reference string,
) (queries.TaskReference, error) {
	referenceModel, err := q.UpsertTaskReference(ctx, queries.UpsertTaskReferenceParams{
		TaskID:   taskID,
		Document: reference,
		Order:    order,
	})
	if err != nil {
		return queries.TaskReference{}, fmt.Errorf("failed to update task reference: %v", err)
	}

	logger.InfoContext(ctx, "Updated task reference", "reference", reference)

	return referenceModel, nil
}

func (m *ACSModel) upsertElement(
	ctx context.Context,
	logger *slog.Logger,
	q *queries.Queries,
	taskID int32,
	elementType TaskElementType,
	element ExternalElement,
) (queries.AcsElement, error) {
	elementModel, err := q.UpsertTaskElement(ctx, queries.UpsertTaskElementParams{
		TaskID:   taskID,
		Type:     elementTypeModel(elementType),
		PublicID: element.ID,
		Content:  element.Content,
	})
	if err != nil {
		return queries.AcsElement{}, fmt.Errorf("failed to update task element: %v", err)
	}

	logger = logger.With("element", fmt.Sprintf("%s%d", elementType, element.ID))
	logger.InfoContext(ctx, "Updated task element")

	knownSubElements := make([]int32, len(element.SubElements))
	for i, s := range element.SubElements {
		subElementModel, err := m.upsertSubElement(ctx, logger, q, elementModel.ID, int32(i), s)
		if err != nil {
			return queries.AcsElement{}, err
		}

		knownSubElements[i] = subElementModel.ID
	}

	unknownSubElementCount, err := q.ClearUnknownSubElements(ctx, queries.ClearUnknownSubElementsParams{
		ElementID: elementModel.ID,
		KnownIds:  knownSubElements,
	})
	if err != nil {
		return queries.AcsElement{}, fmt.Errorf("failed to remove sub-elements: %v", err)
	}

	if unknownSubElementCount == 0 {
		logger.DebugContext(ctx, "No extra sub-elements to remove")
	} else {
		logger.InfoContext(ctx, "Removed extra sub-elements", "count", unknownSubElementCount)
	}

	return elementModel, nil
}

func elementTypeModel(t TaskElementType) queries.AcsElementType {
	switch t {
	case TaskElementTypeKnowledge:
		return queries.AcsElementTypeK

	case TaskElementTypeRiskManagement:
		return queries.AcsElementTypeR

	case TaskElementTypeSkills:
		return queries.AcsElementTypeS
	}

	panic(fmt.Sprintf("Unknown task element type %s", t))
}

const subElementAlphabet = "abcdefghijklmnopqrstuvwxyz"

func (m *ACSModel) upsertSubElement(
	ctx context.Context,
	logger *slog.Logger,
	q *queries.Queries,
	elementID int32,
	order int32,
	subElement ExternalSubElement,
) (queries.AcsSubelement, error) {
	subElementModel, err := q.UpsertSubElement(ctx, queries.UpsertSubElementParams{
		ElementID: elementID,
		Order:     order,
		Content:   subElement.Content,
	})
	if err != nil {
		return queries.AcsSubelement{}, fmt.Errorf("failed to update sub-element: %v", err)
	}

	logger = logger.With("subElement", subElementAlphabet[order])
	logger.InfoContext(ctx, "Updated sub-element")

	return subElementModel, nil
}
