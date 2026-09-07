"use client";

import { FormEvent, useMemo, useState } from "react";
import { useRouter } from "next/navigation";
import type { ContentAsset, ContentChannel, ContentStatus } from "@/lib/content-shared";
import { contentChannelLabels } from "@/lib/content-shared";
import type { WorkItem, WorkItemKind } from "@/lib/work-shared";

const apiURL = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";

const formats: Record<ContentChannel, { value: string; label: string }[]> = {
  blog: [{ value: "article", label: "Article" }, { value: "tutorial", label: "Tutorial" }],
  linkedin: [{ value: "social_post", label: "Social post" }],
  x: [{ value: "social_post", label: "Social post" }, { value: "thread", label: "Thread" }],
  newsletter: [{ value: "newsletter", label: "Newsletter" }],
  youtube: [{ value: "video_script", label: "Video script" }, { value: "tutorial", label: "Tutorial" }],
  short_video: [{ value: "short_script", label: "Short script" }],
  docs: [{ value: "documentation", label: "Documentation" }, { value: "tutorial", label: "Tutorial" }],
  talk: [{ value: "talk_outline", label: "Talk outline" }],
  community: [{ value: "community_post", label: "Community post" }]
};

const statusTransitions: Record<ContentStatus, { label: string; value: ContentStatus }[]> = {
  brief: [{ label: "Start drafting", value: "drafting" }, { label: "Archive", value: "archived" }],
  drafting: [{ label: "Send to review", value: "review" }, { label: "Back to brief", value: "brief" }, { label: "Archive", value: "archived" }],
  review: [{ label: "Approve", value: "approved" }, { label: "Request changes", value: "drafting" }, { label: "Archive", value: "archived" }],
  approved: [{ label: "Back to review", value: "review" }, { label: "Archive", value: "archived" }],
  published: [{ label: "Archive", value: "archived" }],
  archived: [{ label: "Restore", value: "brief" }]
};

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

export function ManualContentAssetForm() {
  const router = useRouter();
  const [channel, setChannel] = useState<ContentChannel>("blog");
  const [busy, setBusy] = useState(false);
  const [message, setMessage] = useState("");
  const availableFormats = formats[channel];

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const form = event.currentTarget;
    const data = new FormData(form);
    setBusy(true);
    setMessage("");
    try {
      const topics = String(data.get("topics") ?? "").split(",").map((value) => value.trim().toLowerCase().replaceAll(" ", "-")).filter(Boolean);
      await request("/api/v1/content-assets", {
        method: "POST",
        body: JSON.stringify({
          channel,
          format: String(data.get("format") ?? availableFormats[0].value),
          title: String(data.get("title") ?? "").trim(),
          audience: String(data.get("audience") ?? "developers").trim(),
          objective: String(data.get("objective") ?? "").trim(),
          brief: String(data.get("brief") ?? "").trim(),
          status: "brief",
          topics,
          sourceUrl: String(data.get("sourceUrl") ?? "").trim(),
          metadata: { source: "manual" }
        })
      });
      form.reset();
      setChannel("blog");
      setMessage("Content asset created.");
      router.refresh();
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "Could not create content asset.");
    } finally {
      setBusy(false);
    }
  }

  return (
    <form className="operator-form content-create-form" onSubmit={submit}>
      <div className="form-grid-three">
        <label>Channel<select name="channel" value={channel} onChange={(event) => setChannel(event.target.value as ContentChannel)}>{Object.entries(contentChannelLabels).map(([value, label]) => <option value={value} key={value}>{label}</option>)}</select></label>
        <label>Format<select name="format" key={channel}>{availableFormats.map((item) => <option value={item.value} key={item.value}>{item.label}</option>)}</select></label>
        <label>Audience<input name="audience" defaultValue="developers" /></label>
      </div>
      <label>Title<input name="title" required placeholder="Developer-facing content title" /></label>
      <label>Objective<input name="objective" placeholder="What should the audience understand or do after consuming this?" /></label>
      <label>Brief<textarea name="brief" rows={5} placeholder="Problem, evidence, angle, structure and review notes…" /></label>
      <div className="form-grid-two">
        <label>Topics<input name="topics" placeholder="kubernetes, platform-engineering" /></label>
        <label>Source URL<input name="sourceUrl" type="url" placeholder="https://…" /></label>
      </div>
      <div className="form-action-row"><button className="button primary" disabled={busy}>{busy ? "Creating…" : "Create asset"}</button>{message && <span className="form-message">{message}</span>}</div>
    </form>
  );
}

