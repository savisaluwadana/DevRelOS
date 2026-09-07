import type { CFP, Community, Event, Submission, Talk } from "@/lib/api";
import type { Relationship } from "@/lib/outreach-api";

export type OpportunityScoreBreakdown = {
  topicFit: number;
  communityQuality: number;
  relationship: number;
  talkReadiness: number;
  timing: number;
  penalty: number;
};

export type SpeakingOpportunity = {
  community: Community;
  talk: Talk;
  score: number;
  breakdown: OpportunityScoreBreakdown;
  reasons: string[];
  relationship?: Relationship;
  existingOutreachState?: string;
};

export type CFPScoreBreakdown = {
  topicFit: number;
  readiness: number;
  deadline: number;
  existingFit: number;
  submissionGap: number;
  penalty: number;
};

export type CFPOpportunity = {
  cfp: CFP;
  event: Event;
  talk: Talk;
  score: number;
  breakdown: CFPScoreBreakdown;
  reasons: string[];
  existingSubmission?: Submission;
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

export async function getSpeakingOpportunities(minScore = 0): Promise<{ items: SpeakingOpportunity[]; connected: boolean }> {
  const items = await getJSON<SpeakingOpportunity[]>(`/api/v1/opportunities/speaking?limit=100&minScore=${minScore}`);
  return { items: items ?? [], connected: items !== null };
}

export async function getOpportunityData(minScore = 0) {
  const [speaking, cfps] = await Promise.all([
    getJSON<SpeakingOpportunity[]>(`/api/v1/opportunities/speaking?limit=100&minScore=${minScore}`),
    getJSON<CFPOpportunity[]>(`/api/v1/opportunities/cfps?limit=100&minScore=${minScore}`)
  ]);
  return {
    speaking: speaking ?? [],
    cfps: cfps ?? [],
    connected: speaking !== null && cfps !== null
  };
}
