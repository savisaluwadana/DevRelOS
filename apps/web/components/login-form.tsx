"use client";

import { FormEvent, useState } from "react";
import { useRouter } from "next/navigation";

const apiURL = process.env.NEXT_PUBLIC_API_URL ?? "/api/devrelos";

export function LoginForm() {
  const router = useRouter();
  const [busy, setBusy] = useState(false);
  const [message, setMessage] = useState("");

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const data = new FormData(event.currentTarget);
    setBusy(true);
    setMessage("");
    try {
      const response = await fetch(`${apiURL}/api/v1/auth/login`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          email: String(data.get("email") ?? "").trim(),
          password: String(data.get("password") ?? "")
        })
      });
      if (!response.ok) {
        const payload = await response.json().catch(() => ({ error: "Sign in failed" }));
        throw new Error(payload.error ?? "Sign in failed");
      }
      router.push("/");
      router.refresh();
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "Sign in failed");
    } finally {
      setBusy(false);
    }
  }

  return (
    <form className="identity-card identity-form" onSubmit={submit}>
      <label>Email<input type="email" name="email" autoComplete="email" required /></label>
      <label>Password<input type="password" name="password" autoComplete="current-password" required /></label>
      <button className="button primary" disabled={busy}>{busy ? "Signing in…" : "Sign in"}</button>
      {message && <span className="form-message">{message}</span>}
    </form>
  );
}

export function LogoutButton() {
  const router = useRouter();
  const [busy, setBusy] = useState(false);

  async function logout() {
    setBusy(true);
    try {
      await fetch(`${apiURL}/api/v1/auth/logout`, { method: "POST" });
    } finally {
      router.push("/login");
      router.refresh();
      setBusy(false);
    }
  }

  return <button className="button ghost" type="button" onClick={logout} disabled={busy}>{busy ? "Signing out…" : "Sign out"}</button>;
}
