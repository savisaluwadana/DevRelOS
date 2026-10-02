"use client";

import { FormEvent, useState } from "react";
import { useRouter } from "next/navigation";

export function SessionLogin() {
  const router = useRouter();
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setBusy(true);
    setError("");
    const data = new FormData(event.currentTarget);
    const token = String(data.get("token") ?? "").trim();
    try {
      const response = await fetch("/api/session/login", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ token })
      });
      const body = await response.json().catch(() => ({}));
      if (!response.ok) throw new Error(body?.error ?? "Sign in failed");
      router.replace("/");
      router.refresh();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Sign in failed");
    } finally {
      setBusy(false);
    }
  }

  return (
    <form className="login-card" onSubmit={submit}>
      <div>
        <span className="eyebrow">DevRelOS</span>
        <h1>Sign in</h1>
        <p>Use your DevRelOS user API key. It is stored only in an HttpOnly, same-site cookie and is never exposed to page JavaScript after sign-in.</p>
      </div>
      <label>API key<input name="token" type="password" autoComplete="off" placeholder="drk_…" required /></label>
      <button type="submit" disabled={busy}>{busy ? "Signing in…" : "Sign in"}</button>
      {error ? <div className="login-error">{error}</div> : null}
    </form>
  );
}

export function SessionLogout() {
  const router = useRouter();
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  async function logout() {
    setBusy(true);
    setError("");
    try {
      // The route clears the session cookie even when backend revocation
      // fails, so a response at all means this browser is signed out. A failed
      // request means the route never ran: the cookie survives, the redirect
      // to /login bounces straight back, and silently claiming success would
      // leave someone believing they had signed out on a shared machine.
      const response = await fetch("/api/session/logout", { method: "POST" });
      if (!response.ok) throw new Error("Sign out failed");
      router.replace("/login");
      router.refresh();
    } catch {
      setError("Sign out failed — you are still signed in.");
    } finally {
      // Without this the button stayed disabled forever after a failure.
      setBusy(false);
    }
  }

  return (
    <div className="session-logout-wrap">
      <button className="session-logout" type="button" onClick={logout} disabled={busy}>
        {busy ? "Signing out…" : "Sign out"}
      </button>
      {error ? <span className="session-logout-error">{error}</span> : null}
    </div>
  );
}
