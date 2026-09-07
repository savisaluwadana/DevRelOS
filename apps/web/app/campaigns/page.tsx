import { CampaignControls, CreateCampaignForm } from "@/components/campaign-actions";
import { CampaignLinkedItems } from "@/components/campaign-linked-items";
import { getCampaignData } from "@/lib/campaign-api";

function money(value: number) {
  return new Intl.NumberFormat("en-US", { style: "currency", currency: "USD", maximumFractionDigits: 0 }).format(value || 0);
}

function percent(value: number) {
  return `${Math.round(value || 0)}%`;
}

export default async function CampaignsPage() {
  const data = await getCampaignData();
  const active = data.campaigns.filter((campaign) => campaign.status === "active").length;
  const spend = data.reports.reduce((sum, report) => sum + report.spendUsd, 0);
  const replies = data.reports.reduce((sum, report) => sum + report.outreachReplies, 0);
  const accepted = data.reports.reduce((sum, report) => sum + report.acceptedTalks, 0);

  return (
    <div className="page-wrap campaigns-page">
      <header className="topbar compact-topbar">
        <div>
          <span className="eyebrow">Campaigns & Attribution</span>
          <h1>Connect DevRel activity to outcomes.</h1>
          <p>Group content, talks, events, outreach, feedback and engineering work into initiatives, then measure what actually moved.</p>
        </div>
        <a className="button ghost" href="/relationships">Relationship Radar →</a>
      </header>

      {!data.connected && <div className="notice"><strong>Campaign API is unavailable.</strong><span>Run the API and migration 000011 to enable attribution.</span></div>}

      <section className="metrics-grid">
        <article className="metric-card"><span className="eyebrow">Campaigns</span><strong className="metric-value">{data.campaigns.length}</strong><span className="muted">{active} currently active</span></article>
        <article className="metric-card"><span className="eyebrow">Attributed spend</span><strong className="metric-value">{money(spend)}</strong><span className="muted">linked activity cost</span></article>
        <article className="metric-card"><span className="eyebrow">Replies</span><strong className="metric-value">{replies}</strong><span className="muted">campaign-attributed outreach replies</span></article>
        <article className="metric-card"><span className="eyebrow">Accepted talks</span><strong className="metric-value">{accepted}</strong><span className="muted">accepted linked CFP submissions</span></article>
      </section>

      <section className="panel campaign-create-panel">
        <div className="panel-head"><div><span className="eyebrow">New initiative</span><h2>Create a measurable campaign</h2></div><span className="muted panel-note">Budget is optional. Outcomes can combine derived and manually recorded metrics.</span></div>
        <CreateCampaignForm />
      </section>

      <section className="campaign-grid">
        {data.reports.length === 0 ? <div className="panel"><p className="empty-copy">No campaigns yet. Create one to start connecting activity with outcomes.</p></div> : data.reports.map((report) => (
          <article className="panel campaign-card" key={report.campaign.id}>
            <div className="campaign-card-head">
              <div><span className={`campaign-status ${report.campaign.status}`}>{report.campaign.status}</span><h2>{report.campaign.name}</h2><p>{report.campaign.objective || "No objective recorded yet."}</p></div>
              <div className="outcome-score"><strong>{report.outcomeScore}</strong><span>outcome score</span></div>
            </div>

            <div className="campaign-kpis">
              <div><span>Published</span><strong>{report.publishedContent}</strong></div>
              <div><span>Work done</span><strong>{report.completedWork}</strong></div>
              <div><span>Reply rate</span><strong>{percent(report.replyRate)}</strong></div>
              <div><span>CFP acceptance</span><strong>{percent(report.acceptanceRate)}</strong></div>
              <div><span>Feedback shipped</span><strong>{report.feedbackShipped}</strong></div>
              <div><span>Spend</span><strong>{money(report.spendUsd)}</strong></div>
            </div>

            <div className="campaign-budget"><span>Budget {money(report.campaign.budgetUsd)}</span><span>Remaining {money(report.budgetRemainingUsd)}</span></div>

            <div className="campaign-linked-types">
              {Object.keys(report.linkedByType).length === 0 ? <span>No activity linked yet.</span> : Object.entries(report.linkedByType).map(([kind, count]) => <span key={kind}>{kind.replaceAll("_", " ")} <strong>{count}</strong></span>)}
            </div>

            <div className="campaign-attribution-block">
              <span className="eyebrow">Attributed activity</span>
              <CampaignLinkedItems campaignId={report.campaign.id} items={report.items ?? []} />
            </div>

            {Object.keys(report.metrics).length > 0 && <div className="campaign-manual-metrics">{Object.entries(report.metrics).map(([key, value]) => <div key={key}><span>{key.replaceAll("_", " ")}</span><strong>{value}</strong></div>)}</div>}
            <CampaignControls campaign={report.campaign} />
          </article>
        ))}
      </section>
    </div>
  );
}
