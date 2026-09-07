package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/savisaluwadana/DevRelOS/internal/connectors"
	developersevents "github.com/savisaluwadana/DevRelOS/internal/providers/developersevents"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	registry := connectors.NewRegistry(
		developersevents.New(),
	)

	log.Printf("DevRelOS worker started with %d provider(s)", len(registry.Providers()))
	for _, provider := range registry.Providers() {
		log.Printf("provider=%s capabilities=%v commercial_use_ok=%t", provider.ID(), provider.Capabilities(), provider.Policy(nil).CommercialUseOK)
	}

	if os.Getenv("DEVRELOS_DEMO_FETCH_DEVELOPERS_EVENTS") == "1" {
		provider, _ := registry.Get("developers.events")
		fetchCtx, fetchCancel := context.WithTimeout(ctx, 30*time.Second)
		result, err := provider.Fetch(fetchCtx, nil, connectors.FetchRequest{PageLimit: 10})
		fetchCancel()
		if err != nil {
			log.Printf("developers.events demo fetch failed: %v", err)
		} else {
			log.Printf("developers.events demo fetch: records=%d requests=%d", len(result.Records), result.RequestsMade)
		}
	}

	<-ctx.Done()
	log.Print("DevRelOS worker stopped")
}
