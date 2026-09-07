export type IdentityUser = {
  id: string;
  email: string;
  displayName: string;
  status: "active" | "disabled";
  createdAt: string;
  updatedAt: string;
};

export type MembershipRole = "owner" | "admin" | "editor" | "viewer";

export type IdentityMembership = {
  workspaceId: string;
  workspaceName: string;
  userId: string;
  email: string;
  displayName: string;
  role: MembershipRole;
  createdAt: string;
  updatedAt: string;
};

export type IdentityAPIKey = {
  id: string;
  userId: string;
  name: string;
  keyPrefix: string;
  expiresAt?: string;
  lastUsedAt?: string;
  revokedAt?: string;
  createdAt: string;
};

export type WorkspaceInvitation = {
  id: string;
  workspaceId: string;
  email: string;
  role: MembershipRole;
  invitedByUserId: string;
  expiresAt: string;
  acceptedAt?: string;
  createdAt: string;
};

export type ConnectorSecret = {
  id: string;
  workspaceId: string;
  provider: string;
  name: string;
  keyVersion: number;
  createdByUserId: string;
  rotatedAt?: string;
  createdAt: string;
  updatedAt: string;
};

export type AuditEvent = {
  id: string;
  workspaceId: string;
  actorUserId: string;
  actorKind: "user" | "operator" | "system";
  actorEmail: string;
  action: string;
  resourceType: string;
  resourceId: string;
  metadata: Record<string, unknown>;
  createdAt: string;
};
