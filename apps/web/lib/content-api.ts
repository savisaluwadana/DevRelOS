import { serverFetch } from "@/lib/server-api";
import type { ContentAsset } from "@/lib/content-shared";
export * from "@/lib/content-shared";

export async function getContentAssets(): Promise<{ items: ContentAsset[]; connected: boolean }> {
  try {
    const response = await serverFetch("/api/v1/content-assets?limit=300", { signal: AbortSignal.timeout(3000) });
    if (!response.ok) return { items: [], connected: false };
    return { items: (await response.json()) as ContentAsset[], connected: true };
  } catch {
    return { items: [], connected: false };
  }
}
