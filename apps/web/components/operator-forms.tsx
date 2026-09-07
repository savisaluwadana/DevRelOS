"use client";

import type { CFP, Event, Talk } from "@/lib/api";
import { FormEvent, useState } from "react";
import { useRouter } from "next/navigation";

const apiURL = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";

type Props = {
  events: Event[];
  cfps: CFP[];
  talks: Talk[];
};

type FormKind = "event" | "cfp" | "talk" | "community" | "submission";

function text(data: FormData, key: string) {
  return String(data.get(key) ?? "").trim();
}

function tags(value: string) {
  return value.split(",").map((item) => item.trim()).filter(Boolean);
}

function optionalNumber(value: string) {
  if (!value) return undefined;
  const parsed = Number(value);
  return Number.isFinite(parsed) ? parsed : undefined;
}

export default function OperatorForms({ events, cfps, talks }: Props) {
  const router = useRouter();
  const [active, setActive] = useState<FormKind>("event");
  const [message, setMessage] = useState("");
  const [saving, setSaving] = useState(false);

  async function submit(path: string, payload: unknown, form: HTMLFormElement) {
    setSaving(true);
    setMessage("");
    try {
      const response = await fetch(`${apiURL}${path}`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(payload)
      });
      if (!response.ok) {
        const error = await response.json().catch(() => ({ error: "Request failed" }));
        throw new Error(error.error ?? "Request failed");
      }
      form.reset();
      setMessage("Saved successfully.");
      router.refresh();
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "Request failed");
    } finally {
      setSaving(false);
    }
  }

  function onEvent(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const form = event.currentTarget;
    const data = new FormData(form);
    return submit("/api/v1/events", {
      name: text(data, "name"),
      description: text(data, "description"),
      websiteUrl: text(data, "websiteUrl"),
      city: text(data, "city"),
      country: text(data, "country"),
      timezone: text(data, "timezone"),
      startsAt: text(data, "startsAt") ? new Date(text(data, "startsAt")).toISOString() : undefined,
      endsAt: text(data, "endsAt") ? new Date(text(data, "endsAt")).toISOString() : undefined,
      eventType: text(data, "eventType") || "conference",
      topics: tags(text(data, "topics")),
      status: "tracking"
    }, form);
  }

  function onCFP(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const form = event.currentTarget;
    const data = new FormData(form);
    return submit("/api/v1/cfps", {
      eventId: text(data, "eventId"),
      name: text(data, "name") || "Main CFP",
      submissionUrl: text(data, "submissionUrl"),
      opensAt: text(data, "opensAt") ? new Date(text(data, "opensAt")).toISOString() : undefined,
      closesAt: text(data, "closesAt") ? new Date(text(data, "closesAt")).toISOString() : undefined,
      tracks: tags(text(data, "tracks")),
      requirements: text(data, "requirements"),
      status: "open",
      fitScore: optionalNumber(text(data, "fitScore")),
      scoreReason: { source: "manual" }
    }, form);
  }

  function onTalk(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const form = event.currentTarget;
    const data = new FormData(form);
    return submit("/api/v1/talks", {
      title: text(data, "title"),
      abstract: text(data, "abstract"),
      description: text(data, "description"),
      level: text(data, "level") || "intermediate",
      durationMinutes: optionalNumber(text(data, "durationMinutes")) ?? 30,
      topics: tags(text(data, "topics")),
      demoUrl: text(data, "demoUrl"),
      slidesUrl: text(data, "slidesUrl"),
      status: "ready"
    }, form);
  }

  function onCommunity(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const form = event.currentTarget;
    const data = new FormData(form);
    return submit("/api/v1/communities", {
      name: text(data, "name"),
      platform: text(data, "platform") || "manual",
      websiteUrl: text(data, "websiteUrl"),
      city: text(data, "city"),
      country: text(data, "country"),
      timezone: text(data, "timezone"),
      topics: tags(text(data, "topics")),
      memberCount: optionalNumber(text(data, "memberCount")),
      activityScore: optionalNumber(text(data, "activityScore")),
      speakingFitScore: optionalNumber(text(data, "speakingFitScore")),
      nextEventAt: text(data, "nextEventAt") ? new Date(text(data, "nextEventAt")).toISOString() : undefined,
      status: "researching"
    }, form);
  }

  function onSubmission(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const form = event.currentTarget;
    const data = new FormData(form);
    return submit("/api/v1/submissions", {
      cfpId: text(data, "cfpId"),
      talkId: text(data, "talkId"),
      notes: text(data, "notes"),
      fitScore: optionalNumber(text(data, "fitScore")),
      status: "draft"
    }, form);
  }

  return (
    <section className="panel form-panel">
      <div className="panel-head form-panel-head">
        <div>
          <span className="eyebrow">Operator actions</span>
          <h2>Create and move work into the pipeline</h2>
        </div>
        {message && <span className="form-message">{message}</span>}
      </div>

      <div className="form-tabs" role="tablist" aria-label="Create record">
        {(["event", "cfp", "talk", "community", "submission"] as FormKind[]).map((kind) => (
          <button
            className={active === kind ? "form-tab active" : "form-tab"}
            key={kind}
            onClick={() => setActive(kind)}
            type="button"
          >
            {kind}
          </button>
        ))}
      </div>

      <div className="form-body">
        {active === "event" && (
          <form className="operator-form" onSubmit={onEvent}>
            <label>Name<input name="name" required placeholder="KubeCon + CloudNativeCon Europe" /></label>
            <label>Website<input name="websiteUrl" type="url" placeholder="https://..." /></label>
            <div className="form-grid-three">
              <label>City<input name="city" /></label>
              <label>Country<input name="country" /></label>
              <label>Timezone<input name="timezone" placeholder="Europe/Amsterdam" /></label>
            </div>
            <div className="form-grid-three">
              <label>Starts<input name="startsAt" type="datetime-local" /></label>
              <label>Ends<input name="endsAt" type="datetime-local" /></label>
              <label>Type<select name="eventType" defaultValue="conference"><option>conference</option><option>meetup</option><option>webinar</option><option>workshop</option><option>hackathon</option></select></label>
            </div>
            <label>Topics<input name="topics" placeholder="kubernetes, platform engineering, backstage" /></label>
            <label>Description<textarea name="description" rows={3} /></label>
            <button className="button primary" disabled={saving}>{saving ? "Saving…" : "Create event"}</button>
          </form>
        )}

        {active === "cfp" && (
          <form className="operator-form" onSubmit={onCFP}>
            <label>Event<select name="eventId" required defaultValue=""><option value="" disabled>Select event</option>{events.map((item) => <option key={item.id} value={item.id}>{item.name}</option>)}</select></label>
            <div className="form-grid-two">
              <label>CFP name<input name="name" placeholder="Main CFP" /></label>
              <label>Fit score<input name="fitScore" type="number" min="0" max="100" placeholder="85" /></label>
            </div>
            <label>Submission URL<input name="submissionUrl" type="url" placeholder="https://sessionize.com/..." /></label>
            <div className="form-grid-two">
              <label>Opens<input name="opensAt" type="datetime-local" /></label>
              <label>Closes<input name="closesAt" type="datetime-local" /></label>
            </div>
            <label>Tracks<input name="tracks" placeholder="platform engineering, kubernetes, developer experience" /></label>
            <label>Requirements<textarea name="requirements" rows={3} /></label>
            <button className="button primary" disabled={saving || events.length === 0}>{saving ? "Saving…" : "Create CFP"}</button>
          </form>
        )}

        {active === "talk" && (
          <form className="operator-form" onSubmit={onTalk}>
            <label>Title<input name="title" required placeholder="Building a Platform API Without Building Another Platform" /></label>
            <label>Abstract<textarea name="abstract" required rows={5} /></label>
            <div className="form-grid-three">
              <label>Level<select name="level" defaultValue="intermediate"><option>beginner</option><option>intermediate</option><option>advanced</option></select></label>
              <label>Duration<input name="durationMinutes" type="number" min="5" defaultValue="30" /></label>
              <label>Demo URL<input name="demoUrl" type="url" /></label>
            </div>
            <label>Topics<input name="topics" placeholder="platform engineering, kubernetes, developer portals" /></label>
            <label>Slides URL<input name="slidesUrl" type="url" /></label>
            <label>Description<textarea name="description" rows={3} /></label>
            <button className="button primary" disabled={saving}>{saving ? "Saving…" : "Add talk"}</button>
          </form>
        )}

        {active === "community" && (
          <form className="operator-form" onSubmit={onCommunity}>
            <div className="form-grid-two">
              <label>Name<input name="name" required placeholder="Cloud Native Colombo" /></label>
              <label>Platform<input name="platform" placeholder="ocg" /></label>
            </div>
            <label>Website<input name="websiteUrl" type="url" /></label>
            <div className="form-grid-three">
              <label>City<input name="city" /></label>
              <label>Country<input name="country" /></label>
              <label>Timezone<input name="timezone" placeholder="Asia/Colombo" /></label>
            </div>
            <label>Topics<input name="topics" placeholder="kubernetes, devops, platform engineering" /></label>
            <div className="form-grid-three">
              <label>Members<input name="memberCount" type="number" min="0" /></label>
              <label>Activity score<input name="activityScore" type="number" min="0" max="100" /></label>
              <label>Speaking fit<input name="speakingFitScore" type="number" min="0" max="100" /></label>
            </div>
            <label>Next event<input name="nextEventAt" type="datetime-local" /></label>
            <button className="button primary" disabled={saving}>{saving ? "Saving…" : "Add community"}</button>
          </form>
        )}

        {active === "submission" && (
          <form className="operator-form" onSubmit={onSubmission}>
            <label>CFP<select name="cfpId" required defaultValue=""><option value="" disabled>Select CFP</option>{cfps.map((item) => <option key={item.id} value={item.id}>{item.eventName || item.name} — {item.name}</option>)}</select></label>
            <label>Talk<select name="talkId" required defaultValue=""><option value="" disabled>Select talk</option>{talks.map((item) => <option key={item.id} value={item.id}>{item.title}</option>)}</select></label>
            <label>Fit score<input name="fitScore" type="number" min="0" max="100" /></label>
            <label>Notes<textarea name="notes" rows={3} /></label>
            <button className="button primary" disabled={saving || cfps.length === 0 || talks.length === 0}>{saving ? "Saving…" : "Create submission"}</button>
          </form>
        )}
      </div>
    </section>
  );
}
