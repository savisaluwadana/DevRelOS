"use client";

import { FormEvent, useState } from "react";
import { useRouter } from "next/navigation";
import type { Community, Talk } from "@/lib/api";
import type { Contact, Outreach, Relationship } from "@/lib/outreach-api";

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

function FormResult({ message }: { message: string }) {
  return message ? <span className="form-message">{message}</span> : null;
}

export function ContactForm() {
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
      await request("/api/v1/contacts", {
        method: "POST",
        body: JSON.stringify({
          name: String(data.get("name") ?? ""), role: String(data.get("role") ?? ""),
          email: String(data.get("email") ?? ""), publicProfileUrl: String(data.get("publicProfileUrl") ?? ""),
          sourceUrl: String(data.get("sourceUrl") ?? ""), doNotContact: false
        })
      });
      form.reset();
      setMessage("Contact added.");
      router.refresh();
    } catch (error) { setMessage(error instanceof Error ? error.message : "Could not add contact."); }
    finally { setBusy(false); }
  }

  return (
    <form className="operator-form" onSubmit={submit}>
      <div className="form-grid-two"><label>Name<input name="name" required /></label><label>Role<input name="role" placeholder="Organizer / community lead" /></label></div>
      <div className="form-grid-two"><label>Email<input name="email" type="email" /></label><label>Public profile<input name="publicProfileUrl" type="url" placeholder="https://…" /></label></div>
      <label>Source URL<input name="sourceUrl" type="url" placeholder="Where this contact information came from" /></label>
      <div className="form-action-row"><button className="button primary" disabled={busy}>{busy ? "Adding…" : "Add contact"}</button><FormResult message={message} /></div>
    </form>
  );
}

export function RelationshipForm({ contacts, communities }: { contacts: Contact[]; communities: Community[] }) {
  const router = useRouter();
  const [busy, setBusy] = useState(false);
  const [message, setMessage] = useState("");

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const form = event.currentTarget;
    const data = new FormData(form);
    setBusy(true); setMessage("");
    try {
      await request("/api/v1/relationships", {
        method: "POST",
        body: JSON.stringify({
          communityId: String(data.get("communityId") ?? ""), contactId: String(data.get("contactId") ?? ""),
          stage: String(data.get("stage") ?? "cold"), strength: Number(data.get("strength") ?? 0),
          nextFollowUpAt: data.get("nextFollowUpAt") ? new Date(String(data.get("nextFollowUpAt"))).toISOString() : undefined,
          notes: String(data.get("notes") ?? "")
        })
      });
      form.reset(); setMessage("Relationship created."); router.refresh();
    } catch (error) { setMessage(error instanceof Error ? error.message : "Could not create relationship."); }
    finally { setBusy(false); }
  }

  return (
    <form className="operator-form" onSubmit={submit}>
      <div className="form-grid-two">
        <label>Community<select name="communityId" defaultValue=""><option value="">None</option>{communities.map((item) => <option value={item.id} key={item.id}>{item.name}</option>)}</select></label>
        <label>Contact<select name="contactId" defaultValue=""><option value="">None</option>{contacts.filter((item) => !item.doNotContact).map((item) => <option value={item.id} key={item.id}>{item.name}</option>)}</select></label>
      </div>
      <div className="form-grid-three">
        <label>Stage<select name="stage" defaultValue="cold"><option value="cold">Cold</option><option value="warm">Warm</option><option value="engaged">Engaged</option><option value="partner">Partner</option><option value="dormant">Dormant</option></select></label>
        <label>Strength<input name="strength" type="number" min="0" max="100" defaultValue="10" /></label>
        <label>Next follow-up<input name="nextFollowUpAt" type="datetime-local" /></label>
      </div>
      <label>Notes<textarea name="notes" rows={3} placeholder="Context, previous interactions, what would make this relationship useful…" /></label>
      <div className="form-action-row"><button className="button primary" disabled={busy}>{busy ? "Saving…" : "Create relationship"}</button><FormResult message={message} /></div>
    </form>
  );
}

export function TouchpointForm({ relationships }: { relationships: Relationship[] }) {
  const router = useRouter();
  const [busy, setBusy] = useState(false); const [message, setMessage] = useState("");
  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault(); const form = event.currentTarget; const data = new FormData(form); setBusy(true); setMessage("");
    try {
      await request("/api/v1/touchpoints", { method: "POST", body: JSON.stringify({
        relationshipId: String(data.get("relationshipId") ?? ""), channel: String(data.get("channel") ?? "email"),
        direction: String(data.get("direction") ?? "outbound"), summary: String(data.get("summary") ?? ""),
        nextFollowUpAt: data.get("nextFollowUpAt") ? new Date(String(data.get("nextFollowUpAt"))).toISOString() : undefined
      }) });
      form.reset(); setMessage("Touchpoint logged."); router.refresh();
    } catch (error) { setMessage(error instanceof Error ? error.message : "Could not log touchpoint."); }
    finally { setBusy(false); }
  }
  return (
    <form className="operator-form" onSubmit={submit}>
      <label>Relationship<select name="relationshipId" required defaultValue=""><option value="" disabled>Select relationship</option>{relationships.map((item) => <option key={item.id} value={item.id}>{item.communityName || item.contactName || "Relationship"}</option>)}</select></label>
      <div className="form-grid-three"><label>Channel<select name="channel" defaultValue="email"><option value="email">Email</option><option value="linkedin">LinkedIn</option><option value="slack">Slack</option><option value="discord">Discord</option><option value="event">Event</option><option value="call">Call</option></select></label><label>Direction<select name="direction" defaultValue="outbound"><option value="outbound">Outbound</option><option value="inbound">Inbound</option><option value="internal">Internal</option></select></label><label>Next follow-up<input name="nextFollowUpAt" type="datetime-local" /></label></div>
      <label>Summary<textarea name="summary" rows={3} required placeholder="What happened and what is the next useful step?" /></label>
      <div className="form-action-row"><button className="button primary" disabled={busy}>{busy ? "Logging…" : "Log touchpoint"}</button><FormResult message={message} /></div>
    </form>
  );
}

