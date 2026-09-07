package main

import (
	"net/http"
	"strings"
	"time"
)

func (a *api) registerCalendarRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/calendar", a.listCalendar)
}

func (a *api) listCalendar(w http.ResponseWriter, r *http.Request) {
	projectID, err := a.projectID(r)
	if err != nil {
		writeError(w, err)
		return
	}

	now := time.Now().UTC()
	from := now.Add(-7 * 24 * time.Hour)
	to := now.Add(90 * 24 * time.Hour)
	if value := strings.TrimSpace(r.URL.Query().Get("from")); value != "" {
		parsed, parseErr := time.Parse(time.RFC3339, value)
		if parseErr != nil {
			writeBadRequest(w, "from must be RFC3339")
			return
		}
		from = parsed.UTC()
	}
	if value := strings.TrimSpace(r.URL.Query().Get("to")); value != "" {
		parsed, parseErr := time.Parse(time.RFC3339, value)
		if parseErr != nil {
			writeBadRequest(w, "to must be RFC3339")
			return
		}
		to = parsed.UTC()
	}
	if !to.After(from) {
		writeBadRequest(w, "to must be after from")
		return
	}
	if to.Sub(from) > 366*24*time.Hour {
		writeBadRequest(w, "calendar range cannot exceed 366 days")
		return
	}

	items, err := a.store.ListCalendarItems(r.Context(), projectID, from, to)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"from": from,
		"to":   to,
		"items": items,
	})
}
