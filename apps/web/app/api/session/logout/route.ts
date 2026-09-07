import { NextRequest, NextResponse } from "next/server";

const backendURL = process.env.DEVRELOS_API_URL ?? "http://localhost:8080";
const cookieName = "devrelos_session";

function secureCookie() {
  return (process.env.DEVRELOS_SESSION_SECURE_COOKIE ?? "").toLowerCase() === "true";
}

export async function POST(request: NextRequest) {
  const token = request.cookies.get(cookieName)?.value?.trim() ?? "";
  if (token.startsWith("ds_")) {
    try {
      await fetch(`${backendURL}/api/v1/identity/sessions/current`, {
        method: "DELETE",
        headers: { Authorization: `Bearer ${token}` },
        cache: "no-store",
        signal: AbortSignal.timeout(5000)
      });
    } catch {
      // Cookie revocation is still performed; the server-side session expires independently.
    }
  }
  const response = NextResponse.json({ ok: true });
  response.cookies.set(cookieName, "", {
    httpOnly: true,
    sameSite: "strict",
    secure: secureCookie(),
    path: "/",
    maxAge: 0
  });
  response.headers.set("Cache-Control", "no-store");
  return response;
}
