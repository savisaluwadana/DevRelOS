package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	connectorruntime "github.com/savisaluwadana/DevRelOS/internal/connectors"
	connectordomain "github.com/savisaluwadana/DevRelOS/internal/domain/connectors"
	developersevents "github.com/savisaluwadana/DevRelOS/internal/providers/developersevents"
	"github.com/savisaluwadana/DevRelOS/internal/storage"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	store, err := storage.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer store.Close()

	registry := connectorruntime.NewRegistry(
		developersevents.New(),
	)
	log.Printf("DevRelOS worker started with %d provider(s)", len(registry.Providers()))

	pollEvery := 10 * time.Second
	if value := os.Getenv("DEVRELOS_WORKER_POLL_INTERVAL"); value != "" {
		if parsed, parseErr := time.ParseDuration(value); parseErr == nil && parsed >= time.Second {
			pollEvery = parsed
		}
	}

	processAvailable(ctx, store, registry)
	ticker := time.NewTicker(pollEvery)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Print("DevRelOS worker stopped")
			return
		case <-ticker.C:
			processAvailable(ctx, store, registry)
		}
	}
}

func processAvailable(ctx context.Context, store *storage.Store, registry *connectorruntime.Registry) {
	for {
		processed, err := processNext(ctx, store, registry)
		if err != nil {
			log.Printf("connector queue error: %v", err)
			return
		}
		if !processed {
			return
		}
	}
}

func processNext(ctx context.Context, store *storage.Store, registry *connectorruntime.Registry) (bool, error) {
	claimed, err := store.ClaimNextConnectorRun(ctx)
	if err != nil {
		return false, err
	}
	if claimed == nil {
		return false, nil
	}

	run := claimed.Run
	connector := claimed.Connector
	fail := func(cause error) (bool, error) {
		run.Status = "failed"
		run.Error = cause.Error()
		if finishErr := store.FinishConnectorRun(ctx, run); finishErr != nil {
			return true, fmt.Errorf("run failed: %v; recording failure: %w", cause, finishErr)
		}
		log.Printf("connector run failed id=%s provider=%s error=%v", run.ID, connector.Provider, cause)
		return true, nil
	}

	if !connector.Enabled {
		return fail(fmt.Errorf("connector is disabled"))
	}
	provider, ok := registry.Get(connector.Provider)
	if !ok {
		return fail(fmt.Errorf("provider %q is not registered in this worker", connector.Provider))
	}
	if err := provider.ValidateConfig(connector.Config); err != nil {
		return fail(fmt.Errorf("invalid connector config: %w", err))
	}

	policy := provider.Policy(connector.Config)
	if !policy.CommercialUseOK && stringValue(connector.Config, "usage_mode") != "noncommercial" {
		return fail(fmt.Errorf("provider policy blocks ingestion by default; reviewed non-commercial usage must set config.usage_mode=noncommercial"))
	}

	pageLimit := intValue(connector.Config, "page_limit", 100)
	if pageLimit < 1 {
		pageLimit = 1
	}
	if pageLimit > 1000 {
		pageLimit = 1000
	}

	fetchCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
	result, fetchErr := provider.Fetch(fetchCtx, connector.Config, connectorruntime.FetchRequest{
		Cursor:    run.Cursor,
		PageLimit: pageLimit,
	})
	cancel()
	if fetchErr != nil {
		return fail(fetchErr)
	}

	run.RequestsMade = result.RequestsMade
	run.ItemsFetched = len(result.Records)
	run.Cursor = result.NextCursor
	run.Warnings = result.Warnings
	cost := result.CostUSD
	run.ProviderCostUSD = &cost

	for _, record := range result.Records {
		payloadJSON, marshalErr := json.Marshal(record.Payload)
		if marshalErr != nil {
			run.ItemsSkipped++
			run.Warnings = append(run.Warnings, "skipped record with unserializable payload")
			continue
		}
		hash := sha256.Sum256(payloadJSON)
		var rawPayload map[string]any
		if policy.StoreRawPayload {
			rawPayload = record.Payload
		}

		_, inserted, upsertErr := store.UpsertSourceRecord(ctx, connectordomain.SourceRecord{
			WorkspaceID:     connector.WorkspaceID,
			Provider:        connector.Provider,
			ExternalID:      record.ExternalID,
			CanonicalURL:    record.CanonicalURL,
			SourceTimestamp: record.SourceTimestamp,
			ContentHash:     hex.EncodeToString(hash[:]),
			RawPayload:      rawPayload,
			Provenance: map[string]any{
				"connector_id": connector.ID,
				"run_id":       run.ID,
				"fetched_at":   time.Now().UTC().Format(time.RFC3339),
				"provider_policy": map[string]any{
					"commercial_use_ok": policy.CommercialUseOK,
					"store_raw_payload": policy.StoreRawPayload,
				},
			},
		})
		if upsertErr != nil {
			run.ItemsSkipped++
			run.Warnings = append(run.Warnings, fmt.Sprintf("failed to persist source record %s", record.ExternalID))
			continue
		}
		if inserted {
			run.ItemsCreated++
		} else {
			run.ItemsUpdated++
		}
	}

	run.Status = "succeeded"
	if err := store.FinishConnectorRun(ctx, run); err != nil {
		return true, err
	}
	log.Printf("connector run succeeded id=%s provider=%s fetched=%d created=%d updated=%d skipped=%d cost_usd=%.4f",
		run.ID, connector.Provider, run.ItemsFetched, run.ItemsCreated, run.ItemsUpdated, run.ItemsSkipped, result.CostUSD)
	return true, nil
}

func stringValue(config map[string]any, key string) string {
	value, _ := config[key].(string)
	return value
}

func intValue(config map[string]any, key string, fallback int) int {
	value, ok := config[key]
	if !ok {
		return fallback
	}
	switch typed := value.(type) {
	case float64:
		return int(typed)
	case int:
		return typed
	case json.Number:
		parsed, err := typed.Int64()
		if err == nil {
			return int(parsed)
		}
	}
	return fallback
}
