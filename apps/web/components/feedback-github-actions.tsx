"use client";

import { useRouter } from "next/navigation";
import { useState } from "react";
import type { FeedbackItem } from "@/lib/feedback-api";

const apiBase = "/api/devrelos";

async function request(path: string, init?: RequestInit) {
  const response = await fetch(`${apiBase}${path}`, {
    ...init,
    headers: { "Content-Type": "application/json", ...(init?.headers ?? {}) },
    cache: "no-store"
  });
  const payload = await response.json().catch(() => ({}));
  if (!response.ok) throw new Error(payload.error ?? "GitHub workflow request failed");
  return payload;
}

export function FeedbackGitHubActions({ item }: { item: FeedbackItem }) {
  const router = useRouter();
  const [busy, setBusy] = useState<"prefill" | "sync" | "">("");
  const [message, setMessage] = useState("");

  async function openPrefill() {
    const popup = window.open("about:blank", "_blank");
    if (popup) popup.opener = null;
    setBusy("prefill");
    setMessage("");
    try {
      const payload = await request(`/api/v1/feedback/${encodeURIComponent(item.id)}/github/prefill`);
      if (!payload.url) throw new Error("GitHub issue URL was not returned");
      if (popup) popup.location.href = String(payload.url);
      else window.location.href = String(payload.url);
      setMessage("Opened a prefilled GitHub issue draft. Link the created issue number back here after submitting it.");
    } catch (error) {
      if (popup) popup.close();
      setMessage(error instanceof Error ? error.message : "Could not prepare GitHub issue.");
    } finally {
      setBusy("");
    }
  }

  async function syncIssue() {
    setBusy("sync");
    setMessage("");
    try {
      const payload = await request(`/api/v1/feedback/${encodeURIComponent(item.id)}/github/sync`, { method: "POST", body: "{}" });
      setMessage(`Synced GitHub issue: ${payload.state ?? "updated"}.`);
      router.refresh();
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "Could not sync GitHub issue.");
    } finally {
      setBusy("");
    }
  }

  if (!item.githubRepository) return null;

  return (
    <div className="feedback-github-actions">
      {!item.githubIssueNumber ? (
        <button className="button ghost small-button" type="button" disabled={Boolean(busy)} onClick={openPrefill}>
          {busy === "prefill" ? "Preparing…" : "Open prefilled GitHub issue"}
        </button>
      ) : (
        <button className="button ghost small-button" type="button" disabled={Boolean(busy)} onClick={syncIssue}>
          {busy === "sync" ? "Syncing…" : "Sync GitHub issue"}
        </button>
      )}
      {message && <span className="action-note">{message}</span>}
    </div>
  );
}
