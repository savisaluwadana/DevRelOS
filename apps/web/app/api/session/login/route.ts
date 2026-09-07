import { NextRequest, NextResponse } from "next/server";

const cookieName = "devrelos_session";
const backendURL = process.env.DEVRELOS_API_URL ?? "http://localhost:8080";

function secureCookie() {
  return (process.env.DEVRELOS_SESSION_SECURE_COOKIE ?? "").toLowerCase() === "true";
}

export async function POST(request: NextRequest) {
  let body: { token?: string; workspaceId?: string } = {};
  try {
    body = await request.json();
  } catch {
    return NextResponse.json({ error: "invalid request" }, { status: 400 });
  }

  const token = body.token?.trim() ?? "";
  if (!token.startsWith("drk_") || token.length < 20) {
    return NextResponse.json({ error: "valid DevRelOS API key required" }, { status: 400 });
  }

  let upstream: Response;
  try {
    upstream = await fetch(`${backendURL}/api/v1/identity/sessions`, {
      method: "POST",
      headers: {
        Authorization: `Bearer ${token}`,
        Accept: "application/json",
        "Content-Type": "application/json"
      },
      body: JSON.stringify({ workspaceId: body.workspaceId?.trim() ?? "" }),
      cache: "no-store",
      signal: AbortSignal.timeout(5000)
    });
  } catch {
    return NextResponse.json({ error: "DevRelOS API unavailable" }, { status: 502 });
  }

  const result = await upstream.json().catch(() => ({}));
  if (!upstream.ok) {
    return NextResponse.json({ error: result?.error ?? "API key is invalid, expired, revoked, or has no workspace access" }, { status: upstream.status });
  }

  const sessionToken = String(result?.sessionToken ?? "");
  if (!sessionToken.startsWith("ds_")) {
    return NextResponse.json({ error: "DevRelOS did not issue a browser session" }, { status: 502 });
  }
  const configuredMaxAge = Number.parseInt(process.env.DEVRELOS_SESSION_MAX_AGE_SECONDS ?? "28800", 10);
  const maxAge = Math.min(86400, Math.max(900, Number.isFinite(configuredMaxAge) ? configuredMaxAge : 28800));
  const response = NextResponse.json({ ok: true, principal: result.principal, expiresAt: result.expiresAt });
  response.cookies.set(cookieName, sessionToken, {
    httpOnly: true,
    sameSite: "strict",
    secure: secureCookie(),
    path: "/",
    maxAge
  });
  response.headers.set("Cache-Control", "no-store");
  return response;
}
