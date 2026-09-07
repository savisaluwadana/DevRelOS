"use client";

import { FormEvent, useState } from "react";
import { useRouter } from "next/navigation";
import type { FeedbackItem, FeedbackStatus } from "@/lib/feedback-api";

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

export function PainPointFeedbackAction({ painPointId }: { painPointId: string }) {
  const router = useRouter();
  const [busy, setBusy] = useState(false);
  const [message, setMessage] = useState("");

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const data = new FormData(event.currentTarget);
    setBusy(true);
    setMessage("");
    try {
      await request(`/api/v1/pain-points/${painPointId}/feedback`, {
        method: "POST",
        body: JSON.stringify({
          component: String(data.get("component") ?? "").trim(),
          owner: String(data.get("owner") ?? "").trim(),
          githubRepository: String(data.get("githubRepository") ?? "").trim()
        })
      });
      setMessage("Product feedback created.");
      router.refresh();
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "Could not create product feedback.");
    } finally {
      setBusy(false);
    }
  }

  return (
    <form className="pain-feedback-action" onSubmit={submit}>
      <span className="eyebrow">Product loop</span>
      <div className="pain-feedback-fields">
        <input name="component" placeholder="Component (optional)" />
        <input name="githubRepository" placeholder="owner/repo (optional)" />
        <input name="owner" placeholder="Owner (optional)" />
      </div>
      <button className="button ghost small-button" disabled={busy}>{busy ? "Creating…" : "Create product feedback"}</button>
      {message && <span className="action-note">{message} {message.includes("created") && <a href="/feedback">Open feedback →</a>}</span>}
    </form>
  );
}

export function ManualFeedbackForm() {
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
      await request("/api/v1/feedback", {
        method: "POST",
        body: JSON.stringify({
          sourceType: "manual",
          title: String(data.get("title") ?? "").trim(),
          summary: String(data.get("summary") ?? "").trim(),
          persona: String(data.get("persona") ?? "").trim(),
          component: String(data.get("component") ?? "").trim(),
          impactScore: Number(data.get("impactScore") ?? 50),
          frequencyScore: Number(data.get("frequencyScore") ?? 50),
          status: "new",
          owner: String(data.get("owner") ?? "").trim(),
          githubRepository: String(data.get("githubRepository") ?? "").trim(),
          metadata: { source: "manual" }
        })
      });
      form.reset();
      setMessage("Feedback captured.");
      router.refresh();
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "Could not capture feedback.");
    } finally {
      setBusy(false);
    }
  }

  return (
    <form className="operator-form feedback-form" onSubmit={submit}>
      <label>Title<input name="title" required placeholder="Developer problem or product gap" /></label>
      <label>Summary<textarea name="summary" rows={4} placeholder="What is happening, who is affected, and why it matters?" /></label>
      <div className="form-grid-three">
        <label>Persona<input name="persona" placeholder="Platform engineers" /></label>
        <label>Component<input name="component" placeholder="deployments" /></label>
        <label>Owner<input name="owner" placeholder="Team or person" /></label>
      </div>
      <div className="form-grid-three">
        <label>Impact<input name="impactScore" type="number" min="0" max="100" defaultValue="50" /></label>
        <label>Frequency<input name="frequencyScore" type="number" min="0" max="100" defaultValue="50" /></label>
        <label>GitHub repository<input name="githubRepository" placeholder="owner/repo" /></label>
      </div>
      <div className="form-action-row"><button className="button primary" disabled={busy}>{busy ? "Capturing…" : "Capture feedback"}</button>{message && <span className="form-message">{message}</span>}</div>
    </form>
  );
}

const transitions: Record<FeedbackStatus, { label: string; value: FeedbackStatus }[]> = {
  new: [{ label: "Triage", value: "triaged" }, { label: "Close", value: "closed" }, { label: "Won't fix", value: "wont_fix" }],
  triaged: [{ label: "Plan", value: "planned" }, { label: "Close", value: "closed" }, { label: "Won't fix", value: "wont_fix" }],
  planned: [{ label: "Start", value: "in_progress" }, { label: "Back to triage", value: "triaged" }, { label: "Won't fix", value: "wont_fix" }],
  in_progress: [{ label: "Back to planned", value: "planned" }, { label: "Won't fix", value: "wont_fix" }],
  shipped: [{ label: "Close loop", value: "closed" }],
  closed: [{ label: "Reopen", value: "triaged" }],
  wont_fix: [{ label: "Reopen", value: "triaged" }]
};

