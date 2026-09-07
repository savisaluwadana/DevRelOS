import { CampaignAttribution } from "@/components/campaign-attribution";
import { WorkToContentAction } from "@/components/content-actions";
import { WorkItemActions, WorkItemForm } from "@/components/work-actions";
import { formatDateTime } from "@/lib/api";
import { getWorkItems, workKindLabels } from "@/lib/work-api";

function priorityClass(priority: number) {
  if (priority >= 80) return "work-priority critical";
  if (priority >= 60) return "work-priority high";
  return "work-priority";
}

export default async function WorkPage() {
  const data = await getWorkItems();
  const active = data.items.filter((item) => item.status === "in_progress").length;
  const blocked = data.items.filter((item) => item.status === "blocked").length;
  const evidenceBacked = data.items.filter((item) => item.sourceType === "pain_point" || item.sourceType === "signal").length;
  const completed = data.items.filter((item) => item.status === "done").length;

  return (
    <div className="page-wrap work-page">
      <header className="topbar compact-topbar">
        <div>
          <span className="eyebrow">DevRel Action Queue</span>
          <h1>Move developer evidence into execution.</h1>
          <p>Plan content, documentation, product feedback, talks, community research and engineering follow-ups in one traceable queue.</p>
        </div>
        <div className="topbar-actions"><a className="button ghost" href="/content">Content Studio</a><a className="button ghost" href="/signals">Find pain points →</a></div>
      </header>

      {!data.connected && <div className="notice"><strong>Action Queue API is not connected.</strong><span>Apply the latest migrations and start the Go API to use this workspace.</span></div>}

      <section className="metrics-grid">
        <article className="metric-card"><span className="eyebrow">Queued</span><strong className="metric-value">{data.items.length}</strong><span className="muted">total actions</span></article>
        <article className="metric-card"><span className="eyebrow">In Progress</span><strong className="metric-value">{active}</strong><span className="muted">being executed</span></article>
        <article className="metric-card"><span className="eyebrow">Evidence-backed</span><strong className="metric-value">{evidenceBacked}</strong><span className="muted">from signals or pain points</span></article>
        <article className="metric-card"><span className="eyebrow">Blocked / Done</span><strong className="metric-value">{blocked} / {completed}</strong><span className="muted">execution health</span></article>
      </section>

      <section className="panel work-create-panel">
        <div className="panel-head"><div><span className="eyebrow">Manual Action</span><h2>Add work directly</h2></div><span className="muted panel-note">Pain-point actions are best created from Signal Radar so provenance is preserved.</span></div>
        <div className="work-form-body"><WorkItemForm /></div>
      </section>

      <section className="panel work-queue-panel">
        <div className="panel-head"><div><span className="eyebrow">Execution Pipeline</span><h2>Prioritized work</h2></div><span className="muted panel-note">In-progress and planned work is surfaced first.</span></div>
        <div className="work-list">
          {data.items.length === 0 ? <p className="empty-copy">No actions yet. Convert a Signal Radar pain point or create a manual action above.</p> : data.items.map((item) => {
            const topics = Array.isArray(item.metadata?.topics) ? (item.metadata.topics as string[]) : [];
            return (
              <article className="work-card" key={item.id}>
                <div className="work-card-main">
                  <div className="work-card-head">
                    <div><span className="eyebrow">{workKindLabels[item.kind]}</span><h3>{item.title}</h3></div>
                    <div className={priorityClass(item.priority)}><strong>{item.priority}</strong><span>priority</span></div>
                  </div>
                  {item.description && <p>{item.description}</p>}
                  {topics.length > 0 && <div className="tag-row">{topics.slice(0, 5).map((topic) => <span className="tag" key={topic}>{topic}</span>)}</div>}
                  <div className="work-meta">
                    <span className="pill neutral">{item.status.replaceAll("_", " ")}</span>
                    <span>Source: {item.sourceType.replaceAll("_", " ")}</span>
                    {item.owner && <span>Owner: {item.owner}</span>}
                    <span>Due: {formatDateTime(item.dueAt)}</span>
                  </div>
                </div>
                <aside className="work-card-side">
                  {item.sourceType === "pain_point" && item.sourceId ? <a className="button ghost" href="/signals">View source evidence →</a> : null}
                  <CampaignAttribution entityType="work_item" entityId={item.id} />
                  <WorkToContentAction item={item} />
                  <WorkItemActions item={item} />
                </aside>
              </article>
            );
          })}
        </div>
      </section>
    </div>
  );
}
