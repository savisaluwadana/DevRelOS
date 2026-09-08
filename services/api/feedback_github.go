package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

var githubSlugPart = regexp.MustCompile(`^[A-Za-z0-9_.-]+$`)

func parseGitHubRepository(value string) (string, string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", "", errors.New("githubRepository is required")
	}
	if strings.Contains(value, "://") {
		parsed, err := url.Parse(value)
		if err != nil || !strings.EqualFold(parsed.Hostname(), "github.com") {
			return "", "", errors.New("githubRepository URL must use github.com")
		}
		value = strings.Trim(parsed.Path, "/")
	}
	value = strings.TrimSuffix(value, ".git")
	parts := strings.Split(value, "/")
	if len(parts) != 2 || !githubSlugPart.MatchString(parts[0]) || !githubSlugPart.MatchString(parts[1]) {
		return "", "", errors.New("githubRepository must be owner/repo or a github.com repository URL")
	}
	return parts[0], parts[1], nil
}

func (a *api) feedbackGitHubPrefill(w http.ResponseWriter, r *http.Request) {
	projectID, err := a.projectID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	item, err := a.store.GetFeedback(r.Context(), projectID, r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	owner, repo, err := parseGitHubRepository(item.GitHubRepository)
	if err != nil {
		writeBadRequest(w, err.Error())
		return
	}

	title := strings.TrimSpace(item.GitHubIssueTitle)
	if title == "" {
		title = strings.TrimSpace(item.Title)
	}
	body := strings.TrimSpace(item.GitHubIssueBody)
	if body == "" {
		body = strings.TrimSpace(item.Summary)
		if item.Persona != "" {
			body += "\n\nAffected persona: " + item.Persona
		}
		if item.Component != "" {
			body += "\nComponent: " + item.Component
		}
	}
	query := url.Values{}
	query.Set("title", title)
	query.Set("body", body)
	writeJSON(w, http.StatusOK, map[string]string{
		"url": fmt.Sprintf("https://github.com/%s/%s/issues/new?%s", owner, repo, query.Encode()),
	})
}

func (a *api) syncFeedbackGitHubIssue(w http.ResponseWriter, r *http.Request) {
	projectID, err := a.projectID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	item, err := a.store.GetFeedback(r.Context(), projectID, r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	if item.GitHubIssueNumber == nil || *item.GitHubIssueNumber < 1 {
		writeBadRequest(w, "link a GitHub issue number before syncing")
		return
	}
	owner, repo, err := parseGitHubRepository(item.GitHubRepository)
	if err != nil {
		writeBadRequest(w, err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		fmt.Sprintf("https://api.github.com/repos/%s/%s/issues/%d", owner, repo, *item.GitHubIssueNumber), nil)
	if err != nil {
		writeError(w, err)
		return
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "DevRelOS/feedback-sync")

	client := &http.Client{Timeout: 6 * time.Second}
	response, err := client.Do(req)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": "GitHub issue lookup failed"})
		return
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusNotFound {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "GitHub issue not found or not public"})
		return
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": fmt.Sprintf("GitHub returned HTTP %d", response.StatusCode)})
		return
	}

	var payload struct {
		HTMLURL     string     `json:"html_url"`
		Title       string     `json:"title"`
		State       string     `json:"state"`
		Comments    int        `json:"comments"`
		UpdatedAt   time.Time  `json:"updated_at"`
		ClosedAt    *time.Time `json:"closed_at"`
		PullRequest any        `json:"pull_request"`
		User        struct {
			Login string `json:"login"`
		} `json:"user"`
		Labels []struct {
			Name string `json:"name"`
		} `json:"labels"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": "GitHub returned an unreadable issue payload"})
		return
	}
	if payload.PullRequest != nil {
		writeBadRequest(w, "linked GitHub number is a pull request, not an issue")
		return
	}
	labels := make([]string, 0, len(payload.Labels))
	for _, label := range payload.Labels {
		if label.Name != "" {
			labels = append(labels, label.Name)
		}
	}
	metadata := map[string]any{
		"githubIssueState":     payload.State,
		"githubIssueUpdatedAt": payload.UpdatedAt.UTC().Format(time.RFC3339),
		"githubIssueAuthor":    payload.User.Login,
		"githubIssueComments":  payload.Comments,
		"githubIssueLabels":    labels,
	}
	if payload.ClosedAt != nil {
		metadata["githubIssueClosedAt"] = payload.ClosedAt.UTC().Format(time.RFC3339)
	}
	if err := a.store.UpdateFeedbackGitHubSync(r.Context(), projectID, item.ID, payload.HTMLURL, payload.Title, metadata); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"state":     payload.State,
		"title":     payload.Title,
		"url":       payload.HTMLURL,
		"updatedAt": payload.UpdatedAt,
		"comments":  payload.Comments,
		"labels":    labels,
		"note":      "DevRelOS does not automatically change feedback lifecycle state from GitHub issue state.",
	})
}
