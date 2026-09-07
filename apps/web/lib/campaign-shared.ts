export type Campaign = {
  id: string;
  projectId: string;
  name: string;
  objective: string;
  status: "planning" | "active" | "paused" | "completed" | "archived";
  startsAt?: string | null;
  endsAt?: string | null;
  budgetUsd: number;
  target: Record<string, unknown>;
  createdAt: string;
  updatedAt: string;
};

export type CampaignItem = {
  id: string;
  campaignId: string;
  entityType: string;
  entityId: string;
  channel: string;
  costUsd: number;
  metadata: Record<string, unknown>;
  createdAt: string;
};

export type CampaignMetric = {
  id: string;
  campaignId: string;
  metricKey: string;
  metricValue: number;
  source: string;
  observedAt: string;
  metadata: Record<string, unknown>;
  createdAt: string;
};

export type CampaignReport = {
  campaign: Campaign;
  items: CampaignItem[];
  linkedByType: Record<string, number>;
  spendUsd: number;
  budgetRemainingUsd: number;
  publishedContent: number;
  completedWork: number;
  outreachSent: number;
  outreachReplies: number;
  replyRate: number;
  submissions: number;
  acceptedTalks: number;
  acceptanceRate: number;
  feedbackShipped: number;
  outcomeScore: number;
  metrics: Record<string, number>;
  recentMetrics: CampaignMetric[];
};

export type RelationshipRadarItem = {
  relationshipId: string;
  projectId: string;
  communityId?: string;
  contactId?: string;
  name: string;
  kind: "community" | "contact";
  stage: string;
  strength: number;
  lastTouchAt?: string | null;
  nextFollowUpAt?: string | null;
  daysSinceTouch?: number | null;
  health: "healthy" | "watch" | "critical";
  riskScore: number;
  recommendedAction: string;
};
