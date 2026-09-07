import { ConnectorForm, RunConnectorButton } from "@/components/integration-controls";
import { formatDate, getIntegrationData } from "@/lib/api";

export default async function IntegrationsPage() {
  const data = await getIntegrationData();

  return (
    <div className="page-wrap manage-page">
      <header className="topbar compact-topbar">
        <div>
          <span className="eyebrow">Integrations</span>
          <h1>Source connectors with policy, provenance and run history.</h1>
          <p>Every ingestion source has explicit limits, an observable execution history and a replaceable provider implementation.</p>
        </div>
        <a className="button ghost" href="/">← Command Center</a>
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
                  {latest?.error && <p className="connector-error">{latest.error}</p>}
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
        <div className="panel-head"><div><span className="eyebrow">Provider roadmap</span><h2>Next adapters</h2></div></div>
        <div className="provider-row"><strong>CNCF / Open Community Groups</strong><span>Community directory, events and organizer relationship enrichment after the public data surface is finalized.</span></div>
        <div className="provider-row"><strong>GitHub</strong><span>Issues, discussions, releases and contribution signals.</span></div>
        <div className="provider-row"><strong>Bluesky</strong><span>Public developer conversations and topic streams.</span></div>
        <div className="provider-row"><strong>Reddit / X</strong><span>API-approved and budget-controlled integrations only.</span></div>
      </section>
    </div>
  );
}
