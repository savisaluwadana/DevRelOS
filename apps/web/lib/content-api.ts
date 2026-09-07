import { serverFetch } from "@/lib/server-api";

export type ContentStatus = "brief" | "drafting" | "review" | "approved" | "published" | "archived";

export type ContentChannel =
  | "blog"
  | "linkedin"
  | "x"
  | "newsletter"
  | "youtube"
  | "short_video"
  | "docs"
  | "talk"
  | "community";

export type ContentAsset = {
  id: string;
  projectId: string;
  workItemId: string;
  channel: ContentChannel;
  format: string;
  title: string;
  audience: string;
  objective: string;
  brief: string;
  draft: string;
  status: ContentStatus;
  topics: string[];
  sourceUrl: string;
  publishedUrl: string;
  scheduledAt?: string;
  publishedAt?: string;
  metadata: Record<string, unknown>;
  createdAt: string;
  updatedAt: string;
};

export async function getContentAssets(): Promise<{ items: ContentAsset[]; connected: boolean }> {
  try {
    const response = await serverFetch("/api/v1/content-assets?limit=300", { signal: AbortSignal.timeout(3000) });
    if (!response.ok) return { items: [], connected: false };
    return { items: (await response.json()) as ContentAsset[], connected: true };
  } catch {
    return { items: [], connected: false };
  }
}

export const contentStatusLabels: Record<ContentStatus, string> = {
  brief: "Brief",
  drafting: "Drafting",
  review: "Review",
  approved: "Approved",
  published: "Published",
  archived: "Archived"
};

export const contentChannelLabels: Record<ContentChannel, string> = {
  blog: "Blog",
  linkedin: "LinkedIn",
  x: "X",
  newsletter: "Newsletter",
  youtube: "YouTube",
  short_video: "Short video",
  docs: "Documentation",
  talk: "Talk",
  community: "Community"
};
