"use client";

import { FormEvent, useMemo, useState } from "react";
import { useRouter } from "next/navigation";
import type { MediaAsset, MediaClip } from "@/lib/media-api";
import { DeleteButton } from "@/components/delete-button";

const apiURL = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";

async function request(path: string, init?: RequestInit) {
  const response = await fetch(`${apiURL}${path}`, {
    ...init,
    headers: { "Content-Type": "application/json", ...(init?.headers ?? {}) }
  });
  if (!response.ok) {
    const body = await response.json().catch(() => ({ error: "request failed" }));
    throw new Error(body.error ?? "request failed");
  }
  return response.json().catch(() => null);
}

export function MediaAssetForm() {
  const router = useRouter();
  const [busy, setBusy] = useState(false);
  const [message, setMessage] = useState("");
  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const form = event.currentTarget;
    const data = new FormData(form);
    setBusy(true); setMessage("");
    try {
      await request("/api/v1/media-assets", { method: "POST", body: JSON.stringify({
        title: String(data.get("title") ?? ""), mediaType: String(data.get("mediaType") ?? "video"),
        sourcePath: String(data.get("sourcePath") ?? ""), sourceUrl: String(data.get("sourceUrl") ?? "")
      }) });
      form.reset(); setMessage("Media asset added."); router.refresh();
    } catch (error) { setMessage(error instanceof Error ? error.message : "Could not add media."); }
    finally { setBusy(false); }
  }
  return <form className="operator-form" onSubmit={submit}>
    <div className="form-grid-two"><label>Title<input name="title" required placeholder="Community call recording" /></label><label>Media type<select name="mediaType" defaultValue="video"><option value="video">Video</option><option value="audio">Audio</option></select></label></div>
    <div className="form-grid-two"><label>Local source path<input name="sourcePath" placeholder="recordings/call-01.mp4" /></label><label>Source URL<input name="sourceUrl" placeholder="https://… (reference only for rendering)" /></label></div>
    <div className="form-action-row"><button className="button primary" disabled={busy}>{busy ? "Adding…" : "Add media"}</button>{message && <span className="form-message">{message}</span>}</div>
  </form>;
}

export function TranscriptForm({ asset }: { asset: MediaAsset }) {
  const router = useRouter();
  const [open, setOpen] = useState(false);
  const [busy, setBusy] = useState(false);
  const [message, setMessage] = useState("");
  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault(); const data = new FormData(event.currentTarget); setBusy(true); setMessage("");
    try {
      await request(`/api/v1/media-assets/${asset.id}`, { method: "PATCH", body: JSON.stringify({ transcriptLanguage: String(data.get("language") ?? "en"), transcriptText: String(data.get("text") ?? ""), transcriptSegments: [] }) });
      setMessage("Transcript saved."); router.refresh();
    } catch (error) { setMessage(error instanceof Error ? error.message : "Could not save transcript."); }
    finally { setBusy(false); }
  }
  if (!open) return <button className="button ghost small-button" onClick={() => setOpen(true)}>Add transcript</button>;
  return <form className="media-inline-form" onSubmit={submit}><label>Language<input name="language" defaultValue={asset.transcriptLanguage || "en"} /></label><label>Transcript<textarea name="text" rows={6} defaultValue={asset.transcriptText} required /></label><div className="form-action-row"><button className="button primary small-button" disabled={busy}>Save</button><button type="button" className="button ghost small-button" onClick={() => setOpen(false)}>Cancel</button></div>{message && <span className="form-message">{message}</span>}</form>;
}

