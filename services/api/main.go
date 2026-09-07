package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/savisaluwadana/DevRelOS/internal/domain/events"
	"github.com/savisaluwadana/DevRelOS/internal/storage"
)

type api struct {
	store *storage.Store
}

func main() {
	ctx := context.Background()
	store, err := storage.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer store.Close()

	a := &api{store: store}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", a.health)
	mux.HandleFunc("GET /api/v1/dashboard", a.dashboard)
	mux.HandleFunc("GET /api/v1/events", a.listEvents)
	mux.HandleFunc("POST /api/v1/events", a.createEvent)
	mux.HandleFunc("GET /api/v1/cfps", a.listCFPs)
	mux.HandleFunc("POST /api/v1/cfps", a.createCFP)
	mux.HandleFunc("GET /api/v1/talks", a.listTalks)
	mux.HandleFunc("POST /api/v1/talks", a.createTalk)
	mux.HandleFunc("GET /api/v1/submissions", a.listSubmissions)
	mux.HandleFunc("POST /api/v1/submissions", a.createSubmission)
	mux.HandleFunc("PATCH /api/v1/submissions/{id}/status", a.updateSubmissionStatus)
	mux.HandleFunc("GET /api/v1/communities", a.listCommunities)
	mux.HandleFunc("POST /api/v1/communities", a.createCommunity)
	mux.HandleFunc("GET /api/v1/connectors", a.listConnectors)
	mux.HandleFunc("POST /api/v1/connectors", a.createConnector)
	mux.HandleFunc("GET /api/v1/connectors/{id}/runs", a.listConnectorRuns)
	mux.HandleFunc("POST /api/v1/connectors/{id}/runs", a.queueConnectorRun)
	a.registerSignalRoutes(mux)

	addr := envOr("DEVRELOS_HTTP_ADDR", ":8080")
	server := &http.Server{
		Addr:              addr,
		Handler:           withMiddleware(mux),
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	log.Printf("DevRelOS API listening on %s", addr)
	if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}

func (a *api) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "service": "devrelos-api"})
}

func (a *api) projectID(r *http.Request) (string, error) {
	if id := strings.TrimSpace(r.URL.Query().Get("projectId")); id != "" {
		return id, nil
	}
	return a.store.DefaultProjectID(r.Context())
}

func (a *api) dashboard(w http.ResponseWriter, r *http.Request) {
	projectID, err := a.projectID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	d, err := a.store.Dashboard(r.Context(), projectID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, d)
}

func (a *api) listEvents(w http.ResponseWriter, r *http.Request) {
	projectID, err := a.projectID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	items, err := a.store.ListEvents(r.Context(), projectID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (a *api) createEvent(w http.ResponseWriter, r *http.Request) {
	var input events.Event
	if err := decodeJSON(r, &input); err != nil {
		writeBadRequest(w, err.Error())
		return
	}
	if strings.TrimSpace(input.Name) == "" {
		writeBadRequest(w, "name is required")
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
	created, err := a.store.CreateEvent(r.Context(), input)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (a *api) listCFPs(w http.ResponseWriter, r *http.Request) {
	projectID, err := a.projectID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	items, err := a.store.ListCFPs(r.Context(), projectID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (a *api) createCFP(w http.ResponseWriter, r *http.Request) {
	var input events.CFP
	if err := decodeJSON(r, &input); err != nil {
		writeBadRequest(w, err.Error())
		return
	}
	if input.EventID == "" {
		writeBadRequest(w, "eventId is required")
		return
	}
	if input.FitScore != nil && (*input.FitScore < 0 || *input.FitScore > 100) {
		writeBadRequest(w, "fitScore must be between 0 and 100")
		return
	}
	created, err := a.store.CreateCFP(r.Context(), input)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (a *api) listTalks(w http.ResponseWriter, r *http.Request) {
	projectID, err := a.projectID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	items, err := a.store.ListTalks(r.Context(), projectID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (a *api) createTalk(w http.ResponseWriter, r *http.Request) {
	var input events.Talk
	if err := decodeJSON(r, &input); err != nil {
		writeBadRequest(w, err.Error())
		return
	}
	if strings.TrimSpace(input.Title) == "" {
		writeBadRequest(w, "title is required")
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
	created, err := a.store.CreateTalk(r.Context(), input)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (a *api) listSubmissions(w http.ResponseWriter, r *http.Request) {
	projectID, err := a.projectID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	items, err := a.store.ListSubmissions(r.Context(), projectID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (a *api) createSubmission(w http.ResponseWriter, r *http.Request) {
	var input events.Submission
	if err := decodeJSON(r, &input); err != nil {
		writeBadRequest(w, err.Error())
		return
	}
	if input.CFPID == "" || input.TalkID == "" {
		writeBadRequest(w, "cfpId and talkId are required")
		return
	}
	created, err := a.store.CreateSubmission(r.Context(), input)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (a *api) updateSubmissionStatus(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Status string `json:"status"`
	}
	if err := decodeJSON(r, &input); err != nil {
		writeBadRequest(w, err.Error())
		return
	}
	allowed := map[string]bool{
		"draft": true, "needs_work": true, "ready": true, "submitted": true,
		"accepted": true, "rejected": true, "withdrawn": true,
	}
	if !allowed[input.Status] {
		writeBadRequest(w, "invalid submission status")
		return
	}
	if err := a.store.UpdateSubmissionStatus(r.Context(), r.PathValue("id"), input.Status); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": input.Status})
}

func (a *api) listCommunities(w http.ResponseWriter, r *http.Request) {
	projectID, err := a.projectID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	items, err := a.store.ListCommunities(r.Context(), projectID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (a *api) createCommunity(w http.ResponseWriter, r *http.Request) {
	var input events.Community
	if err := decodeJSON(r, &input); err != nil {
		writeBadRequest(w, err.Error())
		return
	}
	if strings.TrimSpace(input.Name) == "" {
		writeBadRequest(w, "name is required")
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
	created, err := a.store.CreateCommunity(r.Context(), input)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func decodeJSON(r *http.Request, dst any) error {
	decoder := json.NewDecoder(io.LimitReader(r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	return decoder.Decode(dst)
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeBadRequest(w http.ResponseWriter, message string) {
	writeJSON(w, http.StatusBadRequest, map[string]string{"error": message})
}

func writeError(w http.ResponseWriter, err error) {
	log.Printf("request failed: %v", err)
	writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
}

func withMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, OPTIONS")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		started := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(started).Round(time.Millisecond))
	})
}

func envOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
