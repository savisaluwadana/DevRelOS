import { NextRequest, NextResponse } from "next/server";

const cookieName = "devrelos_user_token";
const backendURL = process.env.DEVRELOS_API_URL ?? "http://localhost:8080";

export async function POST(request: NextRequest) {
  let body: { token?: string } = {};
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
    upstream = await fetch(`${backendURL}/api/v1/identity/me`, {
      headers: { Authorization: `Bearer ${token}`, Accept: "application/json" },
      cache: "no-store",
      signal: AbortSignal.timeout(5000)
    });
  } catch {
    return NextResponse.json({ error: "DevRelOS API unavailable" }, { status: 502 });
  }

  if (!upstream.ok) {
    return NextResponse.json({ error: "API key is invalid, expired, revoked, or has no workspace access" }, { status: 401 });
  }

  const principal = await upstream.json();
  const configuredMaxAge = Number.parseInt(process.env.DEVRELOS_SESSION_MAX_AGE_SECONDS ?? "28800", 10);
  const maxAge = Math.min(86400, Math.max(900, Number.isFinite(configuredMaxAge) ? configuredMaxAge : 28800));
  const response = NextResponse.json({ ok: true, principal });
  response.cookies.set(cookieName, token, {
    httpOnly: true,
    sameSite: "strict",
    secure: process.env.NODE_ENV === "production",
    path: "/",
    maxAge
  });
  response.headers.set("Cache-Control", "no-store");
  return response;
}