export function OutreachDraftForm({ communities, contacts, talks }: { communities: Community[]; contacts: Contact[]; talks: Talk[] }) {
  const router = useRouter(); const [busy, setBusy] = useState(false); const [message, setMessage] = useState("");
  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault(); const form = event.currentTarget; const data = new FormData(form); setBusy(true); setMessage("");
    try {
      await request("/api/v1/outreach", { method: "POST", body: JSON.stringify({
        communityId: String(data.get("communityId") ?? ""), contactId: String(data.get("contactId") ?? ""), talkId: String(data.get("talkId") ?? ""),
        channel: String(data.get("channel") ?? "email"), subject: String(data.get("subject") ?? ""), body: String(data.get("body") ?? ""),
        rationale: String(data.get("rationale") ?? ""), status: "needs_approval"
      }) });
      form.reset(); setMessage("Draft sent to approval queue."); router.refresh();
    } catch (error) { setMessage(error instanceof Error ? error.message : "Could not create outreach draft."); }
    finally { setBusy(false); }
  }
  return (
    <form className="operator-form" onSubmit={submit}>
      <div className="form-grid-three">
        <label>Community<select name="communityId" defaultValue=""><option value="">None</option>{communities.filter((item) => item.status !== "do_not_contact").map((item) => <option key={item.id} value={item.id}>{item.name}</option>)}</select></label>
        <label>Contact<select name="contactId" defaultValue=""><option value="">None</option>{contacts.filter((item) => !item.doNotContact).map((item) => <option key={item.id} value={item.id}>{item.name}</option>)}</select></label>
        <label>Talk<select name="talkId" defaultValue=""><option value="">No talk attached</option>{talks.map((item) => <option key={item.id} value={item.id}>{item.title}</option>)}</select></label>
      </div>
      <div className="form-grid-two"><label>Channel<select name="channel" defaultValue="email"><option value="email">Email</option><option value="linkedin">LinkedIn</option><option value="slack">Slack / community</option></select></label><label>Subject<input name="subject" placeholder="Talk proposal for your community" /></label></div>
      <label>Message<textarea name="body" rows={6} required placeholder="Personalized outreach draft…" /></label>
      <label>Why this recipient / community?<textarea name="rationale" rows={2} placeholder="Fit evidence shown to the approver." /></label>
      <div className="form-action-row"><button className="button primary" disabled={busy}>{busy ? "Creating…" : "Create for approval"}</button><FormResult message={message} /></div>
    </form>
  );
}

const transitions: Record<Outreach["status"], { label: string; value: Outreach["status"] }[]> = {
  draft: [{ label: "Request approval", value: "needs_approval" }, { label: "Cancel", value: "cancelled" }],
  needs_approval: [{ label: "Approve", value: "approved" }, { label: "Return to draft", value: "draft" }, { label: "Cancel", value: "cancelled" }],
  approved: [{ label: "Queue", value: "queued" }, { label: "Cancel", value: "cancelled" }],
  queued: [{ label: "Mark sent", value: "sent" }, { label: "Mark failed", value: "failed" }, { label: "Cancel", value: "cancelled" }],
  sent: [{ label: "Mark replied", value: "replied" }], failed: [{ label: "Retry queue", value: "queued" }, { label: "Cancel", value: "cancelled" }], replied: [], cancelled: []
};

export function OutreachActions({ item }: { item: Outreach }) {
  const router = useRouter(); const [busy, setBusy] = useState(false); const [message, setMessage] = useState("");
  async function move(status: Outreach["status"]) {
    setBusy(true); setMessage("");
    try { await request(`/api/v1/outreach/${item.id}/status`, { method: "PATCH", body: JSON.stringify({ status }) }); router.refresh(); }
    catch (error) { setMessage(error instanceof Error ? error.message : "Transition failed."); }
    finally { setBusy(false); }
  }
  return <div className="outreach-actions">{transitions[item.status].map((action) => <button className={action.value === "approved" ? "button primary small-button" : "button ghost small-button"} disabled={busy} key={action.value} onClick={() => move(action.value)}>{action.label}</button>)}{message && <span className="action-note">{message}</span>}</div>;
}
