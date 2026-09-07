"use client";

import { FormEvent, useState } from "react";
import { useRouter } from "next/navigation";

export function InviteAccept({ token }: { token: string }) {
  const router = useRouter();
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setBusy(true);
    setError("");
    const data = new FormData(event.currentTarget);
    try {
      const response = await fetch("/api/session/invite", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ token, displayName: String(data.get("displayName") ?? "") })
      });
      const body = await response.json().catch(() => ({}));
      if (!response.ok) throw new Error(body?.error ?? "Invitation could not be accepted");
      router.replace("/");
      router.refresh();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Invitation could not be accepted");
    } finally {
      setBusy(false);
    }
  }

  if (!token.startsWith("di_")) {
    return <div className="login-card"><h1>Invalid invitation</h1><p>This invitation link is missing or malformed.</p></div>;
  }

  return (
    <form className="login-card" onSubmit={submit}>
      <div><span className="eyebrow">DevRelOS invitation</span><h1>Join workspace</h1><p>Accept this one-time invitation to create or attach your account and start a secure browser session.</p></div>
      <label>Display name<input name="displayName" autoComplete="name" placeholder="Your name" /></label>
      <button type="submit" disabled={busy}>{busy ? "Joining…" : "Accept invitation"}</button>
      {error ? <div className="login-error">{error}</div> : null}
    </form>
  );
}
