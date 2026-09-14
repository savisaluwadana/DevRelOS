"use client";

import { FormEvent, useState } from "react";
import { useRouter } from "next/navigation";
import type { CFP, Community, Event, Talk } from "@/lib/api";
import { DeleteButton } from "@/components/delete-button";

const apiURL = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";

async function patch(path: string, payload: unknown) {
  const response = await fetch(`${apiURL}${path}`, {
    method: "PATCH",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload)
  });
  if (!response.ok) {
    const body = await response.json().catch(() => ({}));
    throw new Error(body?.error ?? "Request failed");
  }
  return response.json().catch(() => null);
}

function tags(value: string) {
  return value.split(",").map((item) => item.trim()).filter(Boolean);
}

const eventStatuses = ["discovered", "tracking", "attending", "completed", "archived"];
const cfpStatuses = ["upcoming", "open", "closed", "cancelled"];
const talkStatuses = ["draft", "ready", "retired"];
const communityStatuses = ["discovered", "researching", "warm", "active", "paused", "do_not_contact"];

export function EventRowActions({ event }: { event: Event }) {
  const router = useRouter();
  const [editing, setEditing] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  async function save(formEvent: FormEvent<HTMLFormElement>) {
    formEvent.preventDefault();
    const data = new FormData(formEvent.currentTarget);
    setBusy(true);
    setError("");
    try {
      await patch(`/api/v1/events/${event.id}`, {
        name: String(data.get("name") ?? ""),
        description: String(data.get("description") ?? ""),
        websiteUrl: String(data.get("websiteUrl") ?? ""),
        city: String(data.get("city") ?? ""),
        country: String(data.get("country") ?? ""),
        topics: tags(String(data.get("topics") ?? "")),
        status: String(data.get("status") ?? event.status)
      });
      setEditing(false);
      router.refresh();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Could not save event.");
    } finally {
      setBusy(false);
    }
  }

  if (editing) {
    return (
      <form className="inline-edit-form" onSubmit={save}>
        <div className="form-grid-two">
          <label>Name<input name="name" defaultValue={event.name} required /></label>
          <label>Status<select name="status" defaultValue={event.status}>{eventStatuses.map((s) => <option key={s} value={s}>{s}</option>)}</select></label>
        </div>
        <label>Description<textarea name="description" rows={2} defaultValue={event.description} /></label>
        <div className="form-grid-two">
          <label>City<input name="city" defaultValue={event.city} /></label>
          <label>Country<input name="country" defaultValue={event.country} /></label>
        </div>
        <label>Website<input name="websiteUrl" type="url" defaultValue={event.websiteUrl} /></label>
        <label>Topics<input name="topics" defaultValue={event.topics.join(", ")} placeholder="comma-separated" /></label>
        <div className="form-action-row">
          <button className="button primary small-button" disabled={busy}>{busy ? "Saving…" : "Save"}</button>
          <button className="button ghost small-button" type="button" onClick={() => setEditing(false)}>Cancel</button>
          {error && <span className="form-error">{error}</span>}
        </div>
      </form>
    );
  }

  return (
    <span className="row-actions">
      <span className="pill neutral">{event.status}</span>
      <button className="button ghost small-button" type="button" onClick={() => setEditing(true)}>Edit</button>
      <DeleteButton url={`${apiURL}/api/v1/events/${event.id}`} confirmMessage={`Delete event "${event.name}"? This cannot be undone.`} />
    </span>
  );
}

export function CFPRowActions({ cfp }: { cfp: CFP }) {
  const router = useRouter();
  const [editing, setEditing] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  async function save(formEvent: FormEvent<HTMLFormElement>) {
    formEvent.preventDefault();
    const data = new FormData(formEvent.currentTarget);
    setBusy(true);
    setError("");
    try {
      await patch(`/api/v1/cfps/${cfp.id}`, {
        name: String(data.get("name") ?? ""),
        submissionUrl: String(data.get("submissionUrl") ?? ""),
        requirements: String(data.get("requirements") ?? ""),
        tracks: tags(String(data.get("tracks") ?? "")),
        status: String(data.get("status") ?? cfp.status)
      });
      setEditing(false);
      router.refresh();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Could not save CFP.");
    } finally {
      setBusy(false);
    }
  }

  if (editing) {
    return (
      <form className="inline-edit-form" onSubmit={save}>
        <div className="form-grid-two">
          <label>Name<input name="name" defaultValue={cfp.name} required /></label>
          <label>Status<select name="status" defaultValue={cfp.status}>{cfpStatuses.map((s) => <option key={s} value={s}>{s}</option>)}</select></label>
        </div>
        <label>Submission URL<input name="submissionUrl" type="url" defaultValue={cfp.submissionUrl} /></label>
        <label>Requirements<textarea name="requirements" rows={2} defaultValue={cfp.requirements} /></label>
        <label>Tracks<input name="tracks" defaultValue={cfp.tracks.join(", ")} placeholder="comma-separated" /></label>
        <div className="form-action-row">
          <button className="button primary small-button" disabled={busy}>{busy ? "Saving…" : "Save"}</button>
          <button className="button ghost small-button" type="button" onClick={() => setEditing(false)}>Cancel</button>
          {error && <span className="form-error">{error}</span>}
        </div>
      </form>
    );
  }

  return (
    <span className="row-actions">
      <span className="pill open">{cfp.status}</span>
      <button className="button ghost small-button" type="button" onClick={() => setEditing(true)}>Edit</button>
      <DeleteButton url={`${apiURL}/api/v1/cfps/${cfp.id}`} confirmMessage={`Delete CFP "${cfp.name}"? This cannot be undone.`} />
    </span>
  );
}

