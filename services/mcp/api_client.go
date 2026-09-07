package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type apiClient struct {
	baseURL string
	token   string
	client  *http.Client
}

func newAPIClient(baseURL, token string) (*apiClient, error) {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		baseURL = "http://127.0.0.1:8080"
	}
	parsed, err := url.Parse(baseURL)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return nil, errors.New("DEVRELOS_API_URL must be an http or https URL")
	}
	if parsed.User != nil {
		return nil, errors.New("DEVRELOS_API_URL must not include credentials")
	}
	return &apiClient{
		baseURL: baseURL,
		token:   strings.TrimSpace(token),
		client:  &http.Client{Timeout: 15 * time.Second},
	}, nil
}

func (c *apiClient) list(ctx context.Context, path string, query url.Values) ([]map[string]any, error) {
	var out []map[string]any
	if err := c.doJSON(ctx, http.MethodGet, path, query, nil, &out); err != nil {
		return nil, err
	}
	if out == nil {
		out = []map[string]any{}
	}
	return out, nil
}

func (c *apiClient) calendar(ctx context.Context, query url.Values) ([]map[string]any, error) {
	var out struct {
		Items []map[string]any `json:"items"`
	}
	if err := c.doJSON(ctx, http.MethodGet, "/api/v1/calendar", query, nil, &out); err != nil {
		return nil, err
	}
	if out.Items == nil {
		out.Items = []map[string]any{}
	}
	return out.Items, nil
}

func (c *apiClient) create(ctx context.Context, path string, body any) (map[string]any, error) {
	var out map[string]any
	if err := c.doJSON(ctx, http.MethodPost, path, nil, body, &out); err != nil {
		return nil, err
	}
	if out == nil {
		out = map[string]any{}
	}
	return out, nil
}

func (c *apiClient) doJSON(ctx context.Context, method, path string, query url.Values, body any, out any) error {
	if !strings.HasPrefix(path, "/api/v1/") && path != "/healthz" {
		return errors.New("MCP API client only permits DevRelOS API paths")
	}
	endpoint := c.baseURL + path
	if len(query) > 0 {
		endpoint += "?" + query.Encode()
	}

	var reader io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(payload)
	}
	req, err := http.NewRequestWithContext(ctx, method, endpoint, reader)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("DevRelOS API request failed: %w", err)
	}
	defer resp.Body.Close()
	limited := io.LimitReader(resp.Body, 4<<20)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		payload, _ := io.ReadAll(limited)
		message := strings.TrimSpace(string(payload))
		if len(message) > 500 {
			message = message[:500]
		}
		return fmt.Errorf("DevRelOS API returned %s: %s", resp.Status, message)
	}
	if out == nil || resp.StatusCode == http.StatusNoContent {
		return nil
	}
	if err := json.NewDecoder(limited).Decode(out); err != nil {
		return fmt.Errorf("decode DevRelOS API response: %w", err)
	}
	return nil
}

func boundedLimit(limit, fallback, maximum int) int {
	if limit <= 0 {
		return fallback
	}
	if limit > maximum {
		return maximum
	}
	return limit
}
