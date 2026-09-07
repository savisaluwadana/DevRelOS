"use client";

import type { Connector } from "@/lib/api";
import { useRouter } from "next/navigation";
import { FormEvent, useState } from "react";

const apiURL = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";

export function ConnectorForm() {
  const router = useRouter();
  const [saving, setSaving] = useState(false);
  const [message, setMessage] = useState("");
  const [provider, setProvider] = useState("developers.events");

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const form = event.currentTarget;
    const data = new FormData(form);
    const selectedProvider = String(data.get("provider") ?? "");
    const usageMode = String(data.get("usageMode") ?? "");
    const pageLimit = Number(data.get("pageLimit") ?? 100);
    const topics = String(data.get("topics") ?? "")
      .split(",")
      .map((item) => item.trim().toLowerCase().replaceAll(" ", "-"))
      .filter(Boolean);

    setSaving(true);
    setMessage("");
    try {
      const config: Record<string, unknown> = {
        page_limit: Number.isFinite(pageLimit) ? Math.min(Math.max(pageLimit, 1), 1000) : 100
      };
      if (selectedProvider === "developers.events") {
        config.usage_mode = usageMode;
      }
      if (selectedProvider === "bluesky") {
        config.query = String(data.get("query") ?? "").trim();
        config.topics = topics;
      }
      if (selectedProvider === "ocg") {
        config.community = "cncf";
        config.query = String(data.get("query") ?? "").trim();
        config.region = String(data.get("region") ?? "").trim();
        config.group_category = String(data.get("groupCategory") ?? "").trim();
        config.topics = topics;
      }

      const response = await fetch(`${apiURL}/api/v1/connectors`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          provider: selectedProvider,
          name: String(data.get("name") ?? selectedProvider),
          enabled: true,
          config,
          policy: {
            reviewed: selectedProvider !== "developers.events" || usageMode === "noncommercial",
            note: "Provider-specific runtime policy is enforced by the worker."
          }
        })
      });
      if (!response.ok) {
        const error = await response.json().catch(() => ({ error: "Unable to create connector" }));
        throw new Error(error.error ?? "Unable to create connector");
      }
      form.reset();
      setProvider("developers.events");
      setMessage("Connector created.");
      router.refresh();
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "Unable to create connector");
    } finally {
      setSaving(false);
    }
  }

  const placeholder = provider === "bluesky" ? "Platform engineering pain points" : provider === "ocg" ? "US platform engineering communities" : "Developer events discovery";

  return (
    <form className="operator-form integration-form" onSubmit={submit}>
      <div className="form-grid-two">
        <label>Provider
          <select name="provider" value={provider} onChange={(event) => setProvider(event.target.value)}>
            <option value="developers.events">developers.events</option>
            <option value="bluesky">Bluesky public search</option>
            <option value="ocg">CNCF / Open Community Groups</option>
          </select>
        </label>
        <label>Connector name<input name="name" placeholder={placeholder} required /></label>
      </div>

      {provider === "bluesky" && (
        <>
          <label>Search query<input name="query" placeholder='"platform engineering" kubernetes' required /></label>
          <div className="form-grid-two">
            <label>Topics<input name="topics" placeholder="platform-engineering, kubernetes" /></label>
            <label>Maximum posts per run<input name="pageLimit" type="number" min="1" max="100" defaultValue="50" /></label>
          </div>
          <p className="policy-note">Bluesky search uses the public AppView. DevRelOS stores normalized evidence and provenance by default rather than treating public posts as a relicensable content corpus.</p>
        </>
      )}

      {provider === "ocg" && (
        <>
          <div className="form-grid-three">
            <label>Search query<input name="query" placeholder="platform engineering" /></label>
            <label>Region<input name="region" placeholder="Optional OCG region" /></label>
            <label>Group category<input name="groupCategory" placeholder="Optional category" /></label>
          </div>
          <div className="form-grid-two">
            <label>Desired topics<input name="topics" placeholder="platform-engineering, kubernetes, devex" /></label>
            <label>Maximum groups per run<input name="pageLimit" type="number" min="1" max="100" defaultValue="50" /></label>
          </div>
          <p className="policy-note">This connector uses OCG&apos;s public JSON group-search endpoint and defaults to the CNCF community. Discovered groups are upserted into the Community pipeline with source provenance.</p>
        </>
      )}

      {provider === "developers.events" && (
        <>
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
        </>
      )}

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
