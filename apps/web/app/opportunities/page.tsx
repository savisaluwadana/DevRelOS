import { CreateSubmissionDraftButton } from "@/components/cfp-opportunity-action";
import { formatDate, locationLabel } from "@/lib/api";
import { getOpportunityData } from "@/lib/opportunity-api";

function scoreClass(score: number) {
  if (score >= 75) return "opportunity-score high";
  if (score >= 50) return "opportunity-score medium";
  return "opportunity-score low";
}

export default async function OpportunitiesPage() {
  const data = await getOpportunityData();
  const highFitSpeaking = data.speaking.filter((item) => item.score >= 70).length;
  const highFitCFPs = data.cfps.filter((item) => item.score >= 70).length;
  const warm = data.speaking.filter((item) => item.relationship && item.relationship.stage !== "cold").length;
  const closingSoon = data.cfps.filter((item) => item.cfp.closesAt && new Date(item.cfp.closesAt).getTime() - Date.now() <= 14 * 24 * 60 * 60 * 1000).length;

  return (
    <div className="page-wrap opportunities-page">
      <header className="topbar compact-topbar">
        <div>
          <span className="eyebrow">Opportunity Intelligence</span>
          <h1>Match the right DevRel action to the right opportunity.</h1>
          <p>Rank community speaking targets and open CFPs using inspectable fit signals instead of manually scanning every event, group and reusable talk.</p>
        </div>
        <a className="button ghost" href="/integrations">Discover more opportunities →</a>
      </header>

      {!data.connected && <div className="notice"><strong>Opportunity API is not connected.</strong><span>Start PostgreSQL and the Go API to calculate opportunity matches.</span></div>}

      <section className="metrics-grid">
        <article className="metric-card"><span className="eyebrow">Speaking Matches</span><strong className="metric-value">{data.speaking.length}</strong><span className="muted">eligible community/talk pairs</span></article>
        <article className="metric-card"><span className="eyebrow">High-Fit Speaking</span><strong className="metric-value">{highFitSpeaking}</strong><span className="muted">score 70 or higher</span></article>
        <article className="metric-card"><span className="eyebrow">High-Fit CFPs</span><strong className="metric-value">{highFitCFPs}</strong><span className="muted">talk/CFP pairs at 70+</span></article>
        <article className="metric-card"><span className="eyebrow">Closing Soon</span><strong className="metric-value">{closingSoon}</strong><span className="muted">ranked CFPs within 14 days</span></article>
      </section>

      <section className="panel opportunity-panel">
        <div className="panel-head">
          <div><span className="eyebrow">Community Pipeline</span><h2>Best speaking opportunities</h2></div>
          <span className="muted panel-note">{warm} pairings have warm relationship context.</span>
        </div>

        <div className="opportunity-list">
          {data.speaking.length === 0 ? <p className="empty-copy">No pairings yet. Add communities and talks, then return here.</p> : data.speaking.map((item) => {
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

                  <div className="score-breakdown" aria-label="Speaking score breakdown">
                    <span>Topic <strong>{item.breakdown.topicFit}</strong></span>
                    <span>Community <strong>{item.breakdown.communityQuality}</strong></span>
                    <span>Relationship <strong>{item.breakdown.relationship}</strong></span>
                    <span>Readiness <strong>{item.breakdown.talkReadiness}</strong></span>
                    <span>Timing <strong>{item.breakdown.timing}</strong></span>
                    {item.breakdown.penalty > 0 && <span className="penalty">Penalty <strong>-{item.breakdown.penalty}</strong></span>}
                  </div>

                  <div className="reason-list">{item.reasons.length === 0 ? <span>No score evidence yet.</span> : item.reasons.map((reason) => <span key={reason}>{reason}</span>)}</div>
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

      <section className="panel opportunity-panel">
        <div className="panel-head">
          <div><span className="eyebrow">Conference Pipeline</span><h2>Best CFP / talk matches</h2></div>
          <a className="button ghost" href="/manage#submissions">Submission pipeline →</a>
        </div>

        <div className="opportunity-list">
          {data.cfps.length === 0 ? <p className="empty-copy">No open CFP/talk pairings yet.</p> : data.cfps.map((item) => (
            <article className="opportunity-card cfp-opportunity-card" key={`${item.cfp.id}-${item.talk.id}`}>
              <div className="opportunity-main">
                <div className="opportunity-title-row">
                  <div>
                    <span className="eyebrow">Open CFP</span>
                    <h3>{item.event.name || item.cfp.eventName || item.cfp.name}</h3>
                    <span className="subline">Closes {formatDate(item.cfp.closesAt)} · {item.cfp.tracks.join(" · ") || "tracks not specified"}</span>
                  </div>
                  <div className={scoreClass(item.score)}><strong>{item.score}</strong><span>/100</span></div>
                </div>

                <div className="opportunity-talk">
                  <span className="eyebrow">Recommended talk</span>
                  <strong>{item.talk.title}</strong>
                  <span>{item.talk.status} · {item.talk.topics.join(" · ") || "no topics"}</span>
                </div>

                <div className="score-breakdown" aria-label="CFP score breakdown">
                  <span>Topic <strong>{item.breakdown.topicFit}</strong></span>
                  <span>Readiness <strong>{item.breakdown.readiness}</strong></span>
                  <span>Deadline <strong>{item.breakdown.deadline}</strong></span>
                  <span>Project fit <strong>{item.breakdown.existingFit}</strong></span>
                  <span>Submission gap <strong>{item.breakdown.submissionGap}</strong></span>
                  {item.breakdown.penalty > 0 && <span className="penalty">Penalty <strong>-{item.breakdown.penalty}</strong></span>}
                </div>
                <div className="reason-list">{item.reasons.length === 0 ? <span>No score evidence yet.</span> : item.reasons.map((reason) => <span key={reason}>{reason}</span>)}</div>
              </div>

              <aside className="opportunity-side">
                <div><span className="eyebrow">CFP deadline</span><strong>{formatDate(item.cfp.closesAt)}</strong><span>{item.existingSubmission ? `Submission: ${item.existingSubmission.status}` : "No submission yet"}</span></div>
                <div><span className="eyebrow">Event</span><strong>{formatDate(item.event.startsAt)}</strong><span>{locationLabel(item.event.city, item.event.country)}</span></div>
                <CreateSubmissionDraftButton cfpId={item.cfp.id} talkId={item.talk.id} disabled={Boolean(item.existingSubmission)} />
                {item.cfp.submissionUrl && <a className="button ghost" href={item.cfp.submissionUrl} target="_blank" rel="noreferrer">Open CFP ↗</a>}
              </aside>
            </article>
          ))}
        </div>
      </section>
    </div>
  );
}
