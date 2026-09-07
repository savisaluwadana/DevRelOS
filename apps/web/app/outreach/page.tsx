import { ContactForm, OutreachActions, OutreachDraftForm, RelationshipForm, TouchpointForm } from "@/components/outreach-actions";
import { formatDateTime } from "@/lib/api";
import { getOutreachData } from "@/lib/outreach-api";

export default async function OutreachPage() {
  const data = await getOutreachData();
  const now = Date.now();
  const followUpsDue = data.relationships.filter((item) => item.nextFollowUpAt && new Date(item.nextFollowUpAt).getTime() <= now).length;
  const approvalQueue = data.outreach.filter((item) => item.status === "needs_approval").length;
  const replies = data.outreach.filter((item) => item.status === "replied").length;

  return (
    <div className="page-wrap outreach-page">
      <header className="topbar compact-topbar">
        <div>
          <span className="eyebrow">Community Relationships</span>
          <h1>Turn relevant communities into durable speaking relationships.</h1>
          <p>Track organizers, relationship warmth, touchpoints, follow-ups and approval-gated speaking outreach without turning DevRelOS into a mass-email tool.</p>
        </div>
        <a className="button ghost" href="/manage#communities">Community pipeline →</a>
      </header>

      {!data.connected && <div className="notice"><strong>Outreach API is not connected.</strong><span>Start PostgreSQL and the Go API to use relationship workflows.</span></div>}

      <section className="metrics-grid">
        <article className="metric-card"><span className="eyebrow">Relationships</span><strong className="metric-value">{data.relationships.length}</strong><span className="muted">tracked community/contact links</span></article>
        <article className="metric-card"><span className="eyebrow">Follow-ups</span><strong className="metric-value">{followUpsDue}</strong><span className="muted">due now</span></article>
        <article className="metric-card"><span className="eyebrow">Approval Queue</span><strong className="metric-value">{approvalQueue}</strong><span className="muted">outreach drafts</span></article>
        <article className="metric-card"><span className="eyebrow">Replies</span><strong className="metric-value">{replies}</strong><span className="muted">recorded responses</span></article>
      </section>

      <section className="outreach-form-grid">
        <article className="panel"><div className="panel-head"><div><span className="eyebrow">People</span><h2>Add organizer/contact</h2></div></div><div className="outreach-form-body"><ContactForm /></div></article>
        <article className="panel"><div className="panel-head"><div><span className="eyebrow">Relationship</span><h2>Start relationship tracking</h2></div></div><div className="outreach-form-body"><RelationshipForm contacts={data.contacts} communities={data.communities} /></div></article>
        <article className="panel"><div className="panel-head"><div><span className="eyebrow">Activity</span><h2>Log touchpoint</h2></div></div><div className="outreach-form-body"><TouchpointForm relationships={data.relationships} /></div></article>
      </section>

      <section className="panel outreach-draft-panel">
        <div className="panel-head"><div><span className="eyebrow">Speaking Outreach</span><h2>Create an approval-gated pitch</h2></div><span className="muted panel-note">No autonomous send action is exposed.</span></div>
        <div className="outreach-form-body"><OutreachDraftForm communities={data.communities} contacts={data.contacts} talks={data.talks} /></div>
      </section>

      <section className="relationship-grid">
        <article className="panel span-two">
          <div className="panel-head"><div><span className="eyebrow">Relationship Pipeline</span><h2>Warmth and next follow-up</h2></div></div>
          <div className="table-wrap"><table><thead><tr><th>Community / contact</th><th>Stage</th><th>Strength</th><th>Last touch</th><th>Next follow-up</th></tr></thead><tbody>
            {data.relationships.length === 0 ? <tr><td className="empty-cell" colSpan={5}>No relationships tracked yet.</td></tr> : data.relationships.map((item) => (
              <tr key={item.id}><td><strong>{item.communityName || item.contactName || "Relationship"}</strong>{item.communityName && item.contactName && <span className="subline">{item.contactName}</span>}</td><td><span className="pill neutral">{item.stage}</span></td><td><span className="score">{item.strength}</span></td><td>{formatDateTime(item.lastTouchAt)}</td><td className={item.nextFollowUpAt && new Date(item.nextFollowUpAt).getTime() <= now ? "due-cell" : ""}>{formatDateTime(item.nextFollowUpAt)}</td></tr>
            ))}
          </tbody></table></div>
        </article>

        <article className="panel">
          <div className="panel-head"><div><span className="eyebrow">Recent Activity</span><h2>Touchpoints</h2></div></div>
          <div className="touchpoint-list">
            {data.touchpoints.length === 0 ? <p className="empty-copy">No touchpoints yet.</p> : data.touchpoints.slice(0, 12).map((item) => (
              <div className="touchpoint-item" key={item.id}><div><span className="provider-chip">{item.channel}</span><span className="direction-label">{item.direction}</span></div><strong>{item.summary}</strong><span>{formatDateTime(item.occurredAt)}</span></div>
            ))}
          </div>
        </article>
      </section>

      <section className="panel outreach-queue-panel">
        <div className="panel-head"><div><span className="eyebrow">Approval & Delivery State</span><h2>Outreach queue</h2></div><span className="muted panel-note">Sending is not automated in this tranche.</span></div>
        <div className="outreach-card-list">
          {data.outreach.length === 0 ? <p className="empty-copy">No outreach drafts yet.</p> : data.outreach.map((item) => (
            <article className="outreach-card" key={item.id}>
              <div className="outreach-card-head"><div><span className="eyebrow">{item.channel}</span><h3>{item.subject || "Untitled outreach"}</h3></div><span className={item.status === "needs_approval" ? "pill approval-pill" : item.status === "replied" ? "pill open" : "pill neutral"}>{item.status}</span></div>
              <div className="outreach-target">{item.communityName || "No community"}{item.contactName ? ` · ${item.contactName}` : ""}{item.talkTitle ? ` · ${item.talkTitle}` : ""}</div>
              <p>{item.body}</p>
              {item.rationale && <div className="rationale"><strong>Selection rationale</strong><span>{item.rationale}</span></div>}
              <OutreachActions item={item} />
            </article>
          ))}
        </div>
      </section>
    </div>
  );
}
