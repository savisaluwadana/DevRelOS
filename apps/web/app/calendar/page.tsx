import { calendarKindLabels, getCalendar, type CalendarItem } from "@/lib/calendar-api";

function dayKey(value: string) {
  return value.slice(0, 10);
}

function dayLabel(value: string) {
  return new Intl.DateTimeFormat("en", { weekday: "short", month: "short", day: "numeric", year: "numeric", timeZone: "UTC" }).format(new Date(value));
}

function timeLabel(value: string) {
  return new Intl.DateTimeFormat("en", { hour: "2-digit", minute: "2-digit", timeZone: "UTC", timeZoneName: "short" }).format(new Date(value));
}

function itemClass(item: CalendarItem) {
  return `calendar-item ${item.kind} ${item.priority >= 85 ? "important" : ""}`;
}

export default async function CalendarPage() {
  const result = await getCalendar();
  const items = result.data?.items ?? [];
  const now = Date.now();
  const sevenDays = now + 7 * 24 * 60 * 60 * 1000;
  const overdue = items.filter((item) => new Date(item.startsAt).getTime() < now && ["cfp", "work_item", "relationship_follow_up"].includes(item.kind)).length;
  const nextSeven = items.filter((item) => {
    const value = new Date(item.startsAt).getTime();
    return value >= now && value <= sevenDays;
  }).length;
  const deadlines = items.filter((item) => item.kind === "cfp").length;
  const followUps = items.filter((item) => item.kind === "relationship_follow_up").length;

  const grouped = new Map<string, CalendarItem[]>();
  for (const item of items) {
    const key = dayKey(item.startsAt);
    grouped.set(key, [...(grouped.get(key) ?? []), item]);
  }

  return (
    <div className="page-wrap calendar-page">
      <header className="topbar compact-topbar">
        <div>
          <span className="eyebrow">Unified DevRel Calendar</span>
          <h1>See the deadlines and commitments that compete for attention.</h1>
          <p>One timeline for CFP closes, events, campaign milestones, scheduled content, due work and relationship follow-ups.</p>
        </div>
        <div className="topbar-actions"><a className="button ghost" href="/work">Action Queue</a><a className="button ghost" href="/campaigns">Campaigns →</a></div>
      </header>

      {!result.connected && <div className="notice"><strong>Calendar API is unavailable.</strong><span>Start the Go API to load the unified schedule.</span></div>}

      <section className="metrics-grid">
        <article className="metric-card"><span className="eyebrow">Scheduled</span><strong className="metric-value">{items.length}</strong><span className="muted">items in the current horizon</span></article>
        <article className="metric-card"><span className="eyebrow">Next 7 days</span><strong className="metric-value">{nextSeven}</strong><span className="muted">near-term commitments</span></article>
        <article className="metric-card"><span className="eyebrow">CFP deadlines</span><strong className="metric-value">{deadlines}</strong><span className="muted">open/upcoming CFP closes</span></article>
        <article className="metric-card"><span className="eyebrow">Overdue / follow-ups</span><strong className="metric-value">{overdue} / {followUps}</strong><span className="muted">attention risk</span></article>
      </section>

      <section className="panel calendar-panel">
        <div className="panel-head">
          <div><span className="eyebrow">90-day operating view</span><h2>Upcoming timeline</h2></div>
          <div className="calendar-legend">
            {Object.entries(calendarKindLabels).map(([kind, label]) => <span key={kind}><i className={`calendar-dot ${kind}`} />{label}</span>)}
          </div>
        </div>

        <div className="calendar-days">
          {grouped.size === 0 ? <p className="empty-copy">Nothing scheduled in this window yet. Add CFP deadlines, due dates, content schedules, campaigns or relationship follow-ups.</p> : Array.from(grouped.entries()).map(([date, dayItems]) => (
            <section className="calendar-day" key={date}>
              <div className="calendar-date"><strong>{dayLabel(`${date}T00:00:00Z`)}</strong><span>{dayItems.length} item{dayItems.length === 1 ? "" : "s"}</span></div>
              <div className="calendar-day-items">
                {dayItems.map((item) => (
                  <a className={itemClass(item)} href={item.href} key={item.id}>
                    <div className="calendar-time"><strong>{timeLabel(item.startsAt)}</strong><span>{calendarKindLabels[item.kind]}</span></div>
                    <div className="calendar-copy"><strong>{item.title}</strong><span>{item.subtitle || item.status}</span></div>
                    <div className="calendar-state"><span className="pill neutral">{item.status.replaceAll("_", " ")}</span><span className="calendar-priority">P{item.priority}</span></div>
                  </a>
                ))}
              </div>
            </section>
          ))}
        </div>
      </section>
    </div>
  );
}
