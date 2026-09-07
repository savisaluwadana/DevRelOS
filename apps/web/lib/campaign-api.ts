import { serverFetch } from "@/lib/server-api";
import type { Campaign, CampaignReport, RelationshipRadarItem } from "@/lib/campaign-shared";
export * from "@/lib/campaign-shared";

export async function getCampaignData(): Promise<{ campaigns: Campaign[]; reports: CampaignReport[]; connected: boolean }> {
  try {
    const response = await serverFetch("/api/v1/campaigns", { signal: AbortSignal.timeout(3000) });
    if (!response.ok) return { campaigns: [], reports: [], connected: false };
    const campaigns = (await response.json()) as Campaign[];
    const reports = (await Promise.all(campaigns.slice(0, 20).map(async (campaign) => {
      const report = await serverFetch(`/api/v1/campaigns/${encodeURIComponent(campaign.id)}/report`, { signal: AbortSignal.timeout(3000) });
      return report.ok ? (await report.json()) as CampaignReport : null;
    }))).filter((item): item is CampaignReport => Boolean(item));
    return { campaigns, reports, connected: true };
  } catch {
    return { campaigns: [], reports: [], connected: false };
  }
}

export async function getRelationshipRadar(): Promise<{ items: RelationshipRadarItem[]; connected: boolean }> {
  try {
    const response = await serverFetch("/api/v1/relationships/radar", { signal: AbortSignal.timeout(3000) });
    if (!response.ok) return { items: [], connected: false };
    return { items: (await response.json()) as RelationshipRadarItem[], connected: true };
  } catch {
    return { items: [], connected: false };
  }
}
