import { NextResponse } from "next/server";

function secureCookie() {
  return (process.env.DEVRELOS_SESSION_SECURE_COOKIE ?? "").toLowerCase() === "true";
}

export async function POST() {
  const response = NextResponse.json({ ok: true });
  response.cookies.set("devrelos_user_token", "", {
    httpOnly: true,
    sameSite: "strict",
    secure: secureCookie(),
    path: "/",
    maxAge: 0
  });
  response.headers.set("Cache-Control", "no-store");
  return response;
}
