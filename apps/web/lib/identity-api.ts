import { serverFetch } from "@/lib/server-api";
import type { Connector } from "@/lib/api";
import type {
  AuditEvent,
  ConnectorSecret,
  IdentityAPIKey,
  IdentityMembership,
  IdentityUser,
  WorkspaceInvitation
} from "@/lib/identity-types";

async function getJSON<T>(path: string): Promise<T | null> {
  try {
    const response = await serverFetch(path, { signal: AbortSignal.timeout(3000) });
    if (!response.ok) return null;
    return (await response.json()) as T;
  } catch {
    return null;
  }
}

export async function getAccessData() {
  const [users, memberships, audit, invitations, secrets, connectors] = await Promise.all([
    getJSON<IdentityUser[]>("/api/v1/identity/users"),
    getJSON<IdentityMembership[]>("/api/v1/identity/memberships"),
    getJSON<AuditEvent[]>("/api/v1/audit-events?limit=100"),
    getJSON<WorkspaceInvitation[]>("/api/v1/identity/invitations"),
    getJSON<ConnectorSecret[]>("/api/v1/secrets"),
    getJSON<Connector[]>("/api/v1/connectors")
  ]);

  const keyEntries = await Promise.all(
    (users ?? []).map(async (user) => [
      user.id,
      (await getJSON<IdentityAPIKey[]>(`/api/v1/identity/users/${user.id}/api-keys`)) ?? []
    ] as const)
  );

  return {
    users: users ?? [],
    memberships: memberships ?? [],
    audit: audit ?? [],
    invitations: invitations ?? [],
    secrets: secrets ?? [],
    connectors: connectors ?? [],
    apiKeys: new Map(keyEntries),
    connected: [users, memberships, audit, invitations, secrets, connectors].every((value) => value !== null)
  };
}