export function TalkRowActions({ talk }: { talk: Talk }) {
  const router = useRouter();
  const [editing, setEditing] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  async function save(formEvent: FormEvent<HTMLFormElement>) {
    formEvent.preventDefault();
    const data = new FormData(formEvent.currentTarget);
    setBusy(true);
    setError("");
    try {
      await patch(`/api/v1/talks/${talk.id}`, {
        title: String(data.get("title") ?? ""),
        abstract: String(data.get("abstract") ?? ""),
        description: String(data.get("description") ?? ""),
        level: String(data.get("level") ?? talk.level),
        durationMinutes: Number(data.get("durationMinutes") ?? talk.durationMinutes),
        topics: tags(String(data.get("topics") ?? "")),
        status: String(data.get("status") ?? talk.status)
      });
      setEditing(false);
      router.refresh();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Could not save talk.");
    } finally {
      setBusy(false);
    }
  }

  if (editing) {
    return (
      <form className="inline-edit-form" onSubmit={save}>
        <div className="form-grid-two">
          <label>Title<input name="title" defaultValue={talk.title} required /></label>
          <label>Status<select name="status" defaultValue={talk.status}>{talkStatuses.map((s) => <option key={s} value={s}>{s}</option>)}</select></label>
        </div>
        <label>Abstract<textarea name="abstract" rows={2} defaultValue={talk.abstract} /></label>
        <label>Description<textarea name="description" rows={3} defaultValue={talk.description} /></label>
        <div className="form-grid-two">
          <label>Level<input name="level" defaultValue={talk.level} /></label>
          <label>Duration (min)<input name="durationMinutes" type="number" min={5} defaultValue={talk.durationMinutes} /></label>
        </div>
        <label>Topics<input name="topics" defaultValue={talk.topics.join(", ")} placeholder="comma-separated" /></label>
        <div className="form-action-row">
          <button className="button primary small-button" disabled={busy}>{busy ? "Saving…" : "Save"}</button>
          <button className="button ghost small-button" type="button" onClick={() => setEditing(false)}>Cancel</button>
          {error && <span className="form-error">{error}</span>}
        </div>
      </form>
    );
  }

  return (
    <span className="row-actions">
      <span className="pill neutral">{talk.status}</span>
      <button className="button ghost small-button" type="button" onClick={() => setEditing(true)}>Edit</button>
      <DeleteButton url={`${apiURL}/api/v1/talks/${talk.id}`} confirmMessage={`Delete talk "${talk.title}"? This cannot be undone.`} />
    </span>
  );
}

export function CommunityRowActions({ community }: { community: Community }) {
  const router = useRouter();
  const [editing, setEditing] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  async function save(formEvent: FormEvent<HTMLFormElement>) {
    formEvent.preventDefault();
    const data = new FormData(formEvent.currentTarget);
    setBusy(true);
    setError("");
    try {
      await patch(`/api/v1/communities/${community.id}`, {
        name: String(data.get("name") ?? ""),
        platform: String(data.get("platform") ?? ""),
        websiteUrl: String(data.get("websiteUrl") ?? ""),
        city: String(data.get("city") ?? ""),
        country: String(data.get("country") ?? ""),
        topics: tags(String(data.get("topics") ?? "")),
        status: String(data.get("status") ?? community.status)
      });
      setEditing(false);
      router.refresh();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Could not save community.");
    } finally {
      setBusy(false);
    }
  }

  if (editing) {
    return (
      <form className="inline-edit-form" onSubmit={save}>
        <div className="form-grid-two">
          <label>Name<input name="name" defaultValue={community.name} required /></label>
          <label>Status<select name="status" defaultValue={community.status}>{communityStatuses.map((s) => <option key={s} value={s}>{s}</option>)}</select></label>
        </div>
        <div className="form-grid-two">
          <label>Platform<input name="platform" defaultValue={community.platform} /></label>
          <label>Website<input name="websiteUrl" type="url" defaultValue={community.websiteUrl} /></label>
        </div>
        <div className="form-grid-two">
          <label>City<input name="city" defaultValue={community.city} /></label>
          <label>Country<input name="country" defaultValue={community.country} /></label>
        </div>
        <label>Topics<input name="topics" defaultValue={community.topics.join(", ")} placeholder="comma-separated" /></label>
        <div className="form-action-row">
          <button className="button primary small-button" disabled={busy}>{busy ? "Saving…" : "Save"}</button>
          <button className="button ghost small-button" type="button" onClick={() => setEditing(false)}>Cancel</button>
          {error && <span className="form-error">{error}</span>}
        </div>
      </form>
    );
  }

  return (
    <span className="row-actions">
      <span className="score">{community.speakingFitScore ?? "—"}</span>
      <button className="button ghost small-button" type="button" onClick={() => setEditing(true)}>Edit</button>
      <DeleteButton url={`${apiURL}/api/v1/communities/${community.id}`} confirmMessage={`Delete community "${community.name}"? This cannot be undone.`} />
    </span>
  );
}