export function MediaAssetActions({ asset }: { asset: MediaAsset }) {
  const router = useRouter();
  const [editing, setEditing] = useState(false);
  const [busy, setBusy] = useState(false);
  const [message, setMessage] = useState("");

  async function save(event: FormEvent<HTMLFormElement>) {
    event.preventDefault(); const data = new FormData(event.currentTarget); setBusy(true); setMessage("");
    try {
      await request(`/api/v1/media-assets/${asset.id}`, {
        method: "PATCH",
        body: JSON.stringify({
          title: String(data.get("title") ?? ""),
          mediaType: String(data.get("mediaType") ?? asset.mediaType),
          sourcePath: String(data.get("sourcePath") ?? ""),
          sourceUrl: String(data.get("sourceUrl") ?? "")
        })
      });
      setEditing(false); setMessage("Saved."); router.refresh();
    } catch (error) { setMessage(error instanceof Error ? error.message : "Could not save media asset."); }
    finally { setBusy(false); }
  }

  if (editing) {
    return <form className="media-inline-form" onSubmit={save}>
      <label>Title<input name="title" defaultValue={asset.title} required /></label>
      <div className="form-grid-two"><label>Media type<select name="mediaType" defaultValue={asset.mediaType}><option value="video">Video</option><option value="audio">Audio</option></select></label><label>Local source path<input name="sourcePath" defaultValue={asset.sourcePath} /></label></div>
      <label>Source URL<input name="sourceUrl" defaultValue={asset.sourceUrl} /></label>
      <div className="form-action-row"><button className="button primary small-button" disabled={busy}>{busy ? "Saving…" : "Save"}</button><button type="button" className="button ghost small-button" onClick={() => setEditing(false)}>Cancel</button>{message && <span className="form-message">{message}</span>}</div>
    </form>;
  }

  return <div className="work-actions">
    <button type="button" className="button ghost small-button" onClick={() => setEditing(true)}>Edit media</button>
    <DeleteButton
      url={`${apiURL}/api/v1/media-assets/${asset.id}`}
      confirmMessage={`Delete "${asset.title}"? This cannot be undone.`}
      onDeleted={() => router.refresh()}
    />
    {message && <span className="action-note">{message}</span>}
  </div>;
}

export function ClipForm({ asset }: { asset: MediaAsset }) {
  const router = useRouter(); const [busy, setBusy] = useState(false); const [message, setMessage] = useState("");
  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault(); const form = event.currentTarget; const data = new FormData(form); setBusy(true); setMessage("");
    try {
      await request("/api/v1/media-clips", { method: "POST", body: JSON.stringify({
        mediaAssetId: asset.id, title: String(data.get("title") ?? ""), startMs: Number(data.get("startSeconds") ?? 0) * 1000,
        endMs: Number(data.get("endSeconds") ?? 0) * 1000, aspectRatio: String(data.get("aspectRatio") ?? "9:16"),
        score: Number(data.get("score") ?? 50), captionText: String(data.get("captionText") ?? ""), rationale: String(data.get("rationale") ?? "")
      }) });
      form.reset(); setMessage("Clip candidate created."); router.refresh();
    } catch (error) { setMessage(error instanceof Error ? error.message : "Could not create clip."); }
    finally { setBusy(false); }
  }
  return <form className="media-inline-form" onSubmit={submit}><div className="form-grid-two"><label>Clip title<input name="title" required placeholder="Why platform teams get stuck" /></label><label>Aspect<select name="aspectRatio" defaultValue="9:16"><option value="9:16">9:16 Short</option><option value="1:1">1:1 Square</option><option value="16:9">16:9 Landscape</option></select></label></div><div className="form-grid-three"><label>Start seconds<input name="startSeconds" type="number" min="0" step="0.1" required /></label><label>End seconds<input name="endSeconds" type="number" min="0.1" step="0.1" required /></label><label>Score<input name="score" type="number" min="0" max="100" defaultValue="70" /></label></div><label>Caption / transcript excerpt<textarea name="captionText" rows={3} /></label><label>Why this clip matters<input name="rationale" placeholder="Strong hook + clear technical takeaway" /></label><div className="form-action-row"><button className="button primary small-button" disabled={busy}>Create candidate</button>{message && <span className="form-message">{message}</span>}</div></form>;
}

