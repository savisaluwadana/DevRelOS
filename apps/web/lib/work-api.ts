import { serverFetch } from "@/lib/server-api";
import type { WorkItem } from "@/lib/work-shared";
export * from "@/lib/work-shared";

export async function getWorkItems(): Promise<{ items: WorkItem[]; connected: boolean }> {
  try {
    const response = await serverFetch("/api/v1/work-items?limit=300", { signal: AbortSignal.timeout(3000) });
    if (!response.ok) return { items: [], connected: false };
    return { items: (await response.json()) as WorkItem[], connected: true };
  } catch {
    return { items: [], connected: false };
  }
}
