"use client";

import { FormEvent, useState } from "react";
import { useRouter } from "next/navigation";
import type { IdentityUser, MembershipRole } from "@/lib/identity-types";

const apiBase = "/api/devrelos";

async function request(path: string, init: RequestInit) {
  const response = await fetch(`${apiBase}${path}`, {
    ...init,
    headers: { "Content-Type": "application/json", ...(init.headers ?? {}) }
  });
  const body = await response.json().catch(() => ({}));
  if (!response.ok) throw new Error(body?.error ?? "Request failed");
  return body;
}

export function UserForm() {
  const router = useRouter();
  const [error, setError] = useState("");
  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError("");
    const data = new FormData(event.currentTarget);
    try {
      await request("/api/v1/identity/users", {
        method: "POST",
        body: JSON.stringify({
          email: String(data.get("email") ?? ""),
          displayName: String(data.get("displayName") ?? ""),
          status: "active"
        })
      });
      event.currentTarget.reset();
      router.refresh();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Request failed");
    }
  }
  return (
    <form className="access-form" onSubmit={submit}>
      <input name="email" type="email" placeholder="user@example.com" required />
      <input name="displayName" placeholder="Display name" />
      <button type="submit">Create user</button>
      {error ? <span className="form-error">{error}</span> : null}
    </form>
  );
}

export function MembershipForm({ users }: { users: IdentityUser[] }) {
  const router = useRouter();
  const [error, setError] = useState("");
  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError("");
    const data = new FormData(event.currentTarget);
    try {
      await request("/api/v1/identity/memberships", {
        method: "PUT",
        body: JSON.stringify({ userId: String(data.get("userId") ?? ""), role: String(data.get("role") ?? "viewer") })
      });
      router.refresh();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Request failed");
    }
  }
  return (
    <form className="access-form" onSubmit={submit}>
      <select name="userId" required defaultValue="">
        <option value="" disabled>Select user</option>
        {users.map((user) => <option value={user.id} key={user.id}>{user.displayName || user.email}</option>)}
      </select>
      <select name="role" defaultValue="viewer">
        {(["viewer", "editor", "admin", "owner"] as MembershipRole[]).map((role) => <option key={role} value={role}>{role}</option>)}
      </select>
      <button type="submit">Save membership</button>
      {error ? <span className="form-error">{error}</span> : null}
    </form>
  );
}

export function APIKeyForm({ users }: { users: IdentityUser[] }) {
  const router = useRouter();
  const [token, setToken] = useState("");
  const [error, setError] = useState("");
  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setToken("");
    setError("");
    const data = new FormData(event.currentTarget);
    const userId = String(data.get("userId") ?? "");
    try {
      const result = await request(`/api/v1/identity/users/${userId}/api-keys`, {
        method: "POST",
        body: JSON.stringify({ name: String(data.get("name") ?? "") })
      });
      setToken(String(result.token ?? ""));
      router.refresh();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Request failed");
    }
  }
  return (
    <div>
      <form className="access-form" onSubmit={submit}>
        <select name="userId" required defaultValue="">
          <option value="" disabled>Select user</option>
          {users.map((user) => <option value={user.id} key={user.id}>{user.displayName || user.email}</option>)}
        </select>
        <input name="name" placeholder="CLI / automation" required />
        <button type="submit">Create API key</button>
        {error ? <span className="form-error">{error}</span> : null}
      </form>
      {token ? <div className="token-reveal"><strong>Copy this token now.</strong><code>{token}</code><span>It will not be shown again.</span></div> : null}
    </div>
  );
}

export function RevokeKeyButton({ userId, keyId }: { userId: string; keyId: string }) {
  const router = useRouter();
  const [busy, setBusy] = useState(false);
  async function revoke() {
    setBusy(true);
    try {
      await request(`/api/v1/identity/users/${userId}/api-keys/${keyId}/revoke`, { method: "POST", body: "{}" });
      router.refresh();
    } finally {
      setBusy(false);
    }
  }
  return <button className="danger-button" type="button" onClick={revoke} disabled={busy}>{busy ? "Revoking…" : "Revoke"}</button>;
}
