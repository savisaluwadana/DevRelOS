import { NextRequest } from "next/server";

const backendURL = process.env.DEVRELOS_API_URL ?? "http://localhost:8080";
const sessionCookie = "devrelos_session";

type RouteContext = { params: Promise<{ path: string[] }> };

type TokenResolution =
  | { kind: "token"; token: string }
  | { kind: "anonymous" }
  | { kind: "unauthenticated" };

function requestToken(request: NextRequest): TokenResolution {
  const sessionsEnabled = (process.env.DEVRELOS_WEB_SESSIONS ?? "").toLowerCase() === "true";
  if (sessionsEnabled) {
    // Sessions on: a valid session cookie is the only accepted credential.
    const sessionToken = request.cookies.get(sessionCookie)?.value?.trim() ?? "";
    return sessionToken.startsWith("ds_") ? { kind: "token", token: sessionToken } : { kind: "unauthenticated" };
  }

  const operatorToken = process.env.DEVRELOS_API_TOKEN?.trim() ?? "";
  if (operatorToken) return { kind: "token", token: operatorToken };

  // Single-operator local mode: sessions are off and no operator token is set.
  // Forward with no Authorization header and let the API apply its own policy.
  //
  // Returning 401 here used to break the documented quickstart: server-rendered
  // reads worked (serverFetch also omits the header), but every write from the
  // UI failed, so `cp .env.example .env && make up` produced a dashboard where
  // nothing could be created. The API still refuses anonymous requests whenever
  // DEVRELOS_REQUIRE_AUTH=true or an operator token is configured, so this does
  // not widen access - it stops the proxy from rejecting requests the API would
  // have accepted.
  return { kind: "anonymous" };
}

function firstHeaderValue(request: NextRequest, name: string): string {
  return (request.headers.get(name) ?? "").split(",")[0].trim().toLowerCase();
}

// The browser-visible origin of this deployment, reconstructed from the request.
//
// This must not use request.nextUrl.origin: that reports the server's own
// internal origin, not the host the browser actually used. Two ways it broke the
// same-origin check below, which is the CSRF defence for every mutation:
//   - reaching the dashboard on http://127.0.0.1:3000 instead of
//     http://localhost:3000 rejected every write, and
//   - under the Caddy production profile TLS terminates at the proxy, so the
//     browser sends Origin: https://<domain> while the internal origin stays
//     http://localhost:3000 - so every mutation was rejected.
// Caddy forwards X-Forwarded-Proto and X-Forwarded-Host (see deploy/Caddyfile).
function expectedOrigin(request: NextRequest): string {
  const host = firstHeaderValue(request, "x-forwarded-host") || firstHeaderValue(request, "host");
  if (!host) return "";
  const proto = firstHeaderValue(request, "x-forwarded-proto") || "http";
  return `${proto}://${host}`;
}

function sameOriginMutation(request: NextRequest): boolean {
  if (request.method === "GET" || request.method === "HEAD" || request.method === "OPTIONS") return true;
  const origin = request.headers.get("origin");
  // A same-origin form post or a non-browser client may omit Origin entirely.
  if (!origin) return true;
  const expected = expectedOrigin(request);
  if (!expected) return false;
  try {
    return new URL(origin).origin.toLowerCase() === expected;
  } catch {
    return false;
  }
}

async function forward(request: NextRequest, context: RouteContext) {
  if (!sameOriginMutation(request)) {
    return Response.json({ error: "cross-origin mutation rejected" }, { status: 403, headers: { "Cache-Control": "no-store" } });
  }

  const { path } = await context.params;
  const pathname = `/${path.map(encodeURIComponent).join("/")}`;
  const target = new URL(`${backendURL}${pathname}`);
  request.nextUrl.searchParams.forEach((value, key) => target.searchParams.append(key, value));

  const headers = new Headers();
  const contentType = request.headers.get("content-type");
  const accept = request.headers.get("accept");
  const requestID = request.headers.get("x-request-id");
  if (contentType) headers.set("Content-Type", contentType);
  if (accept) headers.set("Accept", accept);
  if (requestID) headers.set("X-Request-ID", requestID);

  const resolved = requestToken(request);
  if (resolved.kind === "unauthenticated") {
    return Response.json({ error: "authentication required" }, { status: 401, headers: { "Cache-Control": "no-store" } });
  }
  if (resolved.kind === "token") {
    headers.set("Authorization", `Bearer ${resolved.token}`);
  }

  const init: RequestInit = {
    method: request.method,
    headers,
    cache: "no-store",
    signal: AbortSignal.timeout(15_000)
  };
  if (request.method !== "GET" && request.method !== "HEAD") init.body = await request.arrayBuffer();

  try {
    const upstream = await fetch(target, init);
    const responseHeaders = new Headers();
    const upstreamType = upstream.headers.get("content-type");
    const upstreamRequestID = upstream.headers.get("x-request-id");
    if (upstreamType) responseHeaders.set("Content-Type", upstreamType);
    if (upstreamRequestID) responseHeaders.set("X-Request-ID", upstreamRequestID);
    responseHeaders.set("Cache-Control", "no-store");
    responseHeaders.set("X-Content-Type-Options", "nosniff");
    responseHeaders.set("Vary", "Cookie");
    return new Response(upstream.body, {
      status: upstream.status,
      statusText: upstream.statusText,
      headers: responseHeaders
    });
  } catch {
    return Response.json({ error: "DevRelOS API unavailable" }, { status: 502, headers: { "Cache-Control": "no-store" } });
  }
}

export const GET = forward;
export const POST = forward;
export const PATCH = forward;
export const PUT = forward;
export const DELETE = forward;
