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

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/savisaluwadana/DevRelOS/internal/domain/events"
	"github.com/savisaluwadana/DevRelOS/internal/intelligence/opportunities"
	signalsintel "github.com/savisaluwadana/DevRelOS/internal/intelligence/signals"
	"github.com/savisaluwadana/DevRelOS/internal/storage"
)

type api struct {
	store   *storage.Store
	metrics *apiMetrics
	limiter *rateLimiter
}

func main() {
	ctx := context.Background()
	store, err := storage.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer store.Close()

	if strings.EqualFold(strings.TrimSpace(os.Getenv("DEVRELOS_REQUIRE_AUTH")), "true") && strings.TrimSpace(os.Getenv("DEVRELOS_API_TOKEN")) == "" {
		log.Fatal("DEVRELOS_API_TOKEN is required when DEVRELOS_REQUIRE_AUTH=true")
	}

	// Resolve the clustering vocabulary up front so a malformed override file is
	// a startup failure rather than a surprise on the first Signal Radar request.
	if _, err := signalsintel.ActiveRuleset(); err != nil {
		log.Fatalf("clustering rules: %v", err)
	}

	a := &api{store: store, metrics: newAPIMetrics(), limiter: newRateLimiterFromEnv()}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", a.health)
	mux.HandleFunc("GET /readyz", a.ready)
	mux.HandleFunc("GET /metrics", a.metricsHandler)
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
	mux.HandleFunc("PATCH /api/v1/connectors/{id}/schedule", a.updateConnectorSchedule)
	mux.HandleFunc("GET /api/v1/connectors/{id}/runs", a.listConnectorRuns)
	mux.HandleFunc("POST /api/v1/connectors/{id}/runs", a.queueConnectorRun)
	mux.HandleFunc("GET /api/v1/opportunities/speaking", a.listSpeakingOpportunities)
	mux.HandleFunc("GET /api/v1/opportunities/cfps", a.listCFPOpportunities)
	a.registerSignalRoutes(mux)
	a.registerOutreachRoutes(mux)
	a.registerWorkItemRoutes(mux)
	a.registerMediaRoutes(mux)
	a.registerIdentityRoutes(mux)
	a.registerSessionRoutes(mux)
	a.registerSecretRoutes(mux)
	a.registerCampaignRoutes(mux)
	a.registerCalendarRoutes(mux)

	addr := envOr("DEVRELOS_HTTP_ADDR", ":8080")
	// withRoutePattern must sit innermost so it sees the same request ServeMux
	// annotates with the matched pattern; withAuth below passes down a clone.
	core := a.withAuth(withRoutePattern(mux))
	core = a.limiter.Middleware(core)
	core = a.withObservability(core)
	server := &http.Server{
		Addr:              addr,
		Handler:           withMiddleware(core),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}

	log.Printf("DevRelOS API listening on %s", addr)
	if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}

func (a *api) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "service": "devrelos-api"})
}

func (a *api) ready(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	if err := a.store.Ping(ctx); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"ok": false, "service": "devrelos-api", "database": "unavailable"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "service": "devrelos-api", "database": "ready"})
}

func (a *api) projectID(r *http.Request) (string, error) {
	if id := strings.TrimSpace(r.URL.Query().Get("projectId")); id != "" {
		return id, nil
	}
	workspaceID, err := a.requestWorkspaceID(r)
	if err != nil {
		return "", err
	}
	return a.store.DefaultProjectIDForWorkspace(r.Context(), workspaceID)
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
	if err := a.fillHighFitCFPs(r, &d); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, d)
}

// highFitCFPScore is the score at or above which a CFP is worth surfacing on
// the Command Center.
const highFitCFPScore = 70

