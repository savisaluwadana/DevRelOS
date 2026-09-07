import { serverFetch } from "@/lib/server-api";
import type { AuditEvent, IdentityAPIKey, IdentityMembership, IdentityUser } from "@/lib/identity-types";

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
  const [users, memberships, audit] = await Promise.all([
    getJSON<IdentityUser[]>("/api/v1/identity/users"),
    getJSON<IdentityMembership[]>("/api/v1/identity/memberships"),
    getJSON<AuditEvent[]>("/api/v1/audit-events?limit=100")
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
    apiKeys: new Map(keyEntries),
    connected: users !== null && memberships !== null && audit !== null
  };
}
