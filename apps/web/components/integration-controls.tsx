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
      if (selectedProvider === "github.issues") {
        config.repository = String(data.get("repository") ?? "").trim();
        config.state = String(data.get("state") ?? "open");
        config.labels = String(data.get("labels") ?? "").trim();
        config.query = String(data.get("query") ?? "").trim();
        config.token_env = String(data.get("tokenEnv") ?? "").trim();
        config.include_pull_requests = data.get("includePullRequests") === "on";
        config.topics = topics;
      }
      if (selectedProvider === "hackernews") {
        const scanLimit = Number(data.get("scanLimit") ?? 60);
        config.query = String(data.get("query") ?? "").trim();
        config.feed = String(data.get("feed") ?? "new");
        config.scan_limit = Number.isFinite(scanLimit) ? Math.min(Math.max(scanLimit, 1), 100) : 60;
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

  const placeholder = provider === "bluesky"
    ? "Platform engineering pain points"
    : provider === "ocg"
      ? "US platform engineering communities"
      : provider === "github.issues"
        ? "OpenChoreo GitHub issues"
        : provider === "hackernews"
          ? "Hacker News platform engineering"
          : "Developer events discovery";

  return (
    <form className="operator-form integration-form" onSubmit={submit}>
      <div className="form-grid-two">
        <label>Provider
          <select name="provider" value={provider} onChange={(event) => setProvider(event.target.value)}>
            <option value="developers.events">developers.events</option>
            <option value="bluesky">Bluesky public search</option>
            <option value="ocg">CNCF / Open Community Groups</option>
            <option value="github.issues">GitHub repository issues</option>
            <option value="hackernews">Hacker News</option>
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

      {provider === "github.issues" && (
        <>
          <div className="form-grid-three">
            <label>Repository<input name="repository" placeholder="owner/repository" required /></label>
            <label>Issue state
              <select name="state" defaultValue="open">
                <option value="open">Open</option>
                <option value="closed">Closed</option>
                <option value="all">All</option>
              </select>
            </label>
            <label>Labels<input name="labels" placeholder="bug, documentation" /></label>
          </div>
          <div className="form-grid-two">
            <label>Local keyword filter<input name="query" placeholder="Optional: kubernetes deployment" /></label>
            <label>GitHub token env var<input name="tokenEnv" placeholder="GITHUB_TOKEN (optional)" /></label>
          </div>
          <div className="form-grid-two">
            <label>Topics<input name="topics" placeholder="kubernetes, devex, documentation" /></label>
            <label>Maximum issues per run<input name="pageLimit" type="number" min="1" max="100" defaultValue="50" /></label>
          </div>
          <label className="checkbox-label"><input name="includePullRequests" type="checkbox" /> Include pull requests returned by the Issues endpoint</label>
          <p className="policy-note">Tokens are never stored in connector config. Enter only the name of an environment variable available to the worker. Public repositories can run without a token at lower GitHub API limits.</p>
        </>
      )}

      {provider === "hackernews" && (
        <>
          <div className="form-grid-three">
            <label>Keyword query<input name="query" placeholder="platform engineering" required /></label>
            <label>Feed
              <select name="feed" defaultValue="new">
                <option value="new">New</option>
                <option value="top">Top</option>
                <option value="best">Best</option>
                <option value="ask">Ask HN</option>
                <option value="show">Show HN</option>
              </select>
            </label>
            <label>Stories scanned<input name="scanLimit" type="number" min="1" max="100" defaultValue="60" /></label>
          </div>
          <div className="form-grid-two">
            <label>Topics<input name="topics" placeholder="platform-engineering, kubernetes, devtools" /></label>
            <label>Maximum matches per run<input name="pageLimit" type="number" min="1" max="100" defaultValue="30" /></label>
          </div>
          <p className="policy-note">Hacker News ingestion uses the official Firebase API and performs bounded keyword filtering locally. DevRelOS stores normalized evidence and the canonical HN discussion link.</p>
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
