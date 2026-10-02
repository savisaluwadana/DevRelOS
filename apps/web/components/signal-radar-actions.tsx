"use client";

import { FormEvent, useState } from "react";
import { useRouter } from "next/navigation";
import type { PainPoint, Signal } from "@/lib/api";
import { DeleteButton } from "@/components/delete-button";

const apiURL = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";

async function api(path: string, init?: RequestInit) {
  const response = await fetch(`${apiURL}${path}`, {
    ...init,
    headers: { "Content-Type": "application/json", ...(init?.headers ?? {}) }
  });
  if (!response.ok) {
    const payload = await response.json().catch(() => ({ error: "request failed" }));
    throw new Error(payload.error ?? "request failed");
  }
  return response.json().catch(() => null);
}

export function SignalCaptureForm() {
  const router = useRouter();
  const [busy, setBusy] = useState(false);
  const [message, setMessage] = useState("");

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setBusy(true);
    setMessage("");
    const form = new FormData(event.currentTarget);
    const topics = String(form.get("topics") ?? "")
      .split(",")
      .map((item) => item.trim().toLowerCase().replaceAll(" ", "-"))
      .filter(Boolean);

    try {
      await api("/api/v1/signals", {
        method: "POST",
        body: JSON.stringify({
          provider: String(form.get("provider") ?? "manual"),
          externalId: String(form.get("externalId") ?? "") || `manual-${Date.now()}`,
          canonicalUrl: String(form.get("canonicalUrl") ?? ""),
          authorHandle: String(form.get("authorHandle") ?? ""),
          title: String(form.get("title") ?? ""),
          body: String(form.get("body") ?? ""),
          topics,
          engagementScore: Number(form.get("engagementScore") ?? 0),
          relevanceScore: Number(form.get("relevanceScore") ?? 70),
          status: "new"
        })
      });
      event.currentTarget.reset();
      setMessage("Signal captured.");
      router.refresh();
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "Could not capture signal.");
    } finally {
      setBusy(false);
    }
  }

  return (
    <form className="signal-capture-form" onSubmit={submit}>
      <div className="form-grid-three">
        <label>Source
          <select name="provider" defaultValue="manual">
            <option value="manual">Manual</option>
            <option value="reddit">Reddit</option>
            <option value="bluesky">Bluesky</option>
            <option value="github">GitHub</option>
            <option value="hackernews">Hacker News</option>
            <option value="rss">RSS / Web</option>
          </select>
        </label>
        <label>Author / handle<input name="authorHandle" placeholder="@developer" /></label>
        <label>Canonical URL<input name="canonicalUrl" placeholder="https://…" type="url" /></label>
      </div>
      <label>Signal title<input name="title" placeholder="Short summary of the developer problem" /></label>
      <label>Evidence / developer comment<textarea name="body" required rows={4} placeholder="Paste or summarize the developer complaint, question or friction point." /></label>
      <div className="form-grid-three">
        <label>Topics<input name="topics" placeholder="comma-separated topics" /></label>
        <label>Relevance (0–100)<input name="relevanceScore" min="0" max="100" type="number" defaultValue="70" /></label>
        <label>Engagement score<input name="engagementScore" min="0" type="number" defaultValue="0" /></label>
      </div>
      <div className="signal-form-footer">
        <button className="button primary" disabled={busy}>{busy ? "Capturing…" : "Capture signal"}</button>
        {message && <span className="form-message">{message}</span>}
      </div>
    </form>
  );
}

export function RebuildPainPointsButton() {
  const router = useRouter();
  const [busy, setBusy] = useState(false);
  const [message, setMessage] = useState("");

  async function rebuild() {
    setBusy(true);
    setMessage("");
    try {
      const result = await api("/api/v1/pain-points/rebuild", { method: "POST" });
      setMessage(`${result.clustersCreated ?? 0} clusters rebuilt`);
      router.refresh();
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "Rebuild failed");
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="inline-action">
      <button className="button primary" disabled={busy} onClick={rebuild}>{busy ? "Clustering…" : "Rebuild pain points"}</button>
      {message && <span className="action-note">{message}</span>}
    </div>
  );
}

export function SignalStatus({ id, status }: { id: string; status: Signal["status"] }) {
  const router = useRouter();
  const [busy, setBusy] = useState(false);

  async function change(next: string) {
    setBusy(true);
    try {
      await api(`/api/v1/signals/${id}`, { method: "PATCH", body: JSON.stringify({ status: next }) });
      router.refresh();
    } finally {
      setBusy(false);
    }
  }

  return (
    <select className="status-select" disabled={busy} value={status} onChange={(event) => change(event.target.value)}>
      <option value="new">New</option>
      <option value="reviewed">Reviewed</option>
      <option value="ignored">Ignored</option>
      <option value="converted">Converted</option>
    </select>
  );
}

