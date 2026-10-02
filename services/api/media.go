package main

import (
	"net/http"
	"strings"

	mediadomain "github.com/savisaluwadana/DevRelOS/internal/domain/media"
)

func (a *api) registerMediaRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/media-assets", a.listMediaAssets)
	mux.HandleFunc("POST /api/v1/media-assets", a.createMediaAsset)
	mux.HandleFunc("PATCH /api/v1/media-assets/{id}", a.updateMediaAsset)
	mux.HandleFunc("DELETE /api/v1/media-assets/{id}", a.deleteMediaAsset)
	mux.HandleFunc("GET /api/v1/media-clips", a.listMediaClips)
	mux.HandleFunc("POST /api/v1/media-clips", a.createMediaClip)
	mux.HandleFunc("PATCH /api/v1/media-clips/{id}", a.updateMediaClip)
	mux.HandleFunc("DELETE /api/v1/media-clips/{id}", a.deleteMediaClip)
	mux.HandleFunc("POST /api/v1/media-clips/{id}/render", a.queueMediaRender)
}

func (a *api) listMediaAssets(w http.ResponseWriter, r *http.Request) {
	projectID, err := a.projectID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	items, err := a.store.ListMediaAssets(r.Context(), projectID, intQuery(r, "limit", 200))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (a *api) createMediaAsset(w http.ResponseWriter, r *http.Request) {
	var input mediadomain.Asset
	if err := decodeJSON(r, &input); err != nil {
		writeBadRequest(w, err.Error())
		return
	}
	input.Title = strings.TrimSpace(input.Title)
	input.SourcePath = strings.TrimSpace(input.SourcePath)
	input.SourceURL = strings.TrimSpace(input.SourceURL)
	if input.Title == "" {
		writeBadRequest(w, "title is required")
		return
	}
	if input.SourcePath == "" && input.SourceURL == "" {
		writeBadRequest(w, "sourcePath or sourceUrl is required")
		return
	}
	if input.MediaType != "" && input.MediaType != "video" && input.MediaType != "audio" {
		writeBadRequest(w, "invalid mediaType")
		return
	}
	projectID, err := a.projectID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	input.ProjectID = projectID
	created, err := a.store.CreateMediaAsset(r.Context(), input)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (a *api) updateMediaAsset(w http.ResponseWriter, r *http.Request) {
	var input mediadomain.AssetUpdate
	if err := decodeJSON(r, &input); err != nil {
		writeBadRequest(w, err.Error())
		return
	}
	if input.MediaType != nil && *input.MediaType != "video" && *input.MediaType != "audio" {
		writeBadRequest(w, "invalid mediaType")
		return
	}
	if input.TranscriptSegments != nil {
		for _, segment := range *input.TranscriptSegments {
			if segment.StartMS < 0 || segment.EndMS <= segment.StartMS || strings.TrimSpace(segment.Text) == "" {
				writeBadRequest(w, "invalid transcript segment")
				return
			}
		}
	}
	projectID, err := a.projectID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	updated, err := a.store.UpdateMediaAsset(r.Context(), projectID, r.PathValue("id"), input)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (a *api) deleteMediaAsset(w http.ResponseWriter, r *http.Request) {
	projectID, err := a.projectID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	if err := a.store.DeleteMediaAsset(r.Context(), projectID, r.PathValue("id")); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *api) listMediaClips(w http.ResponseWriter, r *http.Request) {
	projectID, err := a.projectID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	items, err := a.store.ListMediaClips(r.Context(), projectID, strings.TrimSpace(r.URL.Query().Get("mediaAssetId")), requestPage(r))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (a *api) createMediaClip(w http.ResponseWriter, r *http.Request) {
	var input mediadomain.Clip
	if err := decodeJSON(r, &input); err != nil {
		writeBadRequest(w, err.Error())
		return
	}
	if input.MediaAssetID == "" || strings.TrimSpace(input.Title) == "" {
		writeBadRequest(w, "mediaAssetId and title are required")
		return
	}
	if input.StartMS < 0 || input.EndMS <= input.StartMS {
		writeBadRequest(w, "endMs must be greater than startMs")
		return
	}
	if input.Score < 0 || input.Score > 100 {
		writeBadRequest(w, "score must be between 0 and 100")
		return
	}
	if input.AspectRatio != "" && input.AspectRatio != "9:16" && input.AspectRatio != "1:1" && input.AspectRatio != "16:9" {
		writeBadRequest(w, "invalid aspectRatio")
		return
	}
	projectID, err := a.projectID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	input.ProjectID = projectID
	created, err := a.store.CreateMediaClip(r.Context(), input)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (a *api) updateMediaClip(w http.ResponseWriter, r *http.Request) {
	var input mediadomain.ClipUpdate
	if err := decodeJSON(r, &input); err != nil {
		writeBadRequest(w, err.Error())
		return
	}
	allowedStatus := map[string]bool{"candidate": true, "approved": true, "rejected": true}
	if input.Status != nil && !allowedStatus[*input.Status] {
		writeBadRequest(w, "status must be candidate, approved or rejected")
		return
	}
	if input.StartMS != nil && *input.StartMS < 0 {
		writeBadRequest(w, "startMs must not be negative")
		return
	}
	if input.EndMS != nil && input.StartMS != nil && *input.EndMS <= *input.StartMS {
		writeBadRequest(w, "endMs must be greater than startMs")
		return
	}
	if input.Score != nil && (*input.Score < 0 || *input.Score > 100) {
		writeBadRequest(w, "score must be between 0 and 100")
		return
	}
	if input.AspectRatio != nil && *input.AspectRatio != "9:16" && *input.AspectRatio != "1:1" && *input.AspectRatio != "16:9" {
		writeBadRequest(w, "invalid aspectRatio")
		return
	}
	projectID, err := a.projectID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	updated, err := a.store.UpdateMediaClip(r.Context(), projectID, r.PathValue("id"), input)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (a *api) deleteMediaClip(w http.ResponseWriter, r *http.Request) {
	projectID, err := a.projectID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	if err := a.store.DeleteMediaClip(r.Context(), projectID, r.PathValue("id")); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *api) queueMediaRender(w http.ResponseWriter, r *http.Request) {
	projectID, err := a.projectID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	clip, err := a.store.GetMediaClip(r.Context(), projectID, r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	if clip == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "clip not found"})
		return
	}
	if clip.Status != "approved" {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "clip must be approved before rendering"})
		return
	}
	job, err := a.store.QueueMediaRender(r.Context(), projectID, r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	queued := "queued"
	_, _ = a.store.UpdateMediaClip(r.Context(), projectID, r.PathValue("id"), mediadomain.ClipUpdate{Status: &queued})
	writeJSON(w, http.StatusAccepted, job)
}
