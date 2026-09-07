"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import type { CampaignItem } from "@/lib/campaign-shared";

const apiBase = "/api/devrelos";

export function CampaignLinkedItems({ campaignId, items }: { campaignId: string; items: CampaignItem[] }) {
  const router = useRouter();
  const [busyId, setBusyId] = useState("");
  const [message, setMessage] = useState("");

  async function remove(item: CampaignItem) {
    setBusyId(item.id);
    setMessage("");
    try {
      const response = await fetch(`${apiBase}/api/v1/campaigns/${encodeURIComponent(campaignId)}/items/${encodeURIComponent(item.id)}`, { method: "DELETE" });
      if (!response.ok) {
        const payload = await response.json().catch(() => ({ error: "Could not remove attribution" }));
        throw new Error(payload.error ?? "Could not remove attribution");
      }
      router.refresh();
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "Could not remove attribution");
    } finally {
      setBusyId("");
    }
  }

  if (items.length === 0) return <p className="campaign-linked-empty">No attributed activity yet.</p>;

  return (
    <div className="campaign-linked-list">
      {items.map((item) => (
        <div className="campaign-linked-row" key={item.id}>
          <div>
            <strong>{item.entityType.replaceAll("_", " ")}</strong>
            <span>{item.channel || "unclassified channel"} · {item.entityId.slice(0, 8)}…{item.costUsd > 0 ? ` · $${item.costUsd.toFixed(2)}` : ""}</span>
          </div>
          <button className="text-button" type="button" disabled={busyId === item.id} onClick={() => remove(item)}>{busyId === item.id ? "Removing…" : "Remove"}</button>
        </div>
      ))}
      {message && <span className="action-note">{message}</span>}
    </div>
  );
}
