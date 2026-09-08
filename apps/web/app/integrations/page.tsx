import { ConnectorForm, ConnectorScheduleControl, RunConnectorButton } from "@/components/integration-controls";
import { formatDate, formatDateTime, getIntegrationData } from "@/lib/api";
import Link from "next/link";

export default async function IntegrationsPage() {
  const data = await getIntegrationData();

  return (
    <div className="page-wrap manage-page">
      <header className="topbar compact-topbar">
        <div>
          <span className="eyebrow">Integrations</span>
          <h1>Source connectors with policy, provenance and scheduling.</h1>
          <p>Every ingestion source has explicit limits, an observable execution history, optional recurring runs and a replaceable provider implementation.</p>
        </div>
        <Link className="button ghost" href="/">← Command Center</Link>
      </header>

      {!data.connected && (
        <div className="notice"><strong>API is not connected.</strong><span>Start PostgreSQL and the Go API before configuring integrations.</span></div>
      )}

      <section className="panel">
        <div className="panel-head">
          <div><span className="eyebrow">New source</span><h2>Configure a connector</h2></div>
        </div>
        <div className="form-body"><ConnectorForm /></div>
      </section>

      <section className="panel integration-list">
        <div className="panel-head">
          <div><span className="eyebrow">Runtime</span><h2>{data.connectors.length} configured connectors</h2></div>
        </div>
        {data.connectors.length === 0 ? (
          <p className="empty-copy">No connectors yet. Add the first source above.</p>
        ) : (
          <div className="connector-grid">
            {data.connectors.map((connector) => {
              const runs = data.runs.get(connector.id) ?? [];
              const latest = runs[0];
              return (
                <article className="connector-card" key={connector.id}>
                  <div className="connector-card-head">
                    <div>
                      <span className="platform-badge">{connector.provider}</span>
                      <h3>{connector.name}</h3>
                    </div>
                    <span className={connector.enabled ? "pill open" : "pill neutral"}>{connector.enabled ? "enabled" : "disabled"}</span>
                  </div>
                  <div className="connector-meta-grid">
                    <div><span>Last run</span><strong>{latest ? formatDate(latest.createdAt) : "Never"}</strong></div>
                    <div><span>Status</span><strong>{latest?.status ?? "—"}</strong></div>
                    <div><span>Fetched</span><strong>{latest?.itemsFetched ?? 0}</strong></div>
                    <div><span>Cost</span><strong>${(latest?.providerCostUsd ?? 0).toFixed(4)}</strong></div>
                  </div>
                  <div className="connector-meta-grid">
                    <div><span>Cadence</span><strong>{connector.scheduleMinutes ? `${connector.scheduleMinutes} min` : "Manual"}</strong></div>
                    <div><span>Next run</span><strong>{connector.nextRunAt ? formatDateTime(connector.nextRunAt) : "—"}</strong></div>
                  </div>
                  {latest?.error && <p className="connector-error">{latest.error}</p>}
                  <ConnectorScheduleControl connector={connector} />
                  <div className="connector-actions">
                    <RunConnectorButton connector={connector} />
                    <span>{runs.length} recorded run{runs.length === 1 ? "" : "s"}</span>
                  </div>
                </article>
              );
            })}
          </div>
        )}
      </section>

      <section className="panel provider-roadmap">
        <div className="panel-head"><div><span className="eyebrow">Source catalog</span><h2>Available and planned adapters</h2></div></div>
        <div className="provider-row"><strong>GitHub Issues · available</strong><span>Repository issue pain points, labels, reactions and discussion volume with optional token-env authentication.</span></div>
        <div className="provider-row"><strong>GitHub Discussions · available</strong><span>Authenticated GraphQL monitoring of repository Q&amp;A, categories, upvotes and conversation volume.</span></div>
        <div className="provider-row"><strong>GitHub Releases · available</strong><span>Release notes, tags and asset-download signal with optional private-repository authentication.</span></div>
        <div className="provider-row"><strong>Hacker News · available</strong><span>Bounded keyword monitoring across new, top, best, Ask HN and Show HN feeds.</span></div>
        <div className="provider-row"><strong>RSS / Atom · available</strong><span>HTTPS blog, release and community feeds through an SSRF-safe fetch path.</span></div>
        <div className="provider-row"><strong>CNCF / Open Community Groups · available</strong><span>Public community directory discovery; organizer-detail enrichment remains a separate stage.</span></div>
        <div className="provider-row"><strong>Bluesky · available</strong><span>Public developer conversations and topic search.</span></div>
        <div className="provider-row"><strong>Reddit / X · planned</strong><span>API-approved and budget-controlled integrations only.</span></div>
      </section>
    </div>
  );
}
