export type FeedbackStatus = "new" | "triaged" | "planned" | "in_progress" | "shipped" | "closed" | "wont_fix";

export type FeedbackItem = {
  id: string;
  projectId: string;
  sourceType: string;
  sourceId: string;
  title: string;
  summary: string;
  persona: string;
  component: string;
  impactScore: number;
  frequencyScore: number;
  status: FeedbackStatus;
  owner: string;
  githubRepository: string;
  githubIssueNumber?: number;
  githubIssueUrl: string;
  githubIssueTitle: string;
  githubIssueBody: string;
  followUpNote: string;
  shippedAt?: string;
  metadata: Record<string, unknown>;
  createdAt: string;
  updatedAt: string;
};

const apiURL = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";

export async function getFeedbackItems(): Promise<{ items: FeedbackItem[]; connected: boolean }> {
  try {
    const response = await fetch(`${apiURL}/api/v1/feedback?limit=300`, {
      cache: "no-store",
      signal: AbortSignal.timeout(3000)
    });
    if (!response.ok) return { items: [], connected: false };
    return { items: (await response.json()) as FeedbackItem[], connected: true };
  } catch {
    return { items: [], connected: false };
  }
}

export const feedbackStatusLabels: Record<FeedbackStatus, string> = {
  new: "New",
  triaged: "Triaged",
  planned: "Planned",
  in_progress: "In progress",
  shipped: "Shipped",
  closed: "Closed",
  wont_fix: "Won't fix"
};
