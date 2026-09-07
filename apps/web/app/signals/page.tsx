import { PainPointFeedbackAction } from "@/components/feedback-actions";
import { EvidenceButton, RebuildPainPointsButton, SignalCaptureForm, SignalStatus } from "@/components/signal-radar-actions";
import { PainPointWorkActions } from "@/components/work-actions";
import { formatDateTime, getSignalRadarData } from "@/lib/api";

export default async function SignalsPage() {
  const data = await getSignalRadarData();
  const newSignals = data.signals.filter((item) => item.status === "new").length;
  const highSeverity = data.painPoints.filter((item) => item.severity >= 70).length;
  const providers = new Set(data.signals.map((item) => item.provider).filter(Boolean)).size;

  return (
    <div className="page-wrap signals-page">
      <header className="topbar compact-topbar">
        <div>
          <span className="eyebrow">Signal Radar</span>
          <h1>Turn developer conversation into evidence-backed priorities.</h1>
          <p>Capture developer friction, cluster recurring pain points, inspect the source evidence and convert the strongest findings into trackable DevRel work or product feedback.</p>
        </div>
        <div className="form-action-row"><a className="button ghost" href="/feedback">Product Feedback</a><a className="button ghost" href="/work">Action Queue →</a><RebuildPainPointsButton /></div>
      </header>

      {!data.connected && (
        <div className="notice"><strong>Signal API is not connected.</strong><span>Run PostgreSQL migrations and start the Go API to use this workspace.</span></div>
      )}

      <section className="metrics-grid">
        <article className="metric-card"><span className="eyebrow">Signals</span><strong className="metric-value">{data.signals.length}</strong><span className="muted">evidence records</span></article>
        <article className="metric-card"><span className="eyebrow">Review Queue</span><strong className="metric-value">{newSignals}</strong><span className="muted">new signals</span></article>
        <article className="metric-card"><span className="eyebrow">Pain Points</span><strong className="metric-value">{data.painPoints.length}</strong><span className="muted">active clusters</span></article>
        <article className="metric-card"><span className="eyebrow">High Severity</span><strong className="metric-value">{highSeverity}</strong><span className="muted">across {providers} sources</span></article>
      </section>

      <section className="panel signal-capture-panel">
        <div className="panel-head"><div><span className="eyebrow">Manual Evidence</span><h2>Capture a developer signal</h2></div><span className="muted panel-note">Useful for calls, DMs, support threads and sources without a connector.</span></div>
        <div className="signal-capture-body"><SignalCaptureForm /></div>
      </section>

      <section className="pain-point-grid">
        {data.painPoints.length === 0 ? (
          <article className="panel pain-empty"><span className="eyebrow">Pain-Point Intelligence</span><h2>No clusters yet</h2><p>Capture a few developer signals, then rebuild pain points. DevRelOS will keep every cluster linked to its evidence.</p></article>
        ) : data.painPoints.map((painPoint) => (
          <article className="pain-card" key={painPoint.id}>
            <div className="pain-card-top">
              <span className="eyebrow">{painPoint.persona || "Developer"}</span>
              <span className={painPoint.severity >= 70 ? "score large danger-score" : "score large"}>{painPoint.severity}</span>
            </div>
            <h3>{painPoint.title}</h3>
            <p>{painPoint.summary}</p>
            <div className="tag-row">{painPoint.topics.slice(0, 4).map((topic) => <span className="tag" key={topic}>{topic}</span>)}</div>
            <div className="pain-meta">
              <span className={painPoint.trendScore > 0 ? "trend-up" : painPoint.trendScore < 0 ? "trend-down" : "muted"}>Trend {painPoint.trendScore > 0 ? "+" : ""}{painPoint.trendScore}</span>
              <span>Last seen {formatDateTime(painPoint.lastSeenAt)}</span>
            </div>
            <EvidenceButton painPointId={painPoint.id} count={painPoint.evidenceCount} />
            <PainPointWorkActions painPointId={painPoint.id} />
            <PainPointFeedbackAction painPointId={painPoint.id} />
          </article>
        ))}
      </section>

      <section className="panel signal-table-panel">
        <div className="panel-head"><div><span className="eyebrow">Evidence Inbox</span><h2>Developer signals</h2></div><span className="muted panel-note">Review noise before it influences clustering.</span></div>
        <div className="table-wrap">
          <table>
            <thead><tr><th>Signal</th><th>Source</th><th>Topics</th><th>Scores</th><th>Seen</th><th>Review</th></tr></thead>
            <tbody>
              {data.signals.length === 0 ? <tr><td className="empty-cell" colSpan={6}>No signals captured yet.</td></tr> : data.signals.map((signal) => (
                <tr key={signal.id}>
                  <td className="signal-copy-cell">
                    {signal.canonicalUrl ? <a href={signal.canonicalUrl} target="_blank"><strong>{signal.title || signal.body.slice(0, 110)}</strong></a> : <strong>{signal.title || signal.body.slice(0, 110)}</strong>}
                    <span className="subline">{signal.body.length > 150 ? `${signal.body.slice(0, 150)}…` : signal.body}</span>
                  </td>
                  <td><span className="provider-chip">{signal.provider}</span>{signal.authorHandle && <span className="subline">{signal.authorHandle}</span>}</td>
                  <td><div className="tag-row">{signal.topics.slice(0, 3).map((topic) => <span className="tag" key={topic}>{topic}</span>)}</div></td>
                  <td><strong>{signal.relevanceScore ?? "—"} rel.</strong><span className="subline">{signal.engagementScore} engagement</span></td>
                  <td>{formatDateTime(signal.occurredAt || signal.createdAt)}</td>
                  <td><SignalStatus id={signal.id} status={signal.status} /></td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </section>
    </div>
  );
}