export function ClipActions({ clip }: { clip: MediaClip }) {
  const router = useRouter(); const [busy, setBusy] = useState(false); const [message, setMessage] = useState(""); const [editing, setEditing] = useState(false);
  const actions = useMemo(() => {
    if (clip.status === "candidate") return [{ label: "Approve", status: "approved" }, { label: "Reject", status: "rejected" }];
    if (clip.status === "rejected") return [{ label: "Restore", status: "candidate" }];
    return [];
  }, [clip.status]);
  async function setStatus(status: string) { setBusy(true); setMessage(""); try { await request(`/api/v1/media-clips/${clip.id}`, { method: "PATCH", body: JSON.stringify({ status }) }); router.refresh(); } catch (error) { setMessage(error instanceof Error ? error.message : "Update failed."); } finally { setBusy(false); } }
  async function render() { setBusy(true); setMessage(""); try { await request(`/api/v1/media-clips/${clip.id}/render`, { method: "POST" }); setMessage("Render queued."); router.refresh(); } catch (error) { setMessage(error instanceof Error ? error.message : "Could not queue render."); } finally { setBusy(false); } }
  async function save(event: FormEvent<HTMLFormElement>) {
    event.preventDefault(); const data = new FormData(event.currentTarget); setBusy(true); setMessage("");
    try {
      await request(`/api/v1/media-clips/${clip.id}`, {
        method: "PATCH",
        body: JSON.stringify({
          title: String(data.get("title") ?? ""),
          startMs: Number(data.get("startSeconds") ?? clip.startMs / 1000) * 1000,
          endMs: Number(data.get("endSeconds") ?? clip.endMs / 1000) * 1000,
          aspectRatio: String(data.get("aspectRatio") ?? clip.aspectRatio),
          score: Number(data.get("score") ?? clip.score),
          captionText: String(data.get("captionText") ?? ""),
          rationale: String(data.get("rationale") ?? "")
        })
      });
      setEditing(false); router.refresh();
    } catch (error) { setMessage(error instanceof Error ? error.message : "Could not save clip."); }
    finally { setBusy(false); }
  }
  if (editing) {
    return <form className="media-inline-form" onSubmit={save}>
      <div className="form-grid-two"><label>Clip title<input name="title" defaultValue={clip.title} required /></label><label>Aspect<select name="aspectRatio" defaultValue={clip.aspectRatio}><option value="9:16">9:16 Short</option><option value="1:1">1:1 Square</option><option value="16:9">16:9 Landscape</option></select></label></div>
      <div className="form-grid-three"><label>Start seconds<input name="startSeconds" type="number" min="0" step="0.1" defaultValue={clip.startMs / 1000} required /></label><label>End seconds<input name="endSeconds" type="number" min="0.1" step="0.1" defaultValue={clip.endMs / 1000} required /></label><label>Score<input name="score" type="number" min="0" max="100" defaultValue={clip.score} /></label></div>
      <label>Caption / transcript excerpt<textarea name="captionText" rows={3} defaultValue={clip.captionText} /></label>
      <label>Why this clip matters<input name="rationale" defaultValue={clip.rationale} /></label>
      <div className="form-action-row"><button className="button primary small-button" disabled={busy}>{busy ? "Saving…" : "Save"}</button><button type="button" className="button ghost small-button" onClick={() => setEditing(false)}>Cancel</button>{message && <span className="form-message">{message}</span>}</div>
    </form>;
  }
  return <div className="work-actions">
    <button type="button" className="button ghost small-button" disabled={busy} onClick={() => setEditing(true)}>Edit</button>
    {actions.map(action => <button type="button" className={action.status === "approved" ? "button primary small-button" : "button ghost small-button"} disabled={busy} key={action.status} onClick={() => setStatus(action.status)}>{action.label}</button>)}
    {clip.status === "approved" && <button type="button" className="button primary small-button" disabled={busy} onClick={render}>Queue render</button>}
    <DeleteButton
      url={`${apiURL}/api/v1/media-clips/${clip.id}`}
      confirmMessage={`Delete clip "${clip.title}"? This cannot be undone.`}
      onDeleted={() => router.refresh()}
    />
    {message && <span className="action-note">{message}</span>}
  </div>;
}
