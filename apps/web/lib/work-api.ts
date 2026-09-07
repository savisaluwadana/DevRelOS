import { serverFetch } from "@/lib/server-api";

export type WorkItemKind =
  | "content_brief"
  | "docs_improvement"
  | "product_feedback"
  | "talk_idea"
  | "community_research"
  | "outreach_follow_up"
  | "event_task"
  | "engineering_task";

export type WorkItemStatus = "backlog" | "planned" | "in_progress" | "blocked" | "done" | "cancelled";

export type WorkItem = {
  id: string;
  projectId: string;
  sourceType: string;
  sourceId: string;
  kind: WorkItemKind;
  title: string;
  description: string;
  priority: number;
  status: WorkItemStatus;
  owner: string;
  dueAt?: string;
  metadata: Record<string, unknown>;
  createdAt: string;
  updatedAt: string;
};

export async function getWorkItems(): Promise<{ items: WorkItem[]; connected: boolean }> {
  try {
    const response = await serverFetch("/api/v1/work-items?limit=300", { signal: AbortSignal.timeout(3000) });
    if (!response.ok) return { items: [], connected: false };
    return { items: (await response.json()) as WorkItem[], connected: true };
  } catch {
    return { items: [], connected: false };
  }
}

export const workKindLabels: Record<WorkItemKind, string> = {
  content_brief: "Content brief",
  docs_improvement: "Docs improvement",
  product_feedback: "Product feedback",
  talk_idea: "Talk idea",
  community_research: "Community research",
  outreach_follow_up: "Outreach follow-up",
  event_task: "Event task",
  engineering_task: "Engineering task"
};
