import { formatDate, locationLabel } from "@/lib/api";
import { getSpeakingOpportunities } from "@/lib/opportunity-api";

function scoreClass(score: number) {
  if (score >= 75) return "opportunity-score high";
  if (score >= 50) return "opportunity-score medium";
  return "opportunity-score low";
}

export default async function OpportunitiesPage() {
  const data = await getSpeakingOpportunities();
  const highFit = data.items.filter((item) => item.score >= 70).length;
  const warm = data.items.filter((item) => item.relationship && item.relationship.stage !== "cold").length;
  const withUpcomingEvents = data.items.filter((item) => item.community.nextEventAt).length;

  return (
    <div className="page-wrap opportunities-page">
      <header className="topbar compact-topbar">
        <div>
          <span className="eyebrow">Speaking Opportunity Intelligence</span>
          <h1>Match the right talk to the right developer community.</h1>
          <p>Rank community/talk pairs using explicit topic fit, community quality, relationship strength, talk readiness, timing and current outreach state.</p>
        </div>
        <a className="button ghost" href="/integrations">Discover more communities →</a>
      </header>

      {!data.connected && <div className="notice"><strong>Opportunity API is not connected.</strong><span>Start PostgreSQL and the Go API to calculate speaking matches.</span></div>}

      <section className="metrics-grid">
        <article className="metric-card"><span className="eyebrow">Pairings</span><strong className="metric-value">{data.items.length}</strong><span className="muted">eligible community/talk combinations</span></article>
        <article className="metric-card"><span className="eyebrow">High Fit</span><strong className="metric-value">{highFit}</strong><span className="muted">score 70 or higher</span></article>
        <article className="metric-card"><span className="eyebrow">Warm Context</span><strong className="metric-value">{warm}</strong><span className="muted">pairings with a non-cold relationship</span></article>
        <article className="metric-card"><span className="eyebrow">Upcoming Events</span><strong className="metric-value">{withUpcomingEvents}</strong><span className="muted">pairings with known timing</span></article>
      </section>

      <section className="panel opportunity-panel">
        <div className="panel-head">
          <div><span className="eyebrow">Ranked Pipeline</span><h2>Best speaking opportunities</h2></div>
          <span className="muted panel-note">Scores are deterministic and inspectable.</span>
        </div>

        <div className="opportunity-list">
          {data.items.length === 0 ? <p className="empty-copy">No pairings yet. Add communities and talks, then return here.</p> : data.items.map((item) => {
            const outreachHref = `/outreach?communityId=${encodeURIComponent(item.community.id)}&talkId=${encodeURIComponent(item.talk.id)}&score=${item.score}`;
            return (
              <article className="opportunity-card" key={`${item.community.id}-${item.talk.id}`}>
                <div className="opportunity-main">
                  <div className="opportunity-title-row">
                    <div>
                      <span className="eyebrow">{item.community.platform || "community"}</span>
                      <h3>{item.community.name}</h3>
                      <span className="subline">{locationLabel(item.community.city, item.community.country)} · {item.community.topics.join(" · ") || "topics not enriched"}</span>
                    </div>
                    <div className={scoreClass(item.score)}><strong>{item.score}</strong><span>/100</span></div>
                  </div>

                  <div className="opportunity-talk">
                    <span className="eyebrow">Recommended talk</span>
                    <strong>{item.talk.title}</strong>
                    <span>{item.talk.status} · {item.talk.durationMinutes} min · {item.talk.topics.join(" · ") || "no topics"}</span>
                  </div>

                  <div className="score-breakdown" aria-label="Score breakdown">
                    <span>Topic <strong>{item.breakdown.topicFit}</strong></span>
                    <span>Community <strong>{item.breakdown.communityQuality}</strong></span>
                    <span>Relationship <strong>{item.breakdown.relationship}</strong></span>
                    <span>Readiness <strong>{item.breakdown.talkReadiness}</strong></span>
                    <span>Timing <strong>{item.breakdown.timing}</strong></span>
                    {item.breakdown.penalty > 0 && <span className="penalty">Penalty <strong>-{item.breakdown.penalty}</strong></span>}
                  </div>

                  <div className="reason-list">
                    {item.reasons.length === 0 ? <span>No score evidence yet.</span> : item.reasons.map((reason) => <span key={reason}>{reason}</span>)}
                  </div>
                </div>

                <aside className="opportunity-side">
                  <div><span className="eyebrow">Relationship</span><strong>{item.relationship?.stage ?? "not started"}</strong><span>{item.relationship ? `${item.relationship.strength}/100 strength` : "Create a relationship before outreach"}</span></div>
                  <div><span className="eyebrow">Next event</span><strong>{formatDate(item.community.nextEventAt)}</strong><span>{item.existingOutreachState ? `Outreach: ${item.existingOutreachState}` : "No active outreach"}</span></div>
                  <a className="button primary" href={outreachHref}>Prepare outreach →</a>
                  {item.community.websiteUrl && <a className="button ghost" href={item.community.websiteUrl} target="_blank" rel="noreferrer">Open community ↗</a>}
                </aside>
              </article>
            );
          })}
        </div>
      </section>
    </div>
  );
}
