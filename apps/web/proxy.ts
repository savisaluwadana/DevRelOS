import { NextRequest, NextResponse } from "next/server";

const sessionCookie = "devrelos_user_token";

function equalText(a: string, b: string): boolean {
  if (a.length !== b.length) return false;
  let diff = 0;
  for (let i = 0; i < a.length; i += 1) diff |= a.charCodeAt(i) ^ b.charCodeAt(i);
  return diff === 0;
}

function unauthorized() {
  return new NextResponse("Authentication required", {
    status: 401,
    headers: {
      "WWW-Authenticate": 'Basic realm="DevRelOS", charset="UTF-8"',
      "Cache-Control": "no-store",
      "X-Content-Type-Options": "nosniff",
      "X-Frame-Options": "DENY",
      "Referrer-Policy": "no-referrer"
    }
  });
}

function isSessionPublicPath(pathname: string) {
  return pathname === "/login" || pathname.startsWith("/api/session/");
}

async function sessionPrincipal(token: string) {
  const backendURL = process.env.DEVRELOS_API_URL ?? "http://localhost:8080";
  try {
    const response = await fetch(`${backendURL}/api/v1/identity/me`, {
      headers: { Authorization: `Bearer ${token}`, Accept: "application/json" },
      cache: "no-store",
      signal: AbortSignal.timeout(3000)
    });
    if (!response.ok) return null;
    return await response.json();
  } catch {
    return null;
  }
}

export async function proxy(request: NextRequest) {
  const sessionsEnabled = (process.env.DEVRELOS_WEB_SESSIONS ?? "").toLowerCase() === "true";

  if (sessionsEnabled) {
    if (isSessionPublicPath(request.nextUrl.pathname)) return NextResponse.next();

    const token = request.cookies.get(sessionCookie)?.value?.trim() ?? "";
    if (!token.startsWith("drk_")) {
      return NextResponse.redirect(new URL("/login", request.url));
    }

    const principal = await sessionPrincipal(token);
    if (!principal) {
      const response = NextResponse.redirect(new URL("/login", request.url));
      response.cookies.set(sessionCookie, "", { path: "/", maxAge: 0 });
      return response;
    }

    if (request.nextUrl.pathname === "/login") return NextResponse.redirect(new URL("/", request.url));
  } else {
    const username = process.env.DEVRELOS_WEB_USERNAME?.trim() ?? "";
    const password = process.env.DEVRELOS_WEB_PASSWORD ?? "";

    if (username && password) {
      const header = request.headers.get("authorization") ?? "";
      if (!header.startsWith("Basic ")) return unauthorized();

      let decoded = "";
      try {
        decoded = atob(header.slice(6));
      } catch {
        return unauthorized();
      }
      const separator = decoded.indexOf(":");
      if (separator < 0) return unauthorized();
      const suppliedUser = decoded.slice(0, separator);
      const suppliedPassword = decoded.slice(separator + 1);
      if (!equalText(suppliedUser, username) || !equalText(suppliedPassword, password)) return unauthorized();
    }
  }

  const response = NextResponse.next();
  response.headers.set("Cache-Control", "no-store");
  response.headers.set("X-Content-Type-Options", "nosniff");
  response.headers.set("X-Frame-Options", "DENY");
  response.headers.set("Referrer-Policy", "no-referrer");
  return response;
}

export const config = {
  matcher: ["/((?!_next/static|_next/image|favicon.ico).*)"]
};
