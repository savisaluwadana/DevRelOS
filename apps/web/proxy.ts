import { NextRequest, NextResponse } from "next/server";

const sessionCookie = "devrelos_session";

type SessionPrincipal = {
  kind?: string;
  userId?: string;
  email?: string;
  displayName?: string;
  workspaceId?: string;
  role?: string;
};

function secureCookie() {
  return (process.env.DEVRELOS_SESSION_SECURE_COOKIE ?? "").toLowerCase() === "true";
}

function equalText(a: string, b: string): boolean {
  if (a.length !== b.length) return false;
  let diff = 0;
  for (let i = 0; i < a.length; i += 1) diff |= a.charCodeAt(i) ^ b.charCodeAt(i);
  return diff === 0;
}

function securityHeaders(response: NextResponse) {
  response.headers.set("Cache-Control", "no-store");
  response.headers.set("X-Content-Type-Options", "nosniff");
  response.headers.set("X-Frame-Options", "DENY");
  response.headers.set("Referrer-Policy", "no-referrer");
  response.headers.set("Permissions-Policy", "camera=(), microphone=(), geolocation=()");
  if (secureCookie()) response.headers.set("Strict-Transport-Security", "max-age=31536000; includeSubDomains");
  return response;
}

function unauthorized() {
  return securityHeaders(new NextResponse("Authentication required", {
    status: 401,
    headers: { "WWW-Authenticate": 'Basic realm="DevRelOS", charset="UTF-8"' }
  }));
}

function isPublicSessionPath(pathname: string) {
  return pathname === "/login" || pathname === "/invite" || pathname.startsWith("/api/session/");
}

async function sessionPrincipal(token: string): Promise<SessionPrincipal | null> {
  const backendURL = process.env.DEVRELOS_API_URL ?? "http://localhost:8080";
  try {
    const response = await fetch(`${backendURL}/api/v1/identity/me`, {
      headers: { Authorization: `Bearer ${token}`, Accept: "application/json" },
      cache: "no-store",
      signal: AbortSignal.timeout(3000)
    });
    if (!response.ok) return null;
    return (await response.json()) as SessionPrincipal;
  } catch {
    return null;
  }
}

function clearAndRedirect(request: NextRequest, pathname = "/login") {
  const response = NextResponse.redirect(new URL(pathname, request.url));
  response.cookies.set(sessionCookie, "", {
    httpOnly: true,
    sameSite: "strict",
    secure: secureCookie(),
    path: "/",
    maxAge: 0
  });
  return securityHeaders(response);
}

function nextWithRequestHeaders(request: NextRequest, values: Record<string, string>) {
  const requestHeaders = new Headers(request.headers);
  for (const [key, value] of Object.entries(values)) requestHeaders.set(key, value);
  return securityHeaders(NextResponse.next({ request: { headers: requestHeaders } }));
}

export async function proxy(request: NextRequest) {
  const sessionsEnabled = (process.env.DEVRELOS_WEB_SESSIONS ?? "").toLowerCase() === "true";
  const pathname = request.nextUrl.pathname;

  if (sessionsEnabled) {
    const token = request.cookies.get(sessionCookie)?.value?.trim() ?? "";
    if (!token.startsWith("ds_")) {
      if (isPublicSessionPath(pathname)) return nextWithRequestHeaders(request, { "x-devrelos-login-page": "1" });
      return clearAndRedirect(request);
    }

    const principal = await sessionPrincipal(token);
    if (!principal || principal.kind !== "user") return clearAndRedirect(request);
    if (pathname === "/login" || pathname === "/invite") return securityHeaders(NextResponse.redirect(new URL("/", request.url)));

    if (pathname === "/access" && principal.role !== "owner" && principal.role !== "admin") {
      return securityHeaders(NextResponse.redirect(new URL("/", request.url)));
    }

    return nextWithRequestHeaders(request, {
      "x-devrelos-user-id": principal.userId ?? "",
      "x-devrelos-user-role": principal.role ?? "",
      "x-devrelos-user-email": principal.email ?? "",
      "x-devrelos-user-display-name": principal.displayName ?? "",
      "x-devrelos-workspace-id": principal.workspaceId ?? ""
    });
  }

  const username = process.env.DEVRELOS_WEB_USERNAME?.trim() ?? "";
  const password = process.env.DEVRELOS_WEB_PASSWORD ?? "";

  if (username && password) {
    const header = request.headers.get("authorization") ?? "";
    if (!header.startsWith("Basic ")) return unauthorized();
    let decoded = "";
    try { decoded = atob(header.slice(6)); } catch { return unauthorized(); }
    const separator = decoded.indexOf(":");
    if (separator < 0) return unauthorized();
    const suppliedUser = decoded.slice(0, separator);
    const suppliedPassword = decoded.slice(separator + 1);
    if (!equalText(suppliedUser, username) || !equalText(suppliedPassword, password)) return unauthorized();
  }

  return securityHeaders(NextResponse.next());
}

export const config = {
  matcher: ["/((?!_next/static|_next/image|favicon.ico).*)"]
};