export function FeedbackEditor({ item }: { item: FeedbackItem }) {
  const router = useRouter();
  const [editing, setEditing] = useState(false);
  const [busy, setBusy] = useState(false);
  const [message, setMessage] = useState("");

  async function patch(payload: Record<string, unknown>) {
    setBusy(true);
    setMessage("");
    try {
      await request(`/api/v1/feedback/${item.id}`, { method: "PATCH", body: JSON.stringify(payload) });
      router.refresh();
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "Could not update feedback.");
    } finally {
      setBusy(false);
    }
  }

  async function save(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const data = new FormData(event.currentTarget);
    await patch({
      title: String(data.get("title") ?? ""),
      summary: String(data.get("summary") ?? ""),
      persona: String(data.get("persona") ?? ""),
      component: String(data.get("component") ?? ""),
      impactScore: Number(data.get("impactScore") ?? item.impactScore),
      frequencyScore: Number(data.get("frequencyScore") ?? item.frequencyScore),
      owner: String(data.get("owner") ?? ""),
      githubRepository: String(data.get("githubRepository") ?? ""),
      githubIssueNumber: data.get("githubIssueNumber") ? Number(data.get("githubIssueNumber")) : undefined,
      githubIssueUrl: String(data.get("githubIssueUrl") ?? ""),
      githubIssueTitle: String(data.get("githubIssueTitle") ?? ""),
      githubIssueBody: String(data.get("githubIssueBody") ?? ""),
      followUpNote: String(data.get("followUpNote") ?? "")
    });
    setEditing(false);
  }

  async function ship(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const data = new FormData(event.currentTarget);
    await patch({ status: "shipped", followUpNote: String(data.get("followUpNote") ?? "").trim() });
  }

  if (editing) {
    return (
      <form className="feedback-editor" onSubmit={save}>
        <label>Title<input name="title" defaultValue={item.title} required /></label>
        <label>Summary<textarea name="summary" rows={5} defaultValue={item.summary} /></label>
        <div className="form-grid-two"><label>Persona<input name="persona" defaultValue={item.persona} /></label><label>Component<input name="component" defaultValue={item.component} /></label></div>
        <div className="form-grid-two"><label>Impact<input name="impactScore" type="number" min="0" max="100" defaultValue={item.impactScore} /></label><label>Frequency<input name="frequencyScore" type="number" min="0" max="100" defaultValue={item.frequencyScore} /></label></div>
        <label>Owner<input name="owner" defaultValue={item.owner} /></label>
        <label>GitHub repository<input name="githubRepository" defaultValue={item.githubRepository} placeholder="owner/repo" /></label>
        <div className="form-grid-two"><label>Issue number<input name="githubIssueNumber" type="number" min="1" defaultValue={item.githubIssueNumber} /></label><label>Issue URL<input name="githubIssueUrl" type="url" defaultValue={item.githubIssueUrl} /></label></div>
        <label>Issue title<input name="githubIssueTitle" defaultValue={item.githubIssueTitle} /></label>
        <label>Issue draft<textarea name="githubIssueBody" rows={12} defaultValue={item.githubIssueBody} /></label>
        <label>Shipped follow-up<textarea name="followUpNote" rows={4} defaultValue={item.followUpNote} placeholder="What changed and what should we tell the developers who raised this?" /></label>
        <div className="form-action-row"><button className="button primary" disabled={busy}>{busy ? "Saving…" : "Save"}</button><button className="button ghost" type="button" onClick={() => setEditing(false)}>Cancel</button>{message && <span className="form-message">{message}</span>}</div>
      </form>
    );
  }

  return (
    <div className="feedback-controls">
      <button className="button ghost small-button" type="button" onClick={() => setEditing(true)}>Edit / link GitHub</button>
      {transitions[item.status].map((action) => <button className={action.value === "planned" || action.value === "in_progress" ? "button primary small-button" : "button ghost small-button"} disabled={busy} type="button" key={action.value} onClick={() => patch({ status: action.value })}>{action.label}</button>)}
      {item.status === "in_progress" && <form className="feedback-ship-form" onSubmit={ship}><textarea name="followUpNote" rows={3} defaultValue={item.followUpNote} placeholder="Required: what shipped and how should DevRel follow up?" required /><button className="button primary small-button" disabled={busy}>Mark shipped</button></form>}
      {message && <span className="action-note">{message}</span>}
    </div>
  );
}
