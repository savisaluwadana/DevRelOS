package storage

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/savisaluwadana/DevRelOS/internal/domain/events"
)

// These tests exercise the storage layer against a real PostgreSQL instance.
// They are skipped unless DEVRELOS_TEST_DATABASE_URL points at a database the
// test may create and drop tables in, so `go test ./...` still works without
// one. CI supplies it from a service container.
//
// They cover the two properties that unit tests structurally cannot: that the
// SQL is valid, and that every project-scoped query really is scoped. Project
// scoping was previously only a convention held up by each handler remembering
// to pass a projectID.
func testStore(t *testing.T) *Store {
	t.Helper()
	dsn := os.Getenv("DEVRELOS_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set DEVRELOS_TEST_DATABASE_URL to run storage integration tests")
	}
	store, err := New(context.Background(), dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(store.Close)
	applyMigrations(t, store)
	return store
}

func applyMigrations(t *testing.T, store *Store) {
	t.Helper()
	ctx := context.Background()
	var applied bool
	if err := store.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM information_schema.tables WHERE table_schema='public' AND table_name='projects')`).
		Scan(&applied); err != nil {
		t.Fatalf("probe schema: %v", err)
	}
	if applied {
		return
	}
	files, err := filepath.Glob(filepath.Join("..", "..", "migrations", "*.up.sql"))
	if err != nil {
		t.Fatalf("glob migrations: %v", err)
	}
	sort.Strings(files)
	for _, file := range files {
		sqlBytes, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("read %s: %v", file, err)
		}
		if _, err := store.pool.Exec(ctx, string(sqlBytes)); err != nil {
			t.Fatalf("apply %s: %v", filepath.Base(file), err)
		}
	}
}

// newProject creates an isolated workspace/project pair so tests never observe
// each other's rows.
func newProject(t *testing.T, store *Store, slug string) string {
	t.Helper()
	ctx := context.Background()
	var workspaceID string
	if err := store.pool.QueryRow(ctx,
		`INSERT INTO workspaces (slug, name) VALUES ($1, $1) RETURNING id::text`, slug).Scan(&workspaceID); err != nil {
		t.Fatalf("create workspace %s: %v", slug, err)
	}
	var projectID string
	if err := store.pool.QueryRow(ctx,
		`INSERT INTO projects (workspace_id, slug, name) VALUES ($1, 'p', $2) RETURNING id::text`,
		workspaceID, slug).Scan(&projectID); err != nil {
		t.Fatalf("create project %s: %v", slug, err)
	}
	t.Cleanup(func() {
		// Workspace delete cascades to the project and everything under it.
		_, _ = store.pool.Exec(context.Background(), `DELETE FROM workspaces WHERE id=$1`, workspaceID)
	})
	return projectID
}

func TestListTalksPaginates(t *testing.T) {
	store := testStore(t)
	ctx := context.Background()
	projectID := newProject(t, store, "pagination-"+t.Name())

	// Insert in a known order. ListTalks orders by updated_at DESC, so bump
	// updated_at explicitly to make the ordering deterministic.
	const total = 7
	for i := 0; i < total; i++ {
		if _, err := store.pool.Exec(ctx,
			`INSERT INTO talks (project_id, title, abstract, updated_at)
			 VALUES ($1, $2, '', now() - make_interval(mins => $3))`,
			projectID, fmt.Sprintf("talk-%02d", i), i); err != nil {
			t.Fatalf("insert talk %d: %v", i, err)
		}
	}

	first, err := store.ListTalks(ctx, projectID, Page{Limit: 3})
	if err != nil {
		t.Fatalf("first page: %v", err)
	}
	if len(first) != 3 {
		t.Fatalf("first page returned %d rows, want 3", len(first))
	}

	second, err := store.ListTalks(ctx, projectID, Page{Limit: 3, Offset: 3})
	if err != nil {
		t.Fatalf("second page: %v", err)
	}
	if len(second) != 3 {
		t.Fatalf("second page returned %d rows, want 3", len(second))
	}

	last, err := store.ListTalks(ctx, projectID, Page{Limit: 3, Offset: 6})
	if err != nil {
		t.Fatalf("last page: %v", err)
	}
	if len(last) != 1 {
		t.Fatalf("last page returned %d rows, want 1 (a short page marks the end)", len(last))
	}

	// Pages must be disjoint and together cover every row exactly once.
	seen := map[string]int{}
	for _, page := range [][]events.Talk{first, second, last} {
		for _, talk := range page {
			seen[talk.Title]++
		}
	}
	if len(seen) != total {
		t.Fatalf("paging covered %d distinct talks, want %d", len(seen), total)
	}
	for title, count := range seen {
		if count != 1 {
			t.Fatalf("%s appeared %d times across pages, want exactly once", title, count)
		}
	}

	// Past the end is empty, not an error.
	beyond, err := store.ListTalks(ctx, projectID, Page{Limit: 3, Offset: 99})
	if err != nil {
		t.Fatalf("offset past the end: %v", err)
	}
	if len(beyond) != 0 {
		t.Fatalf("offset past the end returned %d rows, want 0", len(beyond))
	}
}

func TestAllRowsReadsPastTheDefaultLimit(t *testing.T) {
	store := testStore(t)
	ctx := context.Background()
	projectID := newProject(t, store, "allrows-"+t.Name())

	total := DefaultPageLimit + 25
	for i := 0; i < total; i++ {
		if _, err := store.pool.Exec(ctx,
			`INSERT INTO talks (project_id, title, abstract) VALUES ($1, $2, '')`,
			projectID, fmt.Sprintf("talk-%04d", i)); err != nil {
			t.Fatalf("insert talk %d: %v", i, err)
		}
	}

	// A default page stops at the cap: this is exactly the truncation that would
	// corrupt opportunity scoring if it were used there.
	capped, err := store.ListTalks(ctx, projectID, Page{})
	if err != nil {
		t.Fatalf("default page: %v", err)
	}
	if len(capped) != DefaultPageLimit {
		t.Fatalf("default page returned %d rows, want the %d cap", len(capped), DefaultPageLimit)
	}

	all, err := store.ListTalks(ctx, projectID, AllRows())
	if err != nil {
		t.Fatalf("AllRows: %v", err)
	}
	if len(all) != total {
		t.Fatalf("AllRows returned %d rows, want all %d", len(all), total)
	}
}

func TestProjectScopedListsDoNotLeakAcrossProjects(t *testing.T) {
	store := testStore(t)
	ctx := context.Background()
	mine := newProject(t, store, "scope-mine-"+t.Name())
	theirs := newProject(t, store, "scope-theirs-"+t.Name())

	for projectID, title := range map[string]string{mine: "my talk", theirs: "their talk"} {
		if _, err := store.pool.Exec(ctx,
			`INSERT INTO talks (project_id, title, abstract) VALUES ($1, $2, '')`, projectID, title); err != nil {
			t.Fatalf("insert %s: %v", title, err)
		}
		if _, err := store.pool.Exec(ctx,
			`INSERT INTO events (project_id, name, event_type) VALUES ($1, $2, 'conference')`, projectID, title); err != nil {
			t.Fatalf("insert event %s: %v", title, err)
		}
		if _, err := store.pool.Exec(ctx,
			`INSERT INTO communities (project_id, name, platform) VALUES ($1, $2, 'meetup')`, projectID, title); err != nil {
			t.Fatalf("insert community %s: %v", title, err)
		}
	}

	talks, err := store.ListTalks(ctx, mine, AllRows())
	if err != nil {
		t.Fatalf("list talks: %v", err)
	}
	if len(talks) != 1 || talks[0].Title != "my talk" {
		t.Fatalf("ListTalks leaked across projects: %+v", talks)
	}

	eventList, err := store.ListEvents(ctx, mine, AllRows())
	if err != nil {
		t.Fatalf("list events: %v", err)
	}
	if len(eventList) != 1 || eventList[0].Name != "my talk" {
		t.Fatalf("ListEvents leaked across projects: %+v", eventList)
	}

	communities, err := store.ListCommunities(ctx, mine, AllRows())
	if err != nil {
		t.Fatalf("list communities: %v", err)
	}
	if len(communities) != 1 || communities[0].Name != "my talk" {
		t.Fatalf("ListCommunities leaked across projects: %+v", communities)
	}
}

func TestGetMediaClipIsProjectScoped(t *testing.T) {
	store := testStore(t)
	ctx := context.Background()
	mine := newProject(t, store, "clip-mine-"+t.Name())
	theirs := newProject(t, store, "clip-theirs-"+t.Name())

	var assetID string
	if err := store.pool.QueryRow(ctx,
		`INSERT INTO media_assets (project_id, title, media_type, source_path)
		 VALUES ($1, 'recording', 'video', 'talks/recording.mp4') RETURNING id::text`,
		theirs).Scan(&assetID); err != nil {
		t.Fatalf("insert media asset: %v", err)
	}
	var clipID string
	if err := store.pool.QueryRow(ctx,
		`INSERT INTO media_clips (project_id, media_asset_id, title, start_ms, end_ms, status)
		 VALUES ($1, $2, 'clip', 0, 5000, 'approved') RETURNING id::text`,
		theirs, assetID).Scan(&clipID); err != nil {
		t.Fatalf("insert media clip: %v", err)
	}

	found, err := store.GetMediaClip(ctx, theirs, clipID)
	if err != nil {
		t.Fatalf("get own clip: %v", err)
	}
	if found == nil || found.ID != clipID {
		t.Fatalf("expected to find the clip in its own project, got %+v", found)
	}

	// The render gate authorizes on this lookup, so cross-project must be nil.
	leaked, err := store.GetMediaClip(ctx, mine, clipID)
	if err != nil {
		t.Fatalf("cross-project lookup errored: %v", err)
	}
	if leaked != nil {
		t.Fatalf("GetMediaClip returned another project's clip: %+v", leaked)
	}
}

func TestSpeakingCandidatesFilterAndBound(t *testing.T) {
	store := testStore(t)
	ctx := context.Background()
	projectID := newProject(t, store, "candidates-"+t.Name())

	mk := func(table, name string, topics []string, status string) string {
		t.Helper()
		var id string
		var err error
		switch table {
		case "communities":
			err = store.pool.QueryRow(ctx,
				`INSERT INTO communities (project_id, name, platform, topics, status)
				 VALUES ($1,$2,'meetup',$3,$4) RETURNING id::text`, projectID, name, topics, status).Scan(&id)
		case "talks":
			err = store.pool.QueryRow(ctx,
				`INSERT INTO talks (project_id, title, abstract, topics, status)
				 VALUES ($1,$2,'',$3,$4) RETURNING id::text`, projectID, name, topics, status).Scan(&id)
		}
		if err != nil {
			t.Fatalf("insert %s %s: %v", table, name, err)
		}
		return id
	}

	overlapping := mk("communities", "Cloud Native Colombo", []string{"kubernetes", "platform-engineering"}, "active")
	unrelated := mk("communities", "Knitting Circle", []string{"knitting"}, "active")
	paused := mk("communities", "Paused Group", []string{"kubernetes"}, "paused")
	blocked := mk("communities", "Do Not Contact", []string{"kubernetes"}, "do_not_contact")
	warmButUnrelated := mk("communities", "Warm Unrelated", []string{"woodworking"}, "active")

	mk("talks", "Scaling Kubernetes", []string{"kubernetes"}, "ready")
	mk("talks", "Retired Talk", []string{"kubernetes"}, "retired")

	// A relationship keeps an otherwise-unrelated community in the candidate set.
	if _, err := store.pool.Exec(ctx,
		`INSERT INTO relationships (project_id, community_id, stage, strength) VALUES ($1,$2,'engaged',70)`,
		projectID, warmButUnrelated); err != nil {
		t.Fatalf("insert relationship: %v", err)
	}

	candidates, err := store.ListSpeakingCandidates(ctx, projectID, 100)
	if err != nil {
		t.Fatalf("candidates: %v", err)
	}

	byCommunity := map[string]int{}
	for _, candidate := range candidates {
		byCommunity[candidate.Community.ID]++
		if candidate.Talk.Status == "retired" {
			t.Error("a retired talk was offered as a candidate")
		}
	}

	if byCommunity[overlapping] == 0 {
		t.Error("topic-overlapping community produced no candidate")
	}
	if byCommunity[warmButUnrelated] == 0 {
		t.Error("community with an existing relationship was dropped despite no topic overlap")
	}
	if byCommunity[unrelated] != 0 {
		t.Error("community with neither topic overlap nor a relationship was included")
	}
	if byCommunity[paused] != 0 {
		t.Error("paused community was included")
	}
	if byCommunity[blocked] != 0 {
		t.Error("do_not_contact community was included")
	}

	// Topic-overlap candidates must sort ahead of relationship-only ones, so a
	// tight budget keeps the most relevant pairs.
	if len(candidates) > 0 && candidates[0].Community.ID != overlapping {
		t.Errorf("first candidate is %q, want the topic-overlapping community", candidates[0].Community.Name)
	}
}

func TestSpeakingCandidatesRespectBudget(t *testing.T) {
	store := testStore(t)
	ctx := context.Background()
	projectID := newProject(t, store, "budget-"+t.Name())

	// 20 communities x 20 talks all sharing a topic = 400 possible pairs.
	for i := 0; i < 20; i++ {
		// Pass the name pre-formatted: pgx cannot infer an int for a text
		// placeholder, so string concatenation in SQL fails to encode.
		if _, err := store.pool.Exec(ctx,
			`INSERT INTO communities (project_id, name, platform, topics, status)
			 VALUES ($1, $2, 'meetup', ARRAY['kubernetes'], 'active')`,
			projectID, fmt.Sprintf("community-%02d", i)); err != nil {
			t.Fatalf("insert community %d: %v", i, err)
		}
		if _, err := store.pool.Exec(ctx,
			`INSERT INTO talks (project_id, title, abstract, topics, status)
			 VALUES ($1, $2, '', ARRAY['kubernetes'], 'ready')`,
			projectID, fmt.Sprintf("talk-%02d", i)); err != nil {
			t.Fatalf("insert talk %d: %v", i, err)
		}
	}

	all, err := store.ListSpeakingCandidates(ctx, projectID, 1000)
	if err != nil {
		t.Fatalf("candidates: %v", err)
	}
	if len(all) != 400 {
		t.Fatalf("got %d candidates, want the full 400 pairs", len(all))
	}

	bounded, err := store.ListSpeakingCandidates(ctx, projectID, 25)
	if err != nil {
		t.Fatalf("bounded candidates: %v", err)
	}
	if len(bounded) != 25 {
		t.Fatalf("budget of 25 returned %d candidates", len(bounded))
	}

	// Budget scales with the requested result count, between a floor and a cap.
	// The API caps limit at 200, so the reachable budget tops out at 4000; the
	// hard cap guards direct callers.
	if got := SpeakingCandidateBudget(1); got != minSpeakingCandidates {
		t.Errorf("SpeakingCandidateBudget(1) = %d, want the %d floor", got, minSpeakingCandidates)
	}
	if got := SpeakingCandidateBudget(50); got != 1000 {
		t.Errorf("SpeakingCandidateBudget(50) = %d, want 20x the limit", got)
	}
	if got := SpeakingCandidateBudget(200); got != 4000 {
		t.Errorf("SpeakingCandidateBudget(200) = %d, want 4000 at the API's max limit", got)
	}
	if got := SpeakingCandidateBudget(100000); got != maxSpeakingCandidates {
		t.Errorf("SpeakingCandidateBudget(100000) = %d, want the %d cap", got, maxSpeakingCandidates)
	}

	// An over-cap budget passed straight to the query is clamped, not honoured.
	clamped, err := store.ListSpeakingCandidates(ctx, projectID, 999999)
	if err != nil {
		t.Fatalf("over-cap budget: %v", err)
	}
	if len(clamped) != 400 {
		t.Fatalf("over-cap budget returned %d candidates, want the 400 available", len(clamped))
	}
}

// The filter-based list queries carried their own Limit long before the Page
// type existed, and none of them applied an offset: pages overlapped and an
// offset past the end still returned a full page. These cover all five.
func TestFilterBasedListsApplyOffset(t *testing.T) {
	store := testStore(t)
	ctx := context.Background()
	projectID := newProject(t, store, "filteroffset-"+t.Name())

	const total = 7
	for i := 0; i < total; i++ {
		if _, err := store.pool.Exec(ctx,
			`INSERT INTO signals (project_id, provider, external_id, title, body, status, occurred_at)
			 VALUES ($1,'manual',$2,$3,'body','new', now() - make_interval(mins => $4))`,
			projectID, fmt.Sprintf("ext-%02d", i), fmt.Sprintf("signal-%02d", i), i); err != nil {
			t.Fatalf("insert signal %d: %v", i, err)
		}
		if _, err := store.pool.Exec(ctx,
			`INSERT INTO work_items (project_id, kind, title, status, source_type)
			 VALUES ($1,'content_brief',$2,'backlog','manual')`,
			projectID, fmt.Sprintf("work-%02d", i)); err != nil {
			t.Fatalf("insert work item %d: %v", i, err)
		}
		if _, err := store.pool.Exec(ctx,
			`INSERT INTO content_assets (project_id, title, channel, format, status)
			 VALUES ($1,$2,'blog','article','brief')`,
			projectID, fmt.Sprintf("content-%02d", i)); err != nil {
			t.Fatalf("insert content asset %d: %v", i, err)
		}
		if _, err := store.pool.Exec(ctx,
			`INSERT INTO feedback_items (project_id, title, status) VALUES ($1,$2,'new')`,
			projectID, fmt.Sprintf("feedback-%02d", i)); err != nil {
			t.Fatalf("insert feedback %d: %v", i, err)
		}
	}

	// ids collects one page worth of identity strings for a list.
	type pager func(limit, offset int) ([]string, error)

	pagers := map[string]pager{
		"signals": func(limit, offset int) ([]string, error) {
			items, err := store.ListSignals(ctx, projectID, SignalFilter{Limit: limit, Offset: offset})
			out := make([]string, 0, len(items))
			for _, i := range items {
				out = append(out, i.ID)
			}
			return out, err
		},
		"work items": func(limit, offset int) ([]string, error) {
			items, err := store.ListWorkItems(ctx, projectID, WorkItemFilter{Limit: limit, Offset: offset})
			out := make([]string, 0, len(items))
			for _, i := range items {
				out = append(out, i.ID)
			}
			return out, err
		},
		"content assets": func(limit, offset int) ([]string, error) {
			items, err := store.ListContentAssets(ctx, projectID, ContentAssetFilter{Limit: limit, Offset: offset})
			out := make([]string, 0, len(items))
			for _, i := range items {
				out = append(out, i.ID)
			}
			return out, err
		},
		"feedback": func(limit, offset int) ([]string, error) {
			items, err := store.ListFeedback(ctx, projectID, FeedbackFilter{Limit: limit, Offset: offset})
			out := make([]string, 0, len(items))
			for _, i := range items {
				out = append(out, i.ID)
			}
			return out, err
		},
	}

	for name, page := range pagers {
		t.Run(name, func(t *testing.T) {
			seen := map[string]int{}
			for offset := 0; offset < total; offset += 3 {
				got, err := page(3, offset)
				if err != nil {
					t.Fatalf("offset %d: %v", offset, err)
				}
				for _, id := range got {
					seen[id]++
				}
			}
			if len(seen) != total {
				t.Fatalf("paging covered %d distinct rows, want %d", len(seen), total)
			}
			for id, count := range seen {
				if count != 1 {
					t.Fatalf("%s appeared %d times across pages, want once", id, count)
				}
			}
			beyond, err := page(3, 999)
			if err != nil {
				t.Fatalf("offset past the end: %v", err)
			}
			if len(beyond) != 0 {
				t.Fatalf("offset past the end returned %d rows, want 0", len(beyond))
			}
		})
	}
}

func TestListPainPointsAppliesOffset(t *testing.T) {
	store := testStore(t)
	ctx := context.Background()
	projectID := newProject(t, store, "ppoffset-"+t.Name())

	const total = 5
	for i := 0; i < total; i++ {
		if _, err := store.pool.Exec(ctx,
			`INSERT INTO pain_points (project_id, key, title, summary, severity, evidence_count, status)
			 VALUES ($1,$2,$3,'s',$4,1,'active')`,
			projectID, fmt.Sprintf("key-%02d", i), fmt.Sprintf("pain-%02d", i), 90-i); err != nil {
			t.Fatalf("insert pain point %d: %v", i, err)
		}
	}

	seen := map[string]int{}
	for offset := 0; offset < total; offset += 2 {
		items, err := store.ListPainPoints(ctx, projectID, "", 2, offset)
		if err != nil {
			t.Fatalf("offset %d: %v", offset, err)
		}
		for _, item := range items {
			seen[item.ID]++
		}
	}
	if len(seen) != total {
		t.Fatalf("paging covered %d distinct pain points, want %d", len(seen), total)
	}
	beyond, err := store.ListPainPoints(ctx, projectID, "", 2, 999)
	if err != nil {
		t.Fatalf("offset past the end: %v", err)
	}
	if len(beyond) != 0 {
		t.Fatalf("offset past the end returned %d rows, want 0", len(beyond))
	}
}
