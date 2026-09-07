export type WorkspaceRole = "owner" | "admin" | "editor" | "viewer";

export type IdentityUser = {
  id: string;
  email: string;
  displayName: string;
  status: "active" | "disabled";
  createdAt: string;
  updatedAt: string;
};

export type Membership = {
  workspaceId: string;
  workspaceSlug: string;
  workspaceName: string;
  userId: string;
  email: string;
  displayName: string;
  role: WorkspaceRole;
  createdAt: string;
  updatedAt: string;
};

export type Principal = {
  system: boolean;
  user?: IdentityUser;
  memberships: Membership[];
};

export const roleLabels: Record<WorkspaceRole, string> = {
  owner: "Owner",
  admin: "Admin",
  editor: "Editor",
  viewer: "Viewer"
};
