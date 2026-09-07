import { serverFetch } from "@/lib/server-api";
import type { Campaign, CampaignItem, CampaignReport, RelationshipRadarItem } from "@/lib/campaign-shared";
export * from "@/lib/campaign-shared";

export async function getCampaignData(): Promise<{ campaigns: Campaign[]; reports: CampaignReport[]; connected: boolean }> {
  try {
    const response = await serverFetch("/api/v1/campaigns", { signal: AbortSignal.timeout(3000) });
    if (!response.ok) return { campaigns: [], reports: [], connected: false };
    const campaigns = (await response.json()) as Campaign[];
    const reports = (await Promise.all(campaigns.slice(0, 20).map(async (campaign) => {
      const [reportResponse, itemsResponse] = await Promise.all([
        serverFetch(`/api/v1/campaigns/${encodeURIComponent(campaign.id)}/report`, { signal: AbortSignal.timeout(3000) }),
        serverFetch(`/api/v1/campaigns/${encodeURIComponent(campaign.id)}/items`, { signal: AbortSignal.timeout(3000) })
      ]);
      if (!reportResponse.ok) return null;
      const report = (await reportResponse.json()) as CampaignReport;
      const items = itemsResponse.ok ? (await itemsResponse.json()) as CampaignItem[] : [];
      return { ...report, items };
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
