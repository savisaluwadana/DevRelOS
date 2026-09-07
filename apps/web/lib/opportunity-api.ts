import type { Community, Talk } from "@/lib/api";
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

const apiURL = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";

export async function getSpeakingOpportunities(minScore = 0): Promise<{ items: SpeakingOpportunity[]; connected: boolean }> {
  try {
    const response = await fetch(`${apiURL}/api/v1/opportunities/speaking?limit=100&minScore=${minScore}`, {
      cache: "no-store",
      signal: AbortSignal.timeout(3000)
    });
    if (!response.ok) return { items: [], connected: false };
    return { items: (await response.json()) as SpeakingOpportunity[], connected: true };
  } catch {
    return { items: [], connected: false };
  }
}
