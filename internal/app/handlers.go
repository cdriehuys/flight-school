package app

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/cdriehuys/flight-school/internal/models"
)

func (a *App) homepage(w http.ResponseWriter, r *http.Request) {
	areas, err := a.acsModel.ListAreasByACS(r.Context(), "PA")
	if err != nil {
		a.logger.Error("Failed to list ACS areas.", "error", err)
		a.serverError(w, r, err)
		return
	}

	data := templateData{AreasOfOperation: areas}

	a.render(w, r, http.StatusOK, "index.html.tmpl", data)
}

func (a *App) areaDetail(w http.ResponseWriter, r *http.Request) {
	acs := r.PathValue("acs")
	areaID := r.PathValue("areaID")
	area, err := a.acsModel.GetAreaByID(r.Context(), acs, areaID)
	if err != nil {
		a.logger.Error("Failed to retrieve ACS area.", "error", err)
		a.serverError(w, r, err)
		return
	}

	tasks, err := a.acsModel.ListTasksByArea(r.Context(), area.ID)
	if err != nil {
		a.logger.Error("Failed to list tasks for area.", "error", err, "acs", acs, "area", area.PublicID)
		a.serverError(w, r, err)
		return
	}

	data := templateData{AreaOfOperation: area, Tasks: tasks}

	a.render(w, r, http.StatusOK, "area-detail.html.tmpl", data)
}

func (a *App) taskDetail(w http.ResponseWriter, r *http.Request) {
	acs := r.PathValue("acs")
	areaID := r.PathValue("areaID")
	taskID := r.PathValue("taskID")

	task, err := a.acsModel.GetTaskByArea(r.Context(), acs, areaID, taskID)
	if err != nil {
		a.logger.ErrorContext(
			r.Context(),
			"Failed to retrieve task.",
			"error", err,
			"taskPublicID", fmt.Sprintf("%s.%s.%s", acs, areaID, taskID),
		)
		a.serverError(w, r, err)
		return
	}

	data := templateData{Task: task}

	a.render(w, r, http.StatusOK, "task-detail.html.tmpl", data)
}

func (a *App) setElementConfidence(w http.ResponseWriter, r *http.Request) {
	elementID, err := strconv.Atoi(r.PathValue("elementID"))
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, http.StatusText(http.StatusBadRequest))
		return
	}

	if err := r.ParseForm(); err != nil {
		a.logger.ErrorContext(r.Context(), "Failed to parse form.", "error", err)
		a.serverError(w, r, err)
		return
	}

	confidence, err := getConfidenceFromForm(r.PostForm)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, "Unknown confidence level")
		return
	}

	if err := a.acsModel.SetElementConfidence(r.Context(), elementID, confidence); err != nil {
		a.logger.ErrorContext(r.Context(), "Failed to set element confidence.", "error", err, "elementID", elementID, "confidence", confidence)
		a.serverError(w, r, err)
		return
	}

	task, err := a.acsModel.GetTaskByElementID(r.Context(), elementID)
	if err != nil {
		a.logger.ErrorContext(r.Context(), "Failed to retrieve parent task.", "error", err, "elementID", elementID)
		a.serverError(w, r, err)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/acs/%s/%s/%s", task.ACS, task.AreaPublicID, task.PublicID), http.StatusSeeOther)
}

func getConfidenceFromForm(values url.Values) (models.ElementConfidence, error) {
	if values.Has("high") {
		return models.ElementConfidenceHigh, nil
	}

	if values.Has("medium") {
		return models.ElementConfidenceMedium, nil
	}

	if values.Has("low") {
		return models.ElementConfidenceLow, nil
	}

	return 0, errors.New("unknown confidence level")
}
