import { NextRequest } from "next/server";

const backendURL = process.env.DEVRELOS_API_URL ?? "http://localhost:8080";
const sessionCookie = "devrelos_session";

type RouteContext = { params: Promise<{ path: string[] }> };

function requestToken(request: NextRequest): string {
  const sessionsEnabled = (process.env.DEVRELOS_WEB_SESSIONS ?? "").toLowerCase() === "true";
  if (sessionsEnabled) {
    const sessionToken = request.cookies.get(sessionCookie)?.value?.trim() ?? "";
    return sessionToken.startsWith("ds_") ? sessionToken : "";
  }
  return process.env.DEVRELOS_API_TOKEN?.trim() ?? "";
}

function sameOriginMutation(request: NextRequest): boolean {
  if (request.method === "GET" || request.method === "HEAD" || request.method === "OPTIONS") return true;
  const origin = request.headers.get("origin");
  if (!origin) return true;
  try {
    return new URL(origin).origin === request.nextUrl.origin;
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

  const token = requestToken(request);
  if (!token) {
    return Response.json({ error: "authentication required" }, { status: 401, headers: { "Cache-Control": "no-store" } });
  }
  headers.set("Authorization", `Bearer ${token}`);

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
