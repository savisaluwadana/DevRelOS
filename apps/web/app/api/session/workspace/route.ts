import { NextRequest, NextResponse } from "next/server";

const backendURL = process.env.DEVRELOS_API_URL ?? "http://localhost:8080";
const cookieName = "devrelos_session";

export async function POST(request: NextRequest) {
  const token = request.cookies.get(cookieName)?.value?.trim() ?? "";
  if (!token.startsWith("ds_")) return NextResponse.json({ error: "authentication required" }, { status: 401 });

  let body: { workspaceId?: string } = {};
  try { body = await request.json(); } catch { return NextResponse.json({ error: "invalid request" }, { status: 400 }); }
  const workspaceId = body.workspaceId?.trim() ?? "";
  if (!workspaceId) return NextResponse.json({ error: "workspaceId is required" }, { status: 400 });

  try {
    const upstream = await fetch(`${backendURL}/api/v1/identity/sessions/current/workspace`, {
      method: "PUT",
      headers: { Authorization: `Bearer ${token}`, "Content-Type": "application/json", Accept: "application/json" },
      body: JSON.stringify({ workspaceId }),
      cache: "no-store",
      signal: AbortSignal.timeout(5000)
    });
    const result = await upstream.json().catch(() => ({}));
    return NextResponse.json(result, { status: upstream.status, headers: { "Cache-Control": "no-store" } });
  } catch {
    return NextResponse.json({ error: "DevRelOS API unavailable" }, { status: 502 });
  }
}
