package main

import (
	"net/http"
	"strings"

	contentdomain "github.com/savisaluwadana/DevRelOS/internal/domain/content"
	contentintelligence "github.com/savisaluwadana/DevRelOS/internal/intelligence/content"
	"github.com/savisaluwadana/DevRelOS/internal/storage"
)

func (a *api) registerContentRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/content-assets", a.listContentAssets)
	mux.HandleFunc("POST /api/v1/content-assets", a.createContentAsset)
	mux.HandleFunc("PATCH /api/v1/content-assets/{id}", a.updateContentAsset)
	mux.HandleFunc("POST /api/v1/work-items/{id}/content-assets", a.createContentAssetFromWorkItem)
}

func (a *api) listContentAssets(w http.ResponseWriter, r *http.Request) {
	projectID, err := a.projectID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	items, err := a.store.ListContentAssets(r.Context(), projectID, storage.ContentAssetFilter{
		Status:  strings.TrimSpace(r.URL.Query().Get("status")),
		Channel: strings.TrimSpace(r.URL.Query().Get("channel")),
		Limit:   intQuery(r, "limit", 200),
	})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (a *api) createContentAsset(w http.ResponseWriter, r *http.Request) {
	var input contentdomain.Asset
	if err := decodeJSON(r, &input); err != nil {
		writeBadRequest(w, err.Error())
		return
	}
	if !contentintelligence.ValidChannelFormat(strings.TrimSpace(input.Channel), strings.TrimSpace(input.Format)) {
		writeBadRequest(w, "invalid content channel/format combination")
		return
	}
	if strings.TrimSpace(input.Title) == "" {
		writeBadRequest(w, "title is required")
		return
	}
	if input.Status != "" && !validContentStatus(input.Status) {
		writeBadRequest(w, "invalid content status")
		return
	}
	if input.ProjectID == "" {
		var err error
		input.ProjectID, err = a.projectID(r)
		if err != nil {
			writeError(w, err)
			return
		}
	}
	created, err := a.store.CreateContentAsset(r.Context(), input)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (a *api) updateContentAsset(w http.ResponseWriter, r *http.Request) {
	var input contentdomain.AssetUpdate
	if err := decodeJSON(r, &input); err != nil {
		writeBadRequest(w, err.Error())
		return
	}
	if input.Status != nil && !validContentStatus(*input.Status) {
		writeBadRequest(w, "invalid content status")
		return
	}
	projectID, err := a.projectID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	current, err := a.store.GetContentAsset(r.Context(), projectID, r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	if input.Status != nil && *input.Status != current.Status && !validContentTransition(current.Status, *input.Status) {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "invalid content status transition"})
		return
	}
	if input.Status != nil && *input.Status == "published" && input.PublishedURL == nil && strings.TrimSpace(current.PublishedURL) == "" {
		writeBadRequest(w, "publishedUrl is required before marking an asset published")
		return
	}
	updated, err := a.store.UpdateContentAsset(r.Context(), projectID, current.ID, input)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (a *api) createContentAssetFromWorkItem(w http.ResponseWriter, r *http.Request) {
	var input contentdomain.WorkItemConversion
	if err := decodeJSON(r, &input); err != nil {
		writeBadRequest(w, err.Error())
		return
	}
	input.Channel = strings.TrimSpace(input.Channel)
	input.Format = strings.TrimSpace(input.Format)
	input.Audience = strings.TrimSpace(input.Audience)
	if !contentintelligence.ValidChannelFormat(input.Channel, input.Format) {
		writeBadRequest(w, "invalid content channel/format combination")
		return
	}
	projectID, err := a.projectID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	workItem, err := a.store.GetWorkItem(r.Context(), projectID, r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	exists, err := a.store.ContentAssetExistsForWork(r.Context(), projectID, workItem.ID, input.Channel, input.Format)
	if err != nil {
		writeError(w, err)
		return
	}
	if exists {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "this work item already has that content channel/format asset"})
		return
	}
	asset, err := contentintelligence.FromWorkItem(workItem, input.Channel, input.Format, input.Audience)
	if err != nil {
		writeBadRequest(w, err.Error())
		return
	}
	created, err := a.store.CreateContentAsset(r.Context(), asset)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func validContentStatus(status string) bool {
	return map[string]bool{
		"brief": true, "drafting": true, "review": true,
		"approved": true, "published": true, "archived": true,
	}[status]
}

func validContentTransition(from, to string) bool {
	allowed := map[string]map[string]bool{
		"brief":     {"drafting": true, "archived": true},
		"drafting":  {"brief": true, "review": true, "archived": true},
		"review":    {"drafting": true, "approved": true, "archived": true},
		"approved":  {"review": true, "published": true, "archived": true},
		"published": {"archived": true},
		"archived":  {"brief": true},
	}
	return allowed[from][to]
}
