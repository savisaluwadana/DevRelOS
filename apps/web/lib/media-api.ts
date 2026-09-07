import { serverFetch } from "@/lib/server-api";

export type MediaSegment = { startMs: number; endMs: number; text: string };
export type MediaAsset = {
  id: string; projectId: string; contentAssetId: string; title: string; sourcePath: string; sourceUrl: string;
  mediaType: "video" | "audio"; durationMs: number; status: string; transcriptText: string; transcriptLanguage: string;
  transcriptSegments: MediaSegment[]; metadata: Record<string, unknown>; createdAt: string; updatedAt: string;
};
export type MediaClip = {
  id: string; projectId: string; mediaAssetId: string; contentAssetId: string; title: string; startMs: number; endMs: number;
  aspectRatio: "9:16" | "1:1" | "16:9"; score: number; rationale: string; captionText: string; status: string;
  outputPath: string; metadata: Record<string, unknown>; createdAt: string; updatedAt: string;
};

export async function getMediaData(): Promise<{ assets: MediaAsset[]; clips: MediaClip[]; connected: boolean }> {
  try {
    const [assetsResponse, clipsResponse] = await Promise.all([
      serverFetch("/api/v1/media-assets?limit=200", { signal: AbortSignal.timeout(3000) }),
      serverFetch("/api/v1/media-clips", { signal: AbortSignal.timeout(3000) })
    ]);
    if (!assetsResponse.ok || !clipsResponse.ok) return { assets: [], clips: [], connected: false };
    return { assets: await assetsResponse.json(), clips: await clipsResponse.json(), connected: true };
  } catch {
    return { assets: [], clips: [], connected: false };
  }
}

export function msLabel(value: number) {
  const total = Math.max(0, Math.floor(value / 1000));
  const minutes = Math.floor(total / 60);
  const seconds = total % 60;
  return `${minutes}:${seconds.toString().padStart(2, "0")}`;
}
