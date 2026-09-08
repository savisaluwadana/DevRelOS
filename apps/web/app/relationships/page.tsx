import { getRelationshipRadar } from "@/lib/campaign-api";
import { renderTimestamp } from "@/lib/clock";

function date(value?: string | null) {
  if (!value) return "Not scheduled";
  return new Intl.DateTimeFormat("en", { dateStyle: "medium" }).format(new Date(value));
}

export default async function RelationshipsPage() {
  const data = await getRelationshipRadar();
  const critical = data.items.filter((item) => item.health === "critical").length;
  const watch = data.items.filter((item) => item.health === "watch").length;
  const partners = data.items.filter((item) => item.stage === "partner").length;
  const now = renderTimestamp();
  const overdue = data.items.filter((item) => item.nextFollowUpAt && new Date(item.nextFollowUpAt).getTime() < now).length;

  return (
    <div className="page-wrap relationships-page">
      <header className="topbar compact-topbar">
        <div>
          <span className="eyebrow">Relationship Radar</span>
          <h1>Know who needs attention before the relationship goes cold.</h1>
          <p>Prioritize community organizers, partners and developer contacts using relationship strength, recency and follow-up commitments.</p>
        </div>
        <a className="button ghost" href="/campaigns">Campaign attribution →</a>
      </header>

      {!data.connected && <div className="notice"><strong>Relationship API is unavailable.</strong><span>Start the Go API to calculate relationship health.</span></div>}

      <section className="metrics-grid">
        <article className="metric-card"><span className="eyebrow">Relationships</span><strong className="metric-value">{data.items.length}</strong><span className="muted">tracked contacts and communities</span></article>
        <article className="metric-card"><span className="eyebrow">Critical</span><strong className="metric-value">{critical}</strong><span className="muted">high risk of going cold</span></article>
        <article className="metric-card"><span className="eyebrow">Follow-ups overdue</span><strong className="metric-value">{overdue}</strong><span className="muted">commitments needing action</span></article>
        <article className="metric-card"><span className="eyebrow">Partners</span><strong className="metric-value">{partners}</strong><span className="muted">relationships at partner stage</span></article>
      </section>

      <section className="panel relationship-panel">
        <div className="panel-head"><div><span className="eyebrow">Priority queue</span><h2>Relationship health</h2></div><span className="muted panel-note">{watch} relationships need watching.</span></div>
        <div className="relationship-list">
          {data.items.length === 0 ? <p className="empty-copy">No relationships yet. Community/contact relationship records will appear here automatically.</p> : data.items.map((item) => (
            <article className={`relationship-card ${item.health}`} key={item.relationshipId}>
              <div className="relationship-main">
                <div className="relationship-title"><span className="eyebrow">{item.kind} · {item.stage}</span><h3>{item.name}</h3></div>
                <div className="relationship-health"><strong>{item.riskScore}</strong><span>{item.health} risk</span></div>
              </div>
              <div className="relationship-stats">
                <span>Strength <strong>{item.strength}/100</strong></span>
                <span>Last touch <strong>{item.daysSinceTouch == null ? "Never" : `${item.daysSinceTouch}d ago`}</strong></span>
                <span>Next follow-up <strong>{date(item.nextFollowUpAt)}</strong></span>
              </div>
              <div className="relationship-recommendation"><span className="eyebrow">Recommended next move</span><strong>{item.recommendedAction}</strong></div>
            </article>
          ))}
        </div>
      </section>
    </div>
  );
}
