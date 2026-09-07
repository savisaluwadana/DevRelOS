"use client";

import type { Connector } from "@/lib/api";
import { useRouter } from "next/navigation";
import { FormEvent, useState } from "react";

const apiURL = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";

export function ConnectorForm() {
  const router = useRouter();
  const [saving, setSaving] = useState(false);
  const [message, setMessage] = useState("");

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const form = event.currentTarget;
    const data = new FormData(form);
    const provider = String(data.get("provider") ?? "");
    const usageMode = String(data.get("usageMode") ?? "");
    const pageLimit = Number(data.get("pageLimit") ?? 100);

    setSaving(true);
    setMessage("");
    try {
      const response = await fetch(`${apiURL}/api/v1/connectors`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          provider,
          name: String(data.get("name") ?? provider),
          enabled: true,
          config: {
            usage_mode: usageMode,
            page_limit: Number.isFinite(pageLimit) ? Math.min(Math.max(pageLimit, 1), 1000) : 100
          },
          policy: {
            reviewed: usageMode === "noncommercial",
            note: "Provider-specific runtime policy is enforced by the worker."
          }
        })
      });
      if (!response.ok) {
        const error = await response.json().catch(() => ({ error: "Unable to create connector" }));
        throw new Error(error.error ?? "Unable to create connector");
      }
      form.reset();
      setMessage("Connector created.");
      router.refresh();
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "Unable to create connector");
    } finally {
      setSaving(false);
    }
  }

  return (
    <form className="operator-form integration-form" onSubmit={submit}>
      <div className="form-grid-two">
        <label>Provider
          <select name="provider" defaultValue="developers.events">
            <option value="developers.events">developers.events</option>
          </select>
        </label>
        <label>Connector name<input name="name" placeholder="Developer events discovery" required /></label>
      </div>
      <div className="form-grid-two">
        <label>Usage mode
          <select name="usageMode" defaultValue="">
            <option value="">Blocked by default</option>
            <option value="noncommercial">Reviewed non-commercial use</option>
          </select>
        </label>
        <label>Maximum records per run<input name="pageLimit" type="number" min="1" max="1000" defaultValue="100" /></label>
      </div>
      <p className="policy-note">developers.events content/data is not treated as commercially redistributable. The worker will refuse this connector unless its use has been reviewed and explicitly marked non-commercial.</p>
      <div className="form-action-row">
        <button className="button primary" disabled={saving}>{saving ? "Saving…" : "Add connector"}</button>
        {message && <span className="form-message">{message}</span>}
      </div>
    </form>
  );
}

export function RunConnectorButton({ connector }: { connector: Connector }) {
  const router = useRouter();
  const [queued, setQueued] = useState(false);

  async function queue() {
    setQueued(true);
    try {
      const response = await fetch(`${apiURL}/api/v1/connectors/${connector.id}/runs`, { method: "POST" });
      if (!response.ok) throw new Error("Unable to queue connector");
      router.refresh();
    } finally {
      setQueued(false);
    }
  }

  return (
    <button className="button ghost run-button" disabled={queued || !connector.enabled} onClick={queue} type="button">
      {queued ? "Queued…" : "Run now"}
    </button>
  );
}
