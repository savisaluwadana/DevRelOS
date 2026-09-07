import { NextRequest, NextResponse } from "next/server";

const backendURL = process.env.DEVRELOS_API_URL ?? "http://localhost:8080";
const cookieName = "devrelos_session";

function secureCookie() {
  return (process.env.DEVRELOS_SESSION_SECURE_COOKIE ?? "").toLowerCase() === "true";
}

export async function POST(request: NextRequest) {
  let body: { token?: string; displayName?: string } = {};
  try { body = await request.json(); } catch { return NextResponse.json({ error: "invalid request" }, { status: 400 }); }
  const inviteToken = body.token?.trim() ?? "";
  if (!inviteToken.startsWith("di_")) return NextResponse.json({ error: "valid invitation token required" }, { status: 400 });

  try {
    const upstream = await fetch(`${backendURL}/api/v1/identity/invitations/accept`, {
      method: "POST",
      headers: { "Content-Type": "application/json", Accept: "application/json" },
      body: JSON.stringify({ token: inviteToken, displayName: body.displayName?.trim() ?? "" }),
      cache: "no-store",
      signal: AbortSignal.timeout(5000)
    });
    const result = await upstream.json().catch(() => ({}));
    if (!upstream.ok) return NextResponse.json({ error: result?.error ?? "Invitation could not be accepted" }, { status: upstream.status });
    const sessionToken = String(result?.sessionToken ?? "");
    if (!sessionToken.startsWith("ds_")) return NextResponse.json({ error: "DevRelOS did not issue a browser session" }, { status: 502 });
    const configuredMaxAge = Number.parseInt(process.env.DEVRELOS_SESSION_MAX_AGE_SECONDS ?? "28800", 10);
    const maxAge = Math.min(86400, Math.max(900, Number.isFinite(configuredMaxAge) ? configuredMaxAge : 28800));
    const response = NextResponse.json({ ok: true, principal: result.principal, expiresAt: result.expiresAt });
    response.cookies.set(cookieName, sessionToken, { httpOnly: true, sameSite: "strict", secure: secureCookie(), path: "/", maxAge });
    response.headers.set("Cache-Control", "no-store");
    return response;
  } catch {
    return NextResponse.json({ error: "DevRelOS API unavailable" }, { status: 502 });
  }
}
