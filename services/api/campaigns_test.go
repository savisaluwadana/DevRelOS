package main

import "testing"

func TestCampaignStatusValidation(t *testing.T) {
	for _, status := range []string{"planning", "active", "paused", "completed", "archived"} {
		if !validCampaignStatus(status) { t.Fatalf("expected %s to be valid", status) }
	}
	if validCampaignStatus("sent") { t.Fatal("unexpected campaign status accepted") }
}

func TestCampaignStatusTransitions(t *testing.T) {
	allowed := [][2]string{{"planning", "active"}, {"active", "paused"}, {"paused", "active"}, {"active", "completed"}, {"completed", "archived"}}
	for _, pair := range allowed {
		if !campaignStatusTransitionAllowed(pair[0], pair[1]) { t.Fatalf("expected %s -> %s", pair[0], pair[1]) }
	}
	if campaignStatusTransitionAllowed("archived", "active") { t.Fatal("archived campaign must remain terminal") }
	if campaignStatusTransitionAllowed("planning", "completed") { t.Fatal("planning campaign should activate before completion") }
}
