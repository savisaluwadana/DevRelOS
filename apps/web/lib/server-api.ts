import "server-only";

const backendURL = process.env.DEVRELOS_API_URL ?? "http://localhost:8080";

export async function serverFetch(path: string, init: RequestInit = {}): Promise<Response> {
  const headers = new Headers(init.headers);
  const token = process.env.DEVRELOS_API_TOKEN?.trim();
  if (token) {
    headers.set("Authorization", `Bearer ${token}`);
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
