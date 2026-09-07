import "server-only";

import { cookies } from "next/headers";

const backendURL = process.env.DEVRELOS_API_URL ?? "http://localhost:8080";
const sessionCookieName = "devrelos_session";

export async function serverFetch(path: string, init: RequestInit = {}): Promise<Response> {
  const headers = new Headers(init.headers);
  const cookieStore = await cookies();
  const session = cookieStore.get(sessionCookieName)?.value;
  if (session) {
    headers.set("Cookie", `${sessionCookieName}=${encodeURIComponent(session)}`);
  } else {
    const token = process.env.DEVRELOS_API_TOKEN?.trim();
    if (token) headers.set("Authorization", `Bearer ${token}`);
  }
  if (!headers.has("Accept")) {
    headers.set("Accept", "application/json");
  }

  return fetch(`${backendURL}${path}`, {
    ...init,
    headers,
    cache: init.cache ?? "no-store"
  });
}
