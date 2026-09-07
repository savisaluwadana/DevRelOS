import type { Community, Talk } from "@/lib/api";

export type Contact = {
  id: string;
  workspaceId: string;
  name: string;
  role: string;
  email: string;
  publicProfileUrl: string;
  sourceUrl: string;
  doNotContact: boolean;
  createdAt: string;
  updatedAt: string;
};

export type Relationship = {
  id: string;
  projectId: string;
  communityId: string;
  communityName: string;
  contactId: string;
  contactName: string;
  stage: "cold" | "warm" | "engaged" | "partner" | "dormant";
  strength: number;
  lastTouchAt?: string;
  nextFollowUpAt?: string;
  notes: string;
  createdAt: string;
  updatedAt: string;
};

export type Touchpoint = {
  id: string;
  relationshipId: string;
  channel: string;
  direction: "inbound" | "outbound" | "internal";
  summary: string;
  occurredAt: string;
  createdAt: string;
};

export type Outreach = {
  id: string;
  projectId: string;
  communityId: string;
  communityName: string;
  contactId: string;
  contactName: string;
  talkId: string;
  talkTitle: string;
  channel: string;
  subject: string;
  body: string;
  rationale: string;
  status: "draft" | "needs_approval" | "approved" | "queued" | "sent" | "replied" | "failed" | "cancelled";
  approvedAt?: string;
  sentAt?: string;
  createdAt: string;
  updatedAt: string;
};

const apiURL = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";

async function getJSON<T>(path: string): Promise<T | null> {
  try {
    const response = await fetch(`${apiURL}${path}`, { cache: "no-store", signal: AbortSignal.timeout(3000) });
    if (!response.ok) return null;
    return (await response.json()) as T;
  } catch {
    return null;
  }
}

export async function getOutreachData() {
  const [contacts, relationships, touchpoints, outreach, communities, talks] = await Promise.all([
    getJSON<Contact[]>("/api/v1/contacts"),
    getJSON<Relationship[]>("/api/v1/relationships"),
    getJSON<Touchpoint[]>("/api/v1/touchpoints?limit=100"),
    getJSON<Outreach[]>("/api/v1/outreach"),
    getJSON<Community[]>("/api/v1/communities"),
    getJSON<Talk[]>("/api/v1/talks")
  ]);

  return {
    contacts: contacts ?? [],
    relationships: relationships ?? [],
    touchpoints: touchpoints ?? [],
    outreach: outreach ?? [],
    communities: communities ?? [],
    talks: talks ?? [],
    connected: [contacts, relationships, touchpoints, outreach, communities, talks].every((value) => value !== null)
  };
}
