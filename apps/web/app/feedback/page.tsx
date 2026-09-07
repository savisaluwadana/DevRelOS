import { CampaignAttribution } from "@/components/campaign-attribution";
import { FeedbackEditor, ManualFeedbackForm } from "@/components/feedback-actions";
import { FeedbackGitHubActions } from "@/components/feedback-github-actions";
import { feedbackStatusLabels, getFeedbackItems } from "@/lib/feedback-api";
import { formatDateTime } from "@/lib/api";

function scoreClass(score: number) {
  if (score >= 80) return "feedback-score critical";
  if (score >= 60) return "feedback-score high";
  return "feedback-score";
}

export default async function FeedbackPage() {
  const data = await getFeedbackItems();
  const highImpact = data.items.filter((item) => item.impactScore >= 70).length;
  const active = data.items.filter((item) => item.status === "planned" || item.status === "in_progress").length;
  const shipped = data.items.filter((item) => item.status === "shipped" || item.status === "closed").length;
  const githubLinked = data.items.filter((item) => item.githubIssueUrl || item.githubIssueNumber).length;

  return (
    <div className="page-wrap feedback-page">
      <header className="topbar compact-topbar">
        <div>
          <span className="eyebrow">Developer Feedback</span>
          <h1>Close the loop from developer pain to shipped product change.</h1>
          <p>Triage evidence-backed product feedback, prepare reviewable GitHub issue drafts, track engineering motion and preserve the follow-up developers should hear when something ships.</p>
        </div>
        <a className="button ghost" href="/signals">Signal Radar →</a>
      </header>

      {!data.connected && <div className="notice"><strong>Feedback API is not connected.</strong><span>Apply the latest migrations and start the Go API to use this workspace.</span></div>}

      <section className="metrics-grid">
        <article className="metric-card"><span className="eyebrow">Feedback</span><strong className="metric-value">{data.items.length}</strong><span className="muted">tracked product signals</span></article>
        <article className="metric-card"><span className="eyebrow">High Impact</span><strong className="metric-value">{highImpact}</strong><span className="muted">impact score 70+</span></article>
        <article className="metric-card"><span className="eyebrow">Planned / Building</span><strong className="metric-value">{active}</strong><span className="muted">engineering motion</span></article>
        <article className="metric-card"><span className="eyebrow">GitHub / Shipped</span><strong className="metric-value">{githubLinked} / {shipped}</strong><span className="muted">issue linkage and closed loops</span></article>
      </section>

      <section className="panel feedback-capture-panel">
        <div className="panel-head"><div><span className="eyebrow">Manual feedback</span><h2>Capture product feedback directly</h2></div><span className="muted panel-note">For recurring developer pain, create feedback from Signal Radar so evidence and scoring are inherited.</span></div>
        <div className="feedback-form-body"><ManualFeedbackForm /></div>
      </section>

      <section className="panel feedback-pipeline-panel">
        <div className="panel-head"><div><span className="eyebrow">Product Loop</span><h2>New → triage → plan → build → ship</h2></div><span className="muted panel-note">Issue creation stays explicit: DevRelOS opens a prefilled GitHub form, then syncs linked public issue state without auto-changing feedback status.</span></div>
        <div className="feedback-list">
          {data.items.length === 0 ? <p className="empty-copy">No product feedback yet. Convert a pain point from Signal Radar or capture one above.</p> : data.items.map((item) => {
            const evidenceCount = typeof item.metadata?.evidenceCount === "number" ? Number(item.metadata.evidenceCount) : undefined;
            const githubState = typeof item.metadata?.githubIssueState === "string" ? String(item.metadata.githubIssueState) : "";
            const githubSyncedAt = typeof item.metadata?.githubIssueSyncedAt === "string" ? String(item.metadata.githubIssueSyncedAt) : "";
            const githubComments = typeof item.metadata?.githubIssueComments === "number" ? Number(item.metadata.githubIssueComments) : undefined;
            return (
              <article className="feedback-card" key={item.id}>
                <div className="feedback-card-main">
                  <div className="feedback-card-head">
                    <div>
                      <div className="feedback-kickers"><span className="pill neutral">{feedbackStatusLabels[item.status]}</span>{item.component && <span className="platform-badge">{item.component}</span>}{githubState && <span className={`pill github-issue-state ${githubState}`}>GitHub {githubState}</span>}</div>
                      <h3>{item.title}</h3>
                    </div>
                    <div className="feedback-scores">
                      <div className={scoreClass(item.impactScore)}><strong>{item.impactScore}</strong><span>impact</span></div>
                      <div className={scoreClass(item.frequencyScore)}><strong>{item.frequencyScore}</strong><span>frequency</span></div>
                    </div>
                  </div>

                  {item.summary && <p>{item.summary}</p>}

                  <div className="feedback-meta">
                    <span>Persona: {item.persona || "—"}</span>
                    <span>Owner: {item.owner || "unassigned"}</span>
                    <span>Source: {item.sourceType.replaceAll("_", " ")}</span>
                    <span>Evidence: {evidenceCount ?? "—"}</span>
                    <span>Updated: {formatDateTime(item.updatedAt)}</span>
                    {githubSyncedAt && <span>GitHub sync: {formatDateTime(githubSyncedAt)}{githubComments !== undefined ? ` · ${githubComments} comments` : ""}</span>}
                  </div>

                  {(item.githubIssueTitle || item.githubIssueBody) && (
                    <details className="feedback-issue-draft">
                      <summary>Review GitHub issue draft</summary>
                      <div className="feedback-issue-copy"><strong>{item.githubIssueTitle || item.title}</strong><pre>{item.githubIssueBody}</pre></div>
                    </details>
                  )}

                  <div className="feedback-links">
                    {item.githubIssueUrl && <a href={item.githubIssueUrl} target="_blank" rel="noreferrer">GitHub issue #{item.githubIssueNumber ?? ""} ↗</a>}
                    {!item.githubIssueUrl && item.githubRepository && <span>Target repo: {item.githubRepository}</span>}
                    {item.sourceType === "pain_point" && <a href="/signals">View source pain point →</a>}
                  </div>

                  {item.followUpNote && <div className="feedback-follow-up"><span className="eyebrow">Developer follow-up</span><p>{item.followUpNote}</p></div>}
                </div>
                <aside className="feedback-card-side">
                  <CampaignAttribution entityType="feedback" entityId={item.id} />
                  <FeedbackGitHubActions item={item} />
                  <FeedbackEditor item={item} />
                </aside>
              </article>
            );
          })}
        </div>
      </section>
    </div>
  );
}
