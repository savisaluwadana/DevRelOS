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

export type Talk = {
  id: string;
  projectId: string;
  title: string;
  abstract: string;
  description: string;
  level: string;
  durationMinutes: number;
  topics: string[];
  demoUrl: string;
  slidesUrl: string;
  recordingUrl: string;
  status: string;
};

export type Submission = {
  id: string;
  cfpId: string;
  talkId: string;
  eventName?: string;
  talkTitle?: string;
  status: string;
  submittedAt?: string;
  decisionAt?: string;
  notes: string;
  fitScore?: number;
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

export type Connector = {
  id: string;
  workspaceId: string;
  provider: string;
  name: string;
  enabled: boolean;
  config: Record<string, unknown>;
  policy: Record<string, unknown>;
  createdAt: string;
  updatedAt: string;
};

export type ConnectorRun = {
  id: string;
  connectorId: string;
  status: string;
  cursor: string;
  requestsMade: number;
  itemsFetched: number;
  itemsCreated: number;
  itemsUpdated: number;
  itemsSkipped: number;
  providerCostUsd?: number;
  warnings: string[];
  error: string;
  startedAt?: string;
  finishedAt?: string;
  createdAt: string;
};

export type Signal = {
  id: string;
  projectId: string;
  sourceRecordId: string;
  provider: string;
  externalId: string;
  canonicalUrl: string;
  authorHandle: string;
  authorName: string;
  title: string;
  body: string;
  occurredAt?: string;
  topics: string[];
  engagementScore: number;
  relevanceScore?: number;
  status: "new" | "reviewed" | "ignored" | "converted";
  createdAt: string;
  updatedAt: string;
};

export type PainPoint = {
  id: string;
  projectId: string;
  key: string;
  title: string;
  summary: string;
  persona: string;
  severity: number;
  trendScore: number;
  evidenceCount: number;
  topics: string[];
  status: string;
  firstSeenAt?: string;
  lastSeenAt?: string;
  createdAt: string;
  updatedAt: string;
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

async function getJSON<T>(path: string): Promise<T | null> {
  try {
    const response = await fetch(`${apiURL}${path}`, {
      cache: "no-store",
      signal: AbortSignal.timeout(3000)
    });
    if (!response.ok) return null;
    return (await response.json()) as T;
  } catch {
    return null;
  }
}

export function getDashboard(): Promise<Dashboard | null> {
  return getJSON<Dashboard>("/api/v1/dashboard");
}

export async function getOperatorData() {
  const [events, cfps, talks, submissions, communities] = await Promise.all([
    getJSON<Event[]>("/api/v1/events"),
    getJSON<CFP[]>("/api/v1/cfps"),
    getJSON<Talk[]>("/api/v1/talks"),
    getJSON<Submission[]>("/api/v1/submissions"),
    getJSON<Community[]>("/api/v1/communities")
  ]);

  return {
    events: events ?? [],
    cfps: cfps ?? [],
    talks: talks ?? [],
    submissions: submissions ?? [],
    communities: communities ?? [],
    connected: [events, cfps, talks, submissions, communities].every((value) => value !== null)
  };
}

export async function getIntegrationData() {
  const connectors = await getJSON<Connector[]>("/api/v1/connectors");
  const runs = await Promise.all(
    (connectors ?? []).map(async (connector) => ({
      connectorId: connector.id,
      runs: (await getJSON<ConnectorRun[]>(`/api/v1/connectors/${connector.id}/runs`)) ?? []
    }))
  );

  return {
    connectors: connectors ?? [],
    runs: new Map(runs.map((item) => [item.connectorId, item.runs])),
    connected: connectors !== null
  };
}

export async function getSignalRadarData() {
  const [signals, painPoints] = await Promise.all([
    getJSON<Signal[]>("/api/v1/signals?limit=200"),
    getJSON<PainPoint[]>("/api/v1/pain-points?status=active&limit=100")
  ]);

  return {
    signals: signals ?? [],
    painPoints: painPoints ?? [],
    connected: signals !== null && painPoints !== null
  };
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

export function formatDateTime(value?: string): string {
  if (!value) return "TBD";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "TBD";
  return new Intl.DateTimeFormat("en", {
    month: "short",
    day: "numeric",
    hour: "numeric",
    minute: "2-digit"
  }).format(date);
}

export function locationLabel(city?: string, country?: string): string {
  return [city, country].filter(Boolean).join(", ") || "Location TBD";
}
