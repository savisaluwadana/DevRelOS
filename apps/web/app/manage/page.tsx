import OperatorForms from "@/components/operator-forms";
import SubmissionStatus from "@/components/submission-status";
import { formatDate, getOperatorData, locationLabel } from "@/lib/api";

export default async function ManagePage() {
  const data = await getOperatorData();

  return (
    <div className="page-wrap manage-page">
      <header className="topbar compact-topbar">
        <div>
          <span className="eyebrow">Pipeline Workspace</span>
          <h1>Events, talks, communities and submissions.</h1>
          <p>Capture opportunities, score them, attach reusable talks and move the speaking pipeline forward.</p>
        </div>
        <a className="button ghost" href="/">← Command Center</a>
      </header>

      {!data.connected && (
        <div className="notice">
          <strong>API is not connected.</strong>
          <span>Forms will work after PostgreSQL and the Go API are running.</span>
        </div>
      )}

      <OperatorForms events={data.events} cfps={data.cfps} talks={data.talks} />

      <section className="management-grid">
        <article className="panel" id="events">
          <div className="panel-head"><div><span className="eyebrow">Events</span><h2>{data.events.length} tracked</h2></div></div>
          <div className="stack-list">
            {data.events.length === 0 ? <p className="empty-copy">No events yet.</p> : data.events.map((event) => (
              <div className="record-row" key={event.id}>
                <div><strong>{event.name}</strong><span>{locationLabel(event.city, event.country)} · {formatDate(event.startsAt)}</span></div>
                <span className="pill neutral">{event.status}</span>
              </div>
            ))}
          </div>
        </article>

        <article className="panel" id="talks">
          <div className="panel-head"><div><span className="eyebrow">Talk Library</span><h2>{data.talks.length} reusable talks</h2></div></div>
          <div className="stack-list">
            {data.talks.length === 0 ? <p className="empty-copy">No talks yet.</p> : data.talks.map((talk) => (
              <div className="record-row" key={talk.id}>
                <div><strong>{talk.title}</strong><span>{talk.level} · {talk.durationMinutes} min</span></div>
                <span className="pill neutral">{talk.status}</span>
              </div>
            ))}
          </div>
        </article>

        <article className="panel" id="communities">
          <div className="panel-head"><div><span className="eyebrow">Communities</span><h2>{data.communities.length} relationships to develop</h2></div></div>
          <div className="stack-list">
            {data.communities.length === 0 ? <p className="empty-copy">No communities yet.</p> : data.communities.map((community) => (
              <div className="record-row" key={community.id}>
                <div><strong>{community.name}</strong><span>{locationLabel(community.city, community.country)} · {community.platform}</span></div>
                <span className="score">{community.speakingFitScore ?? "—"}</span>
              </div>
            ))}
          </div>
        </article>

        <article className="panel" id="submissions">
          <div className="panel-head"><div><span className="eyebrow">Submissions</span><h2>{data.submissions.length} pipeline records</h2></div></div>
          <div className="stack-list">
            {data.submissions.length === 0 ? <p className="empty-copy">No submissions yet.</p> : data.submissions.map((submission) => (
              <div className="record-row" key={submission.id}>
                <div>
                  <strong>{submission.eventName || "Event"}</strong>
                  <span>{submission.talkTitle || "Talk"}{submission.fitScore !== undefined ? ` · fit ${submission.fitScore}` : ""}</span>
                </div>
                <SubmissionStatus id={submission.id} initialStatus={submission.status} />
              </div>
            ))}
          </div>
        </article>
      </section>

      <section className="panel cfp-panel" id="cfps">
        <div className="panel-head"><div><span className="eyebrow">CFP Pipeline</span><h2>Submission windows</h2></div></div>
        <div className="table-wrap">
          <table>
            <thead><tr><th>Event</th><th>Window</th><th>Deadline</th><th>Tracks</th><th>Fit</th><th>Status</th></tr></thead>
            <tbody>
              {data.cfps.length === 0 ? <tr><td className="empty-cell" colSpan={6}>No CFPs yet.</td></tr> : data.cfps.map((cfp) => (
                <tr key={cfp.id}>
                  <td><strong>{cfp.eventName || "Event"}</strong></td>
                  <td>{cfp.name}</td>
                  <td>{formatDate(cfp.closesAt)}</td>
                  <td><div className="tag-row">{cfp.tracks.slice(0, 3).map((track) => <span className="tag" key={track}>{track}</span>)}</div></td>
                  <td><span className="score">{cfp.fitScore ?? "—"}</span></td>
                  <td><span className="pill open">{cfp.status}</span></td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </section>
    </div>
  );
}