// fillHighFitCFPs derives the Command Center's high-fit panel from the live
// CFP/talk scorer, which is the only thing that actually computes fit. The
// stored cfps.fit_score column it used to read is never populated.
func (a *api) fillHighFitCFPs(r *http.Request, d *events.Dashboard) error {
	projectID, err := a.projectID(r)
	if err != nil {
		return err
	}
	cfps, err := a.store.ListCFPs(r.Context(), projectID, storage.AllRows())
	if err != nil {
		return err
	}
	if len(cfps) == 0 {
		return nil
	}
	eventsList, err := a.store.ListEvents(r.Context(), projectID, storage.AllRows())
	if err != nil {
		return err
	}
	talks, err := a.store.ListTalks(r.Context(), projectID, storage.AllRows())
	if err != nil {
		return err
	}
	submissions, err := a.store.ListSubmissions(r.Context(), projectID, storage.AllRows())
	if err != nil {
		return err
	}

	ranked := opportunities.RankCFPs(cfps, eventsList, talks, submissions, time.Now().UTC())
	seen := map[string]bool{}
	for _, item := range ranked {
		if item.Score < highFitCFPScore || item.CFP.Status != "open" || seen[item.CFP.ID] {
			continue
		}
		seen[item.CFP.ID] = true
		// Surface the score the panel is filtering on, so the UI and the
		// opportunities workspace agree.
		score := item.Score
		cfp := item.CFP
		cfp.FitScore = &score
		d.HighFitCFPs = append(d.HighFitCFPs, cfp)
		if len(d.HighFitCFPs) == 5 {
			break
		}
	}
	return nil
}

func (a *api) listEvents(w http.ResponseWriter, r *http.Request) {
	projectID, err := a.projectID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	items, err := a.store.ListEvents(r.Context(), projectID, requestPage(r))
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
	projectID, err := a.projectID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	input.ProjectID = projectID
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
	items, err := a.store.ListCFPs(r.Context(), projectID, requestPage(r))
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
	projectID, err := a.projectID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	created, err := a.store.CreateScopedCFP(r.Context(), projectID, input)
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
	items, err := a.store.ListTalks(r.Context(), projectID, requestPage(r))
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
	projectID, err := a.projectID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	input.ProjectID = projectID
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
	items, err := a.store.ListSubmissions(r.Context(), projectID, requestPage(r))
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
	projectID, err := a.projectID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	created, err := a.store.CreateScopedSubmission(r.Context(), projectID, input)
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
	allowed := map[string]bool{"draft": true, "needs_work": true, "ready": true, "submitted": true, "accepted": true, "rejected": true, "withdrawn": true}
	if !allowed[input.Status] {
		writeBadRequest(w, "invalid submission status")
		return
	}
	projectID, err := a.projectID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	if err := a.store.UpdateScopedSubmissionStatus(r.Context(), projectID, r.PathValue("id"), input.Status); err != nil {
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
	items, err := a.store.ListCommunities(r.Context(), projectID, requestPage(r))
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
	projectID, err := a.projectID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	input.ProjectID = projectID
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
	if errors.Is(err, pgx.ErrNoRows) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}

	// Classify database errors that are caused by the request rather than by
	// the server. Everything used to collapse into 500 "internal server
	// error", so a client sending an invalid status value or a malformed UUID
	// got an opaque server error, could not tell a bad request from a real
	// fault, and every such request was counted as a server failure in the
	// metrics.
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505": // unique_violation
			writeJSON(w, http.StatusConflict, map[string]string{
				"error":  "that record already exists",
				"detail": pgConstraintDetail(pgErr),
			})
			return
		case "23514": // check_violation - an enum or range the schema rejects
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error":  "a field has a value this record type does not allow",
				"detail": pgConstraintDetail(pgErr),
			})
			return
		case "23503": // foreign_key_violation - a referenced record is missing
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error":  "a referenced record does not exist",
				"detail": pgConstraintDetail(pgErr),
			})
			return
		case "23502": // not_null_violation
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error":  "a required field is missing",
				"detail": pgErr.ColumnName,
			})
			return
		case "22P02", // invalid_text_representation, e.g. a malformed UUID
			"22007", // invalid_datetime_format
			"22003": // numeric_value_out_of_range
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error": "a field is not in the expected format",
			})
			return
		}
	}

	log.Printf("request failed: %v", err)
	writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
}

// pgConstraintDetail names the offending constraint or column without leaking
// SQL, table internals or row values into the response.
func pgConstraintDetail(pgErr *pgconn.PgError) string {
	if pgErr.ColumnName != "" {
		return pgErr.ColumnName
	}
	return pgErr.ConstraintName
}

func withMiddleware(next http.Handler) http.Handler {
	// Read once at wiring time rather than on every request. The value is a
	// fixed configured origin, so no Vary: Origin is warranted (and every
	// response already carries Cache-Control: no-store).
	origin := envOr("DEVRELOS_CORS_ORIGIN", "http://localhost:3000")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Request-ID")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func envOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
