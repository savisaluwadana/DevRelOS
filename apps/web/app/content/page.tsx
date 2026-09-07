import { CampaignAttribution } from "@/components/campaign-attribution";
import { ContentAssetEditor, ManualContentAssetForm } from "@/components/content-actions";
import { contentChannelLabels, contentStatusLabels, getContentAssets } from "@/lib/content-api";
import { formatDateTime } from "@/lib/api";

export default async function ContentPage() {
  const data = await getContentAssets();
  const drafting = data.items.filter((item) => item.status === "drafting").length;
  const review = data.items.filter((item) => item.status === "review").length;
  const approved = data.items.filter((item) => item.status === "approved").length;
  const published = data.items.filter((item) => item.status === "published").length;

  return (
    <div className="page-wrap content-page">
      <header className="topbar compact-topbar">
        <div>
          <span className="eyebrow">Content Studio</span>
          <h1>Turn developer evidence into reviewable content.</h1>
          <p>Build briefs and drafts from Action Queue provenance, keep technical review explicit, and track assets through approval and publication.</p>
        </div>
        <a className="button ghost" href="/work">Open Action Queue →</a>
      </header>

      {!data.connected && <div className="notice"><strong>Content Studio API is not connected.</strong><span>Apply the latest migrations and start the Go API to use this workspace.</span></div>}

      <section className="metrics-grid">
        <article className="metric-card"><span className="eyebrow">Assets</span><strong className="metric-value">{data.items.length}</strong><span className="muted">tracked content items</span></article>
        <article className="metric-card"><span className="eyebrow">Drafting</span><strong className="metric-value">{drafting}</strong><span className="muted">actively being written</span></article>
        <article className="metric-card"><span className="eyebrow">Review / Approved</span><strong className="metric-value">{review} / {approved}</strong><span className="muted">editorial gate health</span></article>
        <article className="metric-card"><span className="eyebrow">Published</span><strong className="metric-value">{published}</strong><span className="muted">recorded outcomes</span></article>
      </section>

      <section className="panel content-create-panel">
        <div className="panel-head"><div><span className="eyebrow">Manual asset</span><h2>Create content directly</h2></div><span className="muted panel-note">For evidence-backed work, create the asset from the Action Queue so provenance is inherited automatically.</span></div>
        <div className="content-form-body"><ManualContentAssetForm /></div>
      </section>

      <section className="panel content-pipeline-panel">
        <div className="panel-head"><div><span className="eyebrow">Editorial Pipeline</span><h2>Brief → draft → review → publish</h2></div><span className="muted panel-note">Publishing is recorded only after an approved asset has a destination URL.</span></div>
        <div className="content-list">
          {data.items.length === 0 ? <p className="empty-copy">No content assets yet. Create one manually or convert an Action Queue item.</p> : data.items.map((asset) => {
            const sourceType = typeof asset.metadata?.source_type === "string" ? String(asset.metadata.source_type) : asset.workItemId ? "work item" : "manual";
            const evidenceCount = typeof asset.metadata?.evidenceCount === "number" ? Number(asset.metadata.evidenceCount) : undefined;
            return (
              <article className="content-card" key={asset.id}>
                <div className="content-card-main">
                  <div className="content-card-head">
                    <div>
                      <div className="content-kickers"><span className="platform-badge">{contentChannelLabels[asset.channel]}</span><span className="pill neutral">{asset.format.replaceAll("_", " ")}</span><span className={`pill content-status ${asset.status}`}>{contentStatusLabels[asset.status]}</span></div>
                      <h3>{asset.title}</h3>
                    </div>
                    <div className="content-updated"><span>Updated</span><strong>{formatDateTime(asset.updatedAt)}</strong></div>
                  </div>

                  <p className="content-objective">{asset.objective || "No editorial objective recorded yet."}</p>

                  <div className="content-context-grid">
                    <div><span>Audience</span><strong>{asset.audience || "developers"}</strong></div>
                    <div><span>Source</span><strong>{sourceType.replaceAll("_", " ")}</strong></div>
                    <div><span>Evidence</span><strong>{evidenceCount ?? "—"}</strong></div>
                    <div><span>Scheduled</span><strong>{formatDateTime(asset.scheduledAt)}</strong></div>
                  </div>

                  {asset.topics.length > 0 && <div className="tag-row">{asset.topics.slice(0, 6).map((topic) => <span className="tag" key={topic}>{topic}</span>)}</div>}

                  <details className="content-brief-preview">
                    <summary>View evidence-backed brief</summary>
                    <pre>{asset.brief || "No brief yet."}</pre>
                  </details>

                  {asset.draft && <details className="content-draft-preview"><summary>View current draft</summary><pre>{asset.draft}</pre></details>}

                  <div className="content-links">
                    {asset.sourceUrl && <a href={asset.sourceUrl} target="_blank" rel="noreferrer">Source evidence ↗</a>}
                    {asset.publishedUrl && <a href={asset.publishedUrl} target="_blank" rel="noreferrer">Published asset ↗</a>}
                    {asset.workItemId && <a href="/work">Linked work item →</a>}
                  </div>
                </div>
                <aside className="content-card-side">
                  <CampaignAttribution entityType="content_asset" entityId={asset.id} channel={asset.channel} />
                  <ContentAssetEditor asset={asset} />
                </aside>
              </article>
            );
          })}
        </div>
      </section>
    </div>
  );
}
