import { serverFetch } from "@/lib/server-api";
import type { Membership, Principal } from "@/lib/identity-shared";
export * from "@/lib/identity-shared";

export async function getPrincipal(): Promise<Principal | null> {
  try {
    const response = await serverFetch("/api/v1/auth/me", { signal: AbortSignal.timeout(3000) });
    if (!response.ok) return null;
    return (await response.json()) as Principal;
  } catch {
    return null;
  }
}

export async function getMembers(): Promise<{ items: Membership[]; connected: boolean; forbidden: boolean }> {
  try {
    const response = await serverFetch("/api/v1/members", { signal: AbortSignal.timeout(3000) });
    if (response.status === 403) return { items: [], connected: true, forbidden: true };
    if (!response.ok) return { items: [], connected: false, forbidden: false };
    return { items: (await response.json()) as Membership[], connected: true, forbidden: false };
  } catch {
    return { items: [], connected: false, forbidden: false };
  }
}
