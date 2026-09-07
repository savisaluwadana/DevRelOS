export type CFP = {
  id: string;
  eventId: string;
  eventName?: string;
  name: string;
  submissionUrl: string;
  opensAt?: string;
  closesAt?: string;
  tracks: string[];
  requirements: string;
  status: string;
  fitScore?: number;
  scoreReason: Record<string, unknown>;
};

export type Event = {
  id: string;
  projectId: string;
  name: string;
  description: string;
  websiteUrl: string;
  city: string;
  country: string;
  timezone: string;
  startsAt?: string;
  endsAt?: string;
  eventType: string;
  topics: string[];
  status: string;
};

export type Community = {
  id: string;
  projectId: string;
  name: string;
  platform: string;
  externalId: string;
  websiteUrl: string;
  city: string;
  country: string;
  timezone: string;
  topics: string[];
  memberCount?: number;
  activityScore?: number;
  speakingFitScore?: number;
  nextEventAt?: string;
  status: string;
};

export type Dashboard = {
  openCfps: number;
  closingSoon: number;
  acceptedTalks: number;
  submissionsInFlight: number;
  highFitCfps: CFP[] | null;
  upcomingEvents: Event[] | null;
  communityOpportunities: Community[] | null;
};

const apiURL = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";

export async function getDashboard(): Promise<Dashboard | null> {
  try {
    const response = await fetch(`${apiURL}/api/v1/dashboard`, {
      cache: "no-store",
      signal: AbortSignal.timeout(3000)
    });
    if (!response.ok) return null;
    return (await response.json()) as Dashboard;
  } catch {
    return null;
  }
}

export function formatDate(value?: string): string {
  if (!value) return "TBD";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "TBD";
  return new Intl.DateTimeFormat("en", {
    month: "short",
    day: "numeric",
    year: "numeric"
  }).format(date);
}

export function locationLabel(city?: string, country?: string): string {
  return [city, country].filter(Boolean).join(", ") || "Location TBD";
}