export function SignalRowActions({ signal }: { signal: Signal }) {
  const router = useRouter();
  const [editing, setEditing] = useState(false);
  const [busy, setBusy] = useState(false);
  const [message, setMessage] = useState("");

  async function save(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const form = new FormData(event.currentTarget);
    const topics = String(form.get("topics") ?? "")
      .split(",")
      .map((item) => item.trim().toLowerCase().replaceAll(" ", "-"))
      .filter(Boolean);
    setBusy(true);
    setMessage("");
    try {
      await api(`/api/v1/signals/${signal.id}`, {
        method: "PATCH",
        body: JSON.stringify({
          title: String(form.get("title") ?? ""),
          body: String(form.get("body") ?? ""),
          topics,
          relevanceScore: Number(form.get("relevanceScore") ?? 0)
        })
      });
      setEditing(false);
      setMessage("Saved.");
      router.refresh();
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "Could not save signal.");
    } finally {
      setBusy(false);
    }
  }

  if (editing) {
    return (
      <form className="signal-editor" onSubmit={save}>
        <label>Title<input name="title" defaultValue={signal.title} /></label>
        <label>Evidence<textarea name="body" rows={3} defaultValue={signal.body} /></label>
        <div className="form-grid-two">
          <label>Topics<input name="topics" defaultValue={signal.topics.join(", ")} /></label>
          <label>Relevance (0–100)<input name="relevanceScore" type="number" min="0" max="100" defaultValue={signal.relevanceScore ?? 0} /></label>
        </div>
        <div className="form-action-row">
          <button className="button primary small-button" disabled={busy}>{busy ? "Saving…" : "Save"}</button>
          <button className="button ghost small-button" type="button" onClick={() => setEditing(false)}>Cancel</button>
          {message && <span className="form-message">{message}</span>}
        </div>
      </form>
    );
  }

  return (
    <div className="signal-row-actions">
      <button className="button ghost small-button" type="button" onClick={() => setEditing(true)}>Edit</button>
      <DeleteButton
        url={`${apiURL}/api/v1/signals/${signal.id}`}
        confirmMessage="Delete this signal? This cannot be undone."
      />
      {message && !editing && <span className="action-note">{message}</span>}
    </div>
  );
}

export function PainPointEditor({ painPoint }: { painPoint: PainPoint }) {
  const router = useRouter();
  const [editing, setEditing] = useState(false);
  const [busy, setBusy] = useState(false);
  const [message, setMessage] = useState("");

  async function save(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const form = new FormData(event.currentTarget);
    setBusy(true);
    setMessage("");
    try {
      await api(`/api/v1/pain-points/${painPoint.id}`, {
        method: "PATCH",
        body: JSON.stringify({
          title: String(form.get("title") ?? ""),
          summary: String(form.get("summary") ?? ""),
          persona: String(form.get("persona") ?? ""),
          severity: Number(form.get("severity") ?? 0),
          status: String(form.get("status") ?? painPoint.status)
        })
      });
      setEditing(false);
      setMessage("Saved.");
      router.refresh();
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "Could not save pain point.");
    } finally {
      setBusy(false);
    }
  }

  if (editing) {
    return (
      <form className="pain-point-editor" onSubmit={save}>
        <label>Title<input name="title" defaultValue={painPoint.title} required /></label>
        <label>Summary<textarea name="summary" rows={3} defaultValue={painPoint.summary} /></label>
        <div className="form-grid-three">
          <label>Persona<input name="persona" defaultValue={painPoint.persona} /></label>
          <label>Severity (0–100)<input name="severity" type="number" min="0" max="100" defaultValue={painPoint.severity} /></label>
          <label>Status
            <select name="status" defaultValue={painPoint.status}>
              <option value="active">Active</option>
              <option value="watching">Watching</option>
              <option value="addressed">Addressed</option>
              <option value="archived">Archived</option>
            </select>
          </label>
        </div>
        <div className="form-action-row">
          <button className="button primary small-button" disabled={busy}>{busy ? "Saving…" : "Save"}</button>
          <button className="button ghost small-button" type="button" onClick={() => setEditing(false)}>Cancel</button>
          {message && <span className="form-message">{message}</span>}
        </div>
      </form>
    );
  }

  return (
    <div className="pain-point-row-actions">
      <button className="button ghost small-button" type="button" onClick={() => setEditing(true)}>Edit</button>
      <DeleteButton
        url={`${apiURL}/api/v1/pain-points/${painPoint.id}`}
        confirmMessage={`Delete pain point "${painPoint.title}"? This cannot be undone.`}
      />
      {message && !editing && <span className="action-note">{message}</span>}
    </div>
  );
}

export function EvidenceButton({ painPointId, count }: { painPointId: string; count: number }) {
  const [open, setOpen] = useState(false);
  const [loading, setLoading] = useState(false);
  const [items, setItems] = useState<Signal[]>([]);

  async function toggle() {
    if (open) {
      setOpen(false);
      return;
    }
    if (items.length === 0 && count > 0) {
      setLoading(true);
      try {
        const result = await api(`/api/v1/pain-points/${painPointId}/evidence`);
        setItems(Array.isArray(result) ? result : []);
      } finally {
        setLoading(false);
      }
    }
    setOpen(true);
  }

  return (
    <div className="evidence-control">
      <button className="text-button evidence-button" onClick={toggle}>{loading ? "Loading…" : `${count} evidence ${open ? "↑" : "↓"}`}</button>
      {open && (
        <div className="evidence-list">
          {items.length === 0 ? <span>No evidence loaded.</span> : items.map((item) => (
            <a className="evidence-item" href={item.canonicalUrl || undefined} key={item.id} target={item.canonicalUrl ? "_blank" : undefined}>
              <strong>{item.title || item.body.slice(0, 90)}</strong>
              <span>{item.provider}{item.authorHandle ? ` · ${item.authorHandle}` : ""}</span>
            </a>
          ))}
        </div>
      )}
    </div>
  );
}
