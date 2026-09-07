import { formatDate, getDashboard, locationLabel } from "@/lib/api";

function Metric({ label, value, note }: { label: string; value: number | string; note: string }) {
  return (
    <article className="metric-card">
      <span className="eyebrow">{label}</span>
      <strong className="metric-value">{value}</strong>
      <span className="muted">{note}</span>
    </article>
  );
}

export default async function Home() {
  const dashboard = await getDashboard();
  const highFit = dashboard?.highFitCfps ?? [];
  const upcoming = dashboard?.upcomingEvents ?? [];
  const communities = dashboard?.communityOpportunities ?? [];

  return (
    <div className="page-wrap">
      <header className="topbar">
        <div>
          <span className="eyebrow">Command Center</span>
          <h1>Run developer relations like an operating system.</h1>
          <p>Prioritize the opportunities, relationships and deadlines that can move the program forward.</p>
        </div>
        <div className="topbar-actions">
          <button className="button ghost">⌘ K Search</button>
          <button className="button primary">+ New activity</button>
        </div>
      </header>

      {!dashboard && (
        <div className="notice">
          <strong>API is not connected yet.</strong>
          <span>Start PostgreSQL, run the migration and launch the Go API to populate this dashboard.</span>
        </div>
      )}

      <section className="metrics-grid" aria-label="Program scorecard">
        <Metric label="Open CFPs" value={dashboard?.openCfps ?? 0} note="Tracked submission windows" />
        <Metric label="Closing soon" value={dashboard?.closingSoon ?? 0} note="Deadlines in the next 7 days" />
        <Metric label="In flight" value={dashboard?.submissionsInFlight ?? 0} note="Draft through submitted" />
        <Metric label="Accepted" value={dashboard?.acceptedTalks ?? 0} note="Confirmed speaking outcomes" />
      </section>

      <section className="dashboard-grid">
        <div className="panel span-two">
          <div className="panel-head">
            <div>
              <span className="eyebrow">CFP Intelligence</span>
              <h2>High-fit opportunities</h2>
            </div>
            <button className="text-button">View pipeline →</button>
          </div>
          <div className="table-wrap">
            <table>
              <thead>
                <tr>
                  <th>Event</th>
                  <th>Deadline</th>
                  <th>Tracks</th>
                  <th>Fit</th>
                  <th>Status</th>
                </tr>
              </thead>
              <tbody>
                {highFit.length === 0 ? (
                  <tr><td colSpan={5} className="empty-cell">No high-fit CFPs yet. Add an event and CFP to start the pipeline.</td></tr>
                ) : highFit.map((cfp) => (
                  <tr key={cfp.id}>
                    <td>
                      <strong>{cfp.eventName || cfp.name}</strong>
                      <span className="subline">{cfp.name}</span>
                    </td>
                    <td>{formatDate(cfp.closesAt)}</td>
                    <td><div className="tag-row">{(cfp.tracks ?? []).slice(0, 2).map((track) => <span className="tag" key={track}>{track}</span>)}</div></td>
                    <td><span className="score">{cfp.fitScore ?? "—"}</span></td>
                    <td><span className="pill open">{cfp.status}</span></td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>

        <aside className="panel priority-panel">
          <div className="panel-head">
            <div>
              <span className="eyebrow">Today</span>
              <h2>Operator queue</h2>
            </div>
          </div>
          <div className="priority-list">
            <div className="priority-item urgent">
              <span className="priority-kicker">Deadline risk</span>
              <strong>{dashboard?.closingSoon ?? 0} CFPs closing soon</strong>
              <span>Review fit, finish abstracts and submit before deadlines.</span>
            </div>
            <div className="priority-item">
              <span className="priority-kicker">Relationship motion</span>
              <strong>{communities.length} high-fit communities</strong>
              <span>Research organizers and prepare human-reviewed speaking pitches.</span>
            </div>
            <div className="priority-item">
              <span className="priority-kicker">Pipeline health</span>
              <strong>{dashboard?.submissionsInFlight ?? 0} active submissions</strong>
              <span>Move drafts to ready, then record the actual submission outcome.</span>
            </div>
          </div>
        </aside>
      </section>

      <section className="dashboard-grid lower-grid">
        <div className="panel">
          <div className="panel-head">
            <div>
              <span className="eyebrow">Event Pipeline</span>
              <h2>Upcoming events</h2>
            </div>
          </div>
          <div className="stack-list">
            {upcoming.length === 0 ? <p className="empty-copy">No upcoming events yet.</p> : upcoming.map((event) => (
              <article className="stack-row" key={event.id}>
                <div className="date-block">
                  <strong>{event.startsAt ? new Date(event.startsAt).getUTCDate() : "—"}</strong>
                  <span>{event.startsAt ? new Date(event.startsAt).toLocaleString("en", { month: "short", timeZone: "UTC" }) : "TBD"}</span>
                </div>
                <div className="stack-main">
                  <strong>{event.name}</strong>
                  <span>{locationLabel(event.city, event.country)} · {event.eventType}</span>
                </div>
                <span className="pill neutral">{event.status}</span>
              </article>
            ))}
          </div>
        </div>

        <div className="panel span-two">
          <div className="panel-head">
            <div>
              <span className="eyebrow">Community Graph</span>
              <h2>Speaking opportunities</h2>
            </div>
            <button className="text-button">Open communities →</button>
          </div>
          <div className="community-grid">
            {communities.length === 0 ? <p className="empty-copy">No ranked communities yet. Add CNCF/OCG or other community records to begin scoring.</p> : communities.map((community) => (
              <article className="community-card" key={community.id}>
                <div className="community-card-head">
                  <span className="platform-badge">{community.platform}</span>
                  <span className="score large">{community.speakingFitScore ?? "—"}</span>
                </div>
                <strong>{community.name}</strong>
                <span>{locationLabel(community.city, community.country)}</span>
                <div className="tag-row">{(community.topics ?? []).slice(0, 3).map((topic) => <span className="tag" key={topic}>{topic}</span>)}</div>
                <div className="community-meta">
                  <span>Activity {community.activityScore ?? "—"}</span>
                  <span>Next {formatDate(community.nextEventAt)}</span>
                </div>
              </article>
            ))}
          </div>
        </div>
      </section>
    </div>
  );
}
