import { serverFetch } from "@/lib/server-api";
import type { AuditEvent, IdentityMembership, IdentityUser } from "@/lib/identity-types";

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

  return {
    users: users ?? [],
    memberships: memberships ?? [],
    audit: audit ?? [],
    connected: users !== null && memberships !== null && audit !== null
  };
}
