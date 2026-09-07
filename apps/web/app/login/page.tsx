import { LoginForm } from "@/components/login-form";
import { getPrincipal } from "@/lib/identity-api";
import { redirect } from "next/navigation";

export default async function LoginPage() {
  const principal = await getPrincipal();
  if (principal?.user) redirect("/");

  return (
    <div className="identity-page">
      <div className="identity-hero">
        <span className="eyebrow">DevRelOS Identity</span>
        <h1>Sign in to your workspace.</h1>
        <p>Use the owner account bootstrapped by your deployment or a workspace account created by an administrator.</p>
      </div>
      <LoginForm />
      {principal?.system && <div className="notice"><strong>System operator mode is active.</strong><span>The configured API operator token still has break-glass access. User sessions are recommended for day-to-day operation.</span></div>}
    </div>
  );
}
