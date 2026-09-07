"use client";

import { FormEvent, useState } from "react";
import { useRouter } from "next/navigation";
import type { Membership, WorkspaceRole } from "@/lib/identity-shared";
import { roleLabels } from "@/lib/identity-shared";

const apiURL = process.env.NEXT_PUBLIC_API_URL ?? "/api/devrelos";
const roles: WorkspaceRole[] = ["owner", "admin", "editor", "viewer"];

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

export function MemberForm() {
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
      await request("/api/v1/members", {
        method: "POST",
        body: JSON.stringify({
          email: String(data.get("email") ?? "").trim(),
          displayName: String(data.get("displayName") ?? "").trim(),
          password: String(data.get("password") ?? ""),
          role: String(data.get("role") ?? "editor")
        })
      });
      form.reset();
      setMessage("Workspace member added.");
      router.refresh();
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "Could not add member.");
    } finally {
      setBusy(false);
    }
  }

  return (
    <form className="operator-form identity-form" onSubmit={submit}>
      <div className="form-grid-two"><label>Email<input name="email" type="email" required /></label><label>Display name<input name="displayName" /></label></div>
      <div className="form-grid-two"><label>Initial password<input name="password" type="password" minLength={12} placeholder="Required for a new user" /></label><label>Role<select name="role" defaultValue="editor">{roles.map((role) => <option value={role} key={role}>{roleLabels[role]}</option>)}</select></label></div>
      <div className="form-action-row"><button className="button primary" disabled={busy}>{busy ? "Adding…" : "Add member"}</button>{message && <span className="form-message">{message}</span>}</div>
    </form>
  );
}

export function MemberRoleControl({ member }: { member: Membership }) {
  const router = useRouter();
  const [role, setRole] = useState<WorkspaceRole>(member.role);
  const [busy, setBusy] = useState(false);
  const [message, setMessage] = useState("");

  async function save() {
    if (role === member.role) return;
    setBusy(true);
    setMessage("");
    try {
      await request(`/api/v1/members/${member.userId}/role`, { method: "PATCH", body: JSON.stringify({ role }) });
      setMessage("Role updated.");
      router.refresh();
    } catch (error) {
      setRole(member.role);
      setMessage(error instanceof Error ? error.message : "Could not update role.");
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="identity-role-control">
      <select aria-label={`Role for ${member.email}`} value={role} onChange={(event) => setRole(event.target.value as WorkspaceRole)}>{roles.map((value) => <option value={value} key={value}>{roleLabels[value]}</option>)}</select>
      <button className="button ghost small-button" type="button" disabled={busy || role === member.role} onClick={save}>{busy ? "Saving…" : "Save role"}</button>
      {message && <span className="action-note">{message}</span>}
    </div>
  );
}