export function ContentAssetEditor({ asset }: { asset: ContentAsset }) {
  const router = useRouter();
  const [busy, setBusy] = useState(false);
  const [editing, setEditing] = useState(false);
  const [message, setMessage] = useState("");

  async function save(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const data = new FormData(event.currentTarget);
    setBusy(true);
    setMessage("");
    try {
      await request(`/api/v1/content-assets/${asset.id}`, {
        method: "PATCH",
        body: JSON.stringify({
          title: String(data.get("title") ?? ""),
          audience: String(data.get("audience") ?? ""),
          objective: String(data.get("objective") ?? ""),
          brief: String(data.get("brief") ?? ""),
          draft: String(data.get("draft") ?? ""),
          sourceUrl: String(data.get("sourceUrl") ?? ""),
          publishedUrl: String(data.get("publishedUrl") ?? "")
        })
      });
      setEditing(false);
      setMessage("Saved.");
      router.refresh();
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "Could not save content asset.");
    } finally {
      setBusy(false);
    }
  }

  async function move(status: ContentStatus) {
    setBusy(true);
    setMessage("");
    try {
      await request(`/api/v1/content-assets/${asset.id}`, { method: "PATCH", body: JSON.stringify({ status }) });
      router.refresh();
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "Could not update content status.");
    } finally {
      setBusy(false);
    }
  }

  async function publish(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const data = new FormData(event.currentTarget);
    const publishedUrl = String(data.get("publishedUrl") ?? "").trim();
    setBusy(true);
    setMessage("");
    try {
      await request(`/api/v1/content-assets/${asset.id}`, {
        method: "PATCH",
        body: JSON.stringify({ status: "published", publishedUrl })
      });
      router.refresh();
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "Could not mark asset published.");
    } finally {
      setBusy(false);
    }
  }

  if (editing) {
    return (
      <form className="content-editor" onSubmit={save}>
        <div className="form-grid-two"><label>Title<input name="title" defaultValue={asset.title} required /></label><label>Audience<input name="audience" defaultValue={asset.audience} /></label></div>
        <label>Objective<textarea name="objective" rows={2} defaultValue={asset.objective} /></label>
        <label>Brief<textarea name="brief" rows={10} defaultValue={asset.brief} /></label>
        <label>Draft<textarea name="draft" rows={14} defaultValue={asset.draft} placeholder="Write or paste the working draft here…" /></label>
        <div className="form-grid-two"><label>Evidence / source URL<input name="sourceUrl" type="url" defaultValue={asset.sourceUrl} /></label><label>Published URL<input name="publishedUrl" type="url" defaultValue={asset.publishedUrl} /></label></div>
        <div className="form-action-row"><button className="button primary" disabled={busy}>{busy ? "Saving…" : "Save changes"}</button><button className="button ghost" type="button" onClick={() => setEditing(false)}>Cancel</button>{message && <span className="form-message">{message}</span>}</div>
      </form>
    );
  }

  return (
    <div className="content-controls">
      <button className="button ghost small-button" type="button" onClick={() => setEditing(true)}>Edit brief / draft</button>
      {statusTransitions[asset.status].map((action) => {
        if (action.value === "published") return null;
        return <button className={action.value === "approved" || action.value === "review" ? "button primary small-button" : "button ghost small-button"} disabled={busy} type="button" key={action.value} onClick={() => move(action.value)}>{action.label}</button>;
      })}
      {asset.status === "approved" && (
        <form className="publish-inline" onSubmit={publish}><input name="publishedUrl" type="url" placeholder="Published URL" defaultValue={asset.publishedUrl} required /><button className="button primary small-button" disabled={busy}>Mark published</button></form>
      )}
      {message && <span className="action-note">{message}</span>}
    </div>
  );
}

const workDefaults: Partial<Record<WorkItemKind, { channel: ContentChannel; format: string }>> = {
  content_brief: { channel: "blog", format: "article" },
  docs_improvement: { channel: "docs", format: "documentation" },
  talk_idea: { channel: "talk", format: "talk_outline" },
  community_research: { channel: "community", format: "community_post" }
};

export function WorkToContentAction({ item }: { item: WorkItem }) {
  const router = useRouter();
  const defaults = workDefaults[item.kind];
  const [channel, setChannel] = useState<ContentChannel>(defaults?.channel ?? "blog");
  const [format, setFormat] = useState(defaults?.format ?? "article");
  const [busy, setBusy] = useState(false);
  const [message, setMessage] = useState("");
  const availableFormats = useMemo(() => formats[channel], [channel]);

  if (!defaults) return null;

  function changeChannel(next: ContentChannel) {
    setChannel(next);
    setFormat(formats[next][0].value);
  }

  async function create() {
    setBusy(true);
    setMessage("");
    try {
      await request(`/api/v1/work-items/${item.id}/content-assets`, {
        method: "POST",
        body: JSON.stringify({ channel, format, audience: "" })
      });
      setMessage("Content brief created.");
      router.refresh();
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "Could not create content brief.");
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="work-content-action">
      <span className="eyebrow">Content handoff</span>
      <div className="content-handoff-fields">
        <select aria-label="Content channel" value={channel} onChange={(event) => changeChannel(event.target.value as ContentChannel)}>{Object.entries(contentChannelLabels).map(([value, label]) => <option value={value} key={value}>{label}</option>)}</select>
        <select aria-label="Content format" value={format} onChange={(event) => setFormat(event.target.value)}>{availableFormats.map((option) => <option value={option.value} key={option.value}>{option.label}</option>)}</select>
      </div>
      <button className="button ghost small-button" disabled={busy} type="button" onClick={create}>{busy ? "Creating…" : "Create content brief"}</button>
      {message && <span className="action-note">{message} {message.includes("created") && <a href="/content">Open Content Studio →</a>}</span>}
    </div>
  );
}
