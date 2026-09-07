import { NextRequest } from "next/server";

const backendURL = process.env.DEVRELOS_API_URL ?? "http://localhost:8080";
const sessionCookie = "devrelos_user_token";

type RouteContext = { params: Promise<{ path: string[] }> };

function requestToken(request: NextRequest): string {
  const sessionsEnabled = (process.env.DEVRELOS_WEB_SESSIONS ?? "").toLowerCase() === "true";
  if (sessionsEnabled) {
    const sessionToken = request.cookies.get(sessionCookie)?.value?.trim() ?? "";
    return sessionToken.startsWith("drk_") ? sessionToken : "";
  }
  return process.env.DEVRELOS_API_TOKEN?.trim() ?? "";
}

async function forward(request: NextRequest, context: RouteContext) {
  const { path } = await context.params;
  const pathname = `/${path.map(encodeURIComponent).join("/")}`;
  const target = new URL(`${backendURL}${pathname}`);
  request.nextUrl.searchParams.forEach((value, key) => target.searchParams.append(key, value));

  const headers = new Headers();
  const contentType = request.headers.get("content-type");
  const accept = request.headers.get("accept");
  if (contentType) headers.set("Content-Type", contentType);
  if (accept) headers.set("Accept", accept);

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
  if (request.method !== "GET" && request.method !== "HEAD") {
    init.body = await request.arrayBuffer();
  }

  try {
    const upstream = await fetch(target, init);
    const responseHeaders = new Headers();
    const upstreamType = upstream.headers.get("content-type");
    if (upstreamType) responseHeaders.set("Content-Type", upstreamType);
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
