"use client";

import { FormEvent, useState } from "react";
import { useRouter } from "next/navigation";
import type { Campaign } from "@/lib/campaign-shared";

const apiBase = "/api/devrelos";

async function request(path: string, init: RequestInit) {
  const response = await fetch(`${apiBase}${path}`, {
    ...init,
    headers: { "Content-Type": "application/json", ...(init.headers ?? {}) }
  });
  if (!response.ok) {
    const payload = await response.json().catch(() => ({ error: "Request failed" }));
    throw new Error(payload.error ?? "Request failed");
  }
  return response;
}

export function CreateCampaignForm() {
  const router = useRouter();
  const [busy, setBusy] = useState(false);
  const [message, setMessage] = useState("");

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const form = new FormData(event.currentTarget);
    setBusy(true);
    setMessage("");
    try {
      await request("/api/v1/campaigns", {
        method: "POST",
        body: JSON.stringify({
          name: String(form.get("name") ?? "").trim(),
          objective: String(form.get("objective") ?? "").trim(),
          budgetUsd: Number(form.get("budgetUsd") ?? 0),
          startsAt: form.get("startsAt") ? new Date(String(form.get("startsAt"))).toISOString() : null,
          endsAt: form.get("endsAt") ? new Date(String(form.get("endsAt"))).toISOString() : null,
          target: { primaryOutcome: String(form.get("primaryOutcome") ?? "").trim() }
        })
      });
      event.currentTarget.reset();
      setMessage("Campaign created.");
      router.refresh();
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "Could not create campaign");
    } finally {
      setBusy(false);
    }
  }

  return (
    <form className="campaign-form" onSubmit={submit}>
      <label><span>Name</span><input name="name" required placeholder="KubeCon NA developer activation" /></label>
      <label className="wide"><span>Objective</span><input name="objective" placeholder="Turn platform-engineering pain points into qualified community conversations" /></label>
      <label><span>Budget USD</span><input name="budgetUsd" type="number" min="0" step="0.01" defaultValue="0" /></label>
      <label><span>Start</span><input name="startsAt" type="date" /></label>
      <label><span>End</span><input name="endsAt" type="date" /></label>
      <label className="wide"><span>Primary outcome</span><input name="primaryOutcome" placeholder="Qualified conversations, accepted talks, shipped feedback..." /></label>
      <div className="campaign-form-actions"><button className="button primary" disabled={busy}>{busy ? "Creating…" : "Create campaign"}</button>{message && <span className="action-note">{message}</span>}</div>
    </form>
  );
}

export function CampaignControls({ campaign }: { campaign: Campaign }) {
  const router = useRouter();
  const [busy, setBusy] = useState(false);
  const [message, setMessage] = useState("");
  const transitions: Record<string, Campaign["status"][]> = {
    planning: ["active", "archived"],
    active: ["paused", "completed", "archived"],
    paused: ["active", "completed", "archived"],
    completed: ["active", "archived"],
    archived: []
  };

  async function setStatus(status: Campaign["status"]) {
    setBusy(true); setMessage("");
    try {
      await request(`/api/v1/campaigns/${campaign.id}/status`, { method: "PATCH", body: JSON.stringify({ status }) });
      router.refresh();
    } catch (error) { setMessage(error instanceof Error ? error.message : "Could not update campaign"); }
    finally { setBusy(false); }
  }

  async function metric(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const form = new FormData(event.currentTarget);
    setBusy(true); setMessage("");
    try {
      await request(`/api/v1/campaigns/${campaign.id}/metrics`, {
        method: "POST",
        body: JSON.stringify({ metricKey: String(form.get("metricKey") ?? "").trim(), metricValue: Number(form.get("metricValue") ?? 0), source: "manual" })
      });
      event.currentTarget.reset(); router.refresh();
    } catch (error) { setMessage(error instanceof Error ? error.message : "Could not record metric"); }
    finally { setBusy(false); }
  }

  async function link(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const form = new FormData(event.currentTarget);
    setBusy(true); setMessage("");
    try {
      await request(`/api/v1/campaigns/${campaign.id}/items`, {
        method: "POST",
        body: JSON.stringify({
          entityType: String(form.get("entityType") ?? ""),
          entityId: String(form.get("entityId") ?? "").trim(),
          channel: String(form.get("channel") ?? "").trim(),
          costUsd: Number(form.get("costUsd") ?? 0)
        })
      });
      event.currentTarget.reset(); router.refresh();
    } catch (error) { setMessage(error instanceof Error ? error.message : "Could not link activity"); }
    finally { setBusy(false); }
  }

  return (
    <div className="campaign-controls">
      <div className="campaign-status-actions">
        {transitions[campaign.status].map((status) => <button className="button ghost compact" type="button" disabled={busy} key={status} onClick={() => setStatus(status)}>{status}</button>)}
      </div>
      <details>
        <summary>Record outcome metric</summary>
        <form className="inline-campaign-form" onSubmit={metric}>
          <input name="metricKey" required placeholder="registrations / leads / docs_visits" />
          <input name="metricValue" required type="number" step="0.01" placeholder="Value" />
          <button className="button ghost" disabled={busy}>Record</button>
        </form>
      </details>
      <details>
        <summary>Link existing DevRel activity</summary>
        <form className="inline-campaign-form campaign-link-form" onSubmit={link}>
          <select name="entityType" defaultValue="content_asset">
            <option value="content_asset">Content</option><option value="work_item">Work item</option><option value="event">Event</option>
            <option value="cfp">CFP</option><option value="submission">Submission</option><option value="outreach">Outreach</option>
            <option value="feedback">Feedback</option><option value="media_asset">Media</option><option value="community">Community</option><option value="talk">Talk</option>
          </select>
          <input name="entityId" required placeholder="Entity UUID" />
          <input name="channel" placeholder="Channel (optional)" />
          <input name="costUsd" type="number" min="0" step="0.01" defaultValue="0" />
          <button className="button ghost" disabled={busy}>Attach</button>
        </form>
      </details>
      {message && <span className="action-note">{message}</span>}
    </div>
  );
}
