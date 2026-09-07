import { serverFetch } from "@/lib/server-api";

export type CalendarItem = {
  id: string;
  kind: "event" | "cfp" | "work_item" | "content" | "campaign_start" | "campaign_end" | "relationship_follow_up";
  title: string;
  subtitle: string;
  startsAt: string;
  endsAt?: string | null;
  status: string;
  href: string;
  priority: number;
  sourceId: string;
};

export type CalendarPayload = {
  from: string;
  to: string;
  items: CalendarItem[];
};

export const calendarKindLabels: Record<CalendarItem["kind"], string> = {
  event: "Event",
  cfp: "CFP deadline",
  work_item: "Work due",
  content: "Content",
  campaign_start: "Campaign start",
  campaign_end: "Campaign end",
  relationship_follow_up: "Follow-up"
};

export async function getCalendar(): Promise<{ data: CalendarPayload | null; connected: boolean }> {
  try {
    const response = await serverFetch("/api/v1/calendar", { signal: AbortSignal.timeout(3000) });
    if (!response.ok) return { data: null, connected: false };
    return { data: (await response.json()) as CalendarPayload, connected: true };
  } catch {
    return { data: null, connected: false };
  }
}
