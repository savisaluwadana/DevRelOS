"use client";

import { FormEvent, useState } from "react";
import { useRouter } from "next/navigation";
import type { WorkItem, WorkItemKind, WorkItemStatus } from "@/lib/work-api";
import { workKindLabels } from "@/lib/work-api";

const apiURL = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";

async function request(path: string, init?: RequestInit) {
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

const kinds = Object.entries(workKindLabels) as [WorkItemKind, string][];

export function WorkItemForm() {
  const router = useRouter();
  const [busy, setBusy] = useState(false);
  const [message, setMessage] = useState("");

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const form = event.currentTarget;
    const data = new FormData(form);
    setBusy(true);
    setMessage("");
    try {
      const due = String(data.get("dueAt") ?? "");
      await request("/api/v1/work-items", {
        method: "POST",
        body: JSON.stringify({
          sourceType: "manual",
          kind: String(data.get("kind") ?? "content_brief"),
          title: String(data.get("title") ?? ""),
          description: String(data.get("description") ?? ""),
          priority: Number(data.get("priority") ?? 50),
          status: "backlog",
          owner: String(data.get("owner") ?? ""),
          dueAt: due ? new Date(due).toISOString() : undefined,
          metadata: {}
        })
      });
      form.reset();
      setMessage("Action added to the queue.");
      router.refresh();
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "Could not create action.");
    } finally {
      setBusy(false);
    }
  }

  return (
    <form className="operator-form" onSubmit={submit}>
      <div className="form-grid-two">
        <label>Action type<select name="kind" defaultValue="content_brief">{kinds.map(([value, label]) => <option value={value} key={value}>{label}</option>)}</select></label>
        <label>Title<input name="title" required placeholder="What needs to happen?" /></label>
      </div>
      <label>Description<textarea name="description" rows={3} placeholder="Context, intended outcome and useful evidence…" /></label>
      <div className="form-grid-three">
        <label>Priority<input name="priority" type="number" min="0" max="100" defaultValue="50" /></label>
        <label>Owner<input name="owner" placeholder="Owner or team" /></label>
        <label>Due date<input name="dueAt" type="datetime-local" /></label>
      </div>
      <div className="form-action-row"><button className="button primary" disabled={busy}>{busy ? "Adding…" : "Add action"}</button>{message && <span className="form-message">{message}</span>}</div>
    </form>
  );
}

const transitions: Record<WorkItemStatus, { label: string; value: WorkItemStatus }[]> = {
  backlog: [{ label: "Plan", value: "planned" }, { label: "Start", value: "in_progress" }, { label: "Cancel", value: "cancelled" }],
  planned: [{ label: "Start", value: "in_progress" }, { label: "Block", value: "blocked" }, { label: "Backlog", value: "backlog" }, { label: "Cancel", value: "cancelled" }],
  in_progress: [{ label: "Complete", value: "done" }, { label: "Block", value: "blocked" }, { label: "Plan", value: "planned" }],
  blocked: [{ label: "Resume", value: "in_progress" }, { label: "Plan", value: "planned" }, { label: "Cancel", value: "cancelled" }],
  done: [{ label: "Reopen", value: "planned" }],
  cancelled: [{ label: "Restore", value: "backlog" }]
};

export function WorkItemActions({ item }: { item: WorkItem }) {
  const router = useRouter();
  const [busy, setBusy] = useState(false);
  const [message, setMessage] = useState("");

  async function move(status: WorkItemStatus) {
    setBusy(true);
    setMessage("");
    try {
      await request(`/api/v1/work-items/${item.id}/status`, { method: "PATCH", body: JSON.stringify({ status }) });
      router.refresh();
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "Could not update action.");
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="work-actions">
      {transitions[item.status].map((action) => <button type="button" className={action.value === "done" || action.value === "in_progress" ? "button primary small-button" : "button ghost small-button"} disabled={busy} key={action.value} onClick={() => move(action.value)}>{action.label}</button>)}
      {message && <span className="action-note">{message}</span>}
    </div>
  );
}

const painPointKinds: WorkItemKind[] = ["content_brief", "docs_improvement", "product_feedback", "talk_idea", "community_research", "engineering_task"];

export function PainPointWorkActions({ painPointId }: { painPointId: string }) {
  const router = useRouter();
  const [busyKind, setBusyKind] = useState<WorkItemKind | null>(null);
  const [message, setMessage] = useState("");

  async function convert(kind: WorkItemKind) {
    setBusyKind(kind);
    setMessage("");
    try {
      await request(`/api/v1/pain-points/${painPointId}/work-items`, {
        method: "POST",
        body: JSON.stringify({ kind })
      });
      setMessage(`${workKindLabels[kind]} created.`);
      router.refresh();
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "Could not create action.");
    } finally {
      setBusyKind(null);
    }
  }

  return (
    <div className="pain-work-actions">
      <span className="eyebrow">Turn into action</span>
      <div className="pain-action-buttons">{painPointKinds.map((kind) => <button type="button" className="mini-action" disabled={busyKind !== null} key={kind} onClick={() => convert(kind)}>{busyKind === kind ? "Creating…" : workKindLabels[kind]}</button>)}</div>
      {message && <span className="action-note">{message} <a href="/work">Open queue →</a></span>}
    </div>
  );
}
