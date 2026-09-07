"use client";

import { FormEvent, useState } from "react";
import { useRouter } from "next/navigation";
import type { Connector } from "@/lib/api";
import type { ConnectorSecret, IdentityUser, MembershipRole } from "@/lib/identity-types";

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
          status: "active",
          role: String(data.get("role") ?? "viewer")
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
      <select name="role" defaultValue="viewer">
        {(["viewer", "editor", "admin", "owner"] as MembershipRole[]).map((role) => <option key={role} value={role}>{role}</option>)}
      </select>
      <button type="submit">Provision user</button>
      {error ? <span className="form-error">{error}</span> : null}
    </form>
  );
}

export function InvitationForm() {
  const router = useRouter();
  const [inviteURL, setInviteURL] = useState("");
  const [error, setError] = useState("");
  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setInviteURL("");
    setError("");
    const data = new FormData(event.currentTarget);
    try {
      const result = await request("/api/v1/identity/invitations", {
        method: "POST",
        body: JSON.stringify({
          email: String(data.get("email") ?? ""),
          role: String(data.get("role") ?? "viewer"),
          expiresHours: Number(data.get("expiresHours") ?? 168)
        })
      });
      const token = String(result.inviteToken ?? "");
      if (token) setInviteURL(`${window.location.origin}/invite?token=${encodeURIComponent(token)}`);
      router.refresh();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Request failed");
    }
  }
  return (
    <div>
      <form className="access-form" onSubmit={submit}>
        <input name="email" type="email" placeholder="teammate@example.com" required />
        <select name="role" defaultValue="viewer">
          {(["viewer", "editor", "admin", "owner"] as MembershipRole[]).map((role) => <option key={role} value={role}>{role}</option>)}
        </select>
        <input name="expiresHours" type="number" min={1} max={720} defaultValue={168} aria-label="Invitation expiry in hours" />
        <button type="submit">Create invitation</button>
        {error ? <span className="form-error">{error}</span> : null}
      </form>
      {inviteURL ? <div className="token-reveal"><strong>Copy this invitation link now.</strong><code>{inviteURL}</code><span>The token is shown only at creation.</span></div> : null}
    </div>
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

export function SecretForm() {
  const router = useRouter();
  const [error, setError] = useState("");
  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError("");
    const data = new FormData(event.currentTarget);
    try {
      await request("/api/v1/secrets", {
        method: "POST",
        body: JSON.stringify({
          provider: String(data.get("provider") ?? ""),
          name: String(data.get("name") ?? ""),
          value: String(data.get("value") ?? "")
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
      <select name="provider" defaultValue="github.issues" required>
        <option value="github.issues">GitHub Issues</option>
        <option value="github.discussions">GitHub Discussions</option>
        <option value="github.releases">GitHub Releases</option>
        <option value="generic">Generic connector</option>
      </select>
      <input name="name" placeholder="github-read-token" required />
      <input name="value" type="password" autoComplete="new-password" placeholder="Secret value" required />
      <button type="submit">Encrypt &amp; store</button>
      {error ? <span className="form-error">{error}</span> : null}
    </form>
  );
}

export function RotateSecretButton({ secretId }: { secretId: string }) {
  const router = useRouter();
  const [value, setValue] = useState("");
  const [error, setError] = useState("");
  async function rotate() {
    if (!value.trim()) return;
    setError("");
    try {
      await request(`/api/v1/secrets/${secretId}`, { method: "PUT", body: JSON.stringify({ value }) });
      setValue("");
      router.refresh();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Request failed");
    }
  }
  return (
    <div className="inline-secret-control">
      <input value={value} onChange={(event) => setValue(event.target.value)} type="password" autoComplete="new-password" placeholder="New secret value" />
      <button type="button" onClick={rotate} disabled={!value.trim()}>Rotate</button>
      {error ? <span className="form-error">{error}</span> : null}
    </div>
  );
}

export function DeleteSecretButton({ secretId }: { secretId: string }) {
  const router = useRouter();
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  async function remove() {
    setBusy(true);
    setError("");
    try {
      await request(`/api/v1/secrets/${secretId}`, { method: "DELETE" });
      router.refresh();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Request failed");
    } finally {
      setBusy(false);
    }
  }
  return <div><button className="danger-button" type="button" onClick={remove} disabled={busy}>{busy ? "Deleting…" : "Delete"}</button>{error ? <span className="form-error">{error}</span> : null}</div>;
}

export function ConnectorSecretControl({ connector, secrets }: { connector: Connector; secrets: ConnectorSecret[] }) {
  const router = useRouter();
  const [secretId, setSecretId] = useState(connector.secretId ?? "");
  const [busy, setBusy] = useState(false);
  const compatible = secrets.filter((secret) => secret.provider === connector.provider || secret.provider === "generic");
  async function save() {
    setBusy(true);
    try {
      await request(`/api/v1/connectors/${connector.id}/secret`, { method: "PUT", body: JSON.stringify({ secretId }) });
      router.refresh();
    } finally {
      setBusy(false);
    }
  }
  return (
    <div className="inline-secret-control">
      <select value={secretId} onChange={(event) => setSecretId(event.target.value)}>
        <option value="">No encrypted secret</option>
        {compatible.map((secret) => <option key={secret.id} value={secret.id}>{secret.name} · v{secret.keyVersion}</option>)}
      </select>
      <button type="button" onClick={save} disabled={busy}>{busy ? "Saving…" : "Attach"}</button>
    </div>
  );
}

export function RevokeInvitationButton({ invitationId }: { invitationId: string }) {
  const router = useRouter();
  const [busy, setBusy] = useState(false);
  async function revoke() {
    setBusy(true);
    try {
      await request(`/api/v1/identity/invitations/${invitationId}`, { method: "DELETE" });
      router.refresh();
    } finally {
      setBusy(false);
    }
  }
  return <button className="danger-button" type="button" onClick={revoke} disabled={busy}>{busy ? "Revoking…" : "Revoke"}</button>;
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
