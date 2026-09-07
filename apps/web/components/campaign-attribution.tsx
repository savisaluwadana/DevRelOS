"use client";

import { useEffect, useMemo, useState } from "react";
import { useRouter } from "next/navigation";
import type { Campaign } from "@/lib/campaign-shared";

const apiBase = "/api/devrelos";

export function CampaignAttribution({
  entityType,
  entityId,
  channel = "",
  label = "Attribute to campaign"
}: {
  entityType: string;
  entityId: string;
  channel?: string;
  label?: string;
}) {
  const router = useRouter();
  const [campaigns, setCampaigns] = useState<Campaign[]>([]);
  const [campaignId, setCampaignId] = useState("");
  const [cost, setCost] = useState("0");
  const [open, setOpen] = useState(false);
  const [busy, setBusy] = useState(false);
  const [message, setMessage] = useState("");

  useEffect(() => {
    if (!open || campaigns.length > 0) return;
    fetch(`${apiBase}/api/v1/campaigns`, { cache: "no-store" })
      .then((response) => response.ok ? response.json() : [])
      .then((items) => setCampaigns(Array.isArray(items) ? items : []))
      .catch(() => setCampaigns([]));
  }, [open, campaigns.length]);

  const eligible = useMemo(() => campaigns.filter((campaign) => campaign.status === "planning" || campaign.status === "active" || campaign.status === "paused"), [campaigns]);

  async function attach() {
    if (!campaignId) return;
    setBusy(true);
    setMessage("");
    try {
      const response = await fetch(`${apiBase}/api/v1/campaigns/${encodeURIComponent(campaignId)}/items`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ entityType, entityId, channel, costUsd: Number(cost || 0) })
      });
      if (!response.ok) {
        const payload = await response.json().catch(() => ({ error: "Could not attribute activity" }));
        throw new Error(payload.error ?? "Could not attribute activity");
      }
      setMessage("Attributed.");
      router.refresh();
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "Could not attribute activity");
    } finally {
      setBusy(false);
    }
  }

  if (!open) {
    return <button className="button ghost compact" type="button" onClick={() => setOpen(true)}>{label}</button>;
  }

  return (
    <div className="campaign-attribution-control">
      <select value={campaignId} onChange={(event) => setCampaignId(event.target.value)} aria-label="Campaign">
        <option value="">Select campaign…</option>
        {eligible.map((campaign) => <option value={campaign.id} key={campaign.id}>{campaign.name} · {campaign.status}</option>)}
      </select>
      <input value={cost} onChange={(event) => setCost(event.target.value)} type="number" min="0" step="0.01" aria-label="Attributed cost in USD" title="Attributed cost USD" />
      <button className="button primary compact" type="button" disabled={!campaignId || busy} onClick={attach}>{busy ? "Adding…" : "Add"}</button>
      <button className="text-button" type="button" onClick={() => { setOpen(false); setMessage(""); }}>Cancel</button>
      {eligible.length === 0 && <span className="action-note">Create an active or planning campaign first.</span>}
      {message && <span className="action-note">{message}</span>}
    </div>
  );
}
