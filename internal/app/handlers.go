package app

import (
	"fmt"
	"net/http"
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

	fmt.Fprintf(w, "%s.%s.%s", acs, areaID, taskID)
}
