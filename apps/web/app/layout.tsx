import type { Metadata } from "next";
import { headers } from "next/headers";
import { PrimaryNav } from "@/components/primary-nav";
import { SessionLogout } from "@/components/session-login";
import { WorkspaceSwitcher } from "@/components/workspace-switcher";
import "./globals.css";
import "./manage.css";
import "./signals.css";
import "./outreach.css";
import "./opportunities.css";
import "./work.css";
import "./content.css";
import "./feedback.css";
import "./feedback-github.css";
import "./media.css";
import "./access.css";
import "./campaigns.css";
import "./calendar.css";

export const metadata: Metadata = {
  title: "DevRelOS",
  description: "Developer Relations operating system"
};

const nav = [
  { label: "Command Center", href: "/" },
  { label: "Calendar", href: "/calendar" },
  { label: "Signals", href: "/signals" },
  { label: "Opportunities", href: "/opportunities" },
  { label: "Campaigns & Attribution", href: "/campaigns" },
  { label: "Relationship Radar", href: "/relationships" },
  { label: "Action Queue", href: "/work" },
  { label: "Content Studio", href: "/content" },
  { label: "Media Studio", href: "/media" },
  { label: "Events & CFPs", href: "/manage#cfps" },
  { label: "Communities", href: "/manage#communities" },
  { label: "Talk Library", href: "/manage#talks" },
  { label: "Outreach", href: "/outreach" },
  { label: "Feedback", href: "/feedback" },
  { label: "Integrations", href: "/integrations" },
  { label: "Access & Security", href: "/access", adminOnly: true }
];

export default async function RootLayout({ children }: Readonly<{ children: React.ReactNode }>) {
  const requestHeaders = await headers();
  const loginPage = requestHeaders.get("x-devrelos-login-page") === "1";
  const sessionsEnabled = (process.env.DEVRELOS_WEB_SESSIONS ?? "").toLowerCase() === "true";
  const role = requestHeaders.get("x-devrelos-user-role") ?? "";
  const email = requestHeaders.get("x-devrelos-user-email") ?? "";
  const displayName = requestHeaders.get("x-devrelos-user-display-name") ?? "";
  const workspaceId = requestHeaders.get("x-devrelos-workspace-id") ?? "";
  const canAdmin = !sessionsEnabled || role === "owner" || role === "admin";
  const visibleNav = nav.filter((item) => !item.adminOnly || canAdmin).map(({ label, href }) => ({ label, href }));

  if (loginPage) {
    return <html lang="en"><body><main className="login-shell">{children}</main></body></html>;
  }

  return (
    <html lang="en">
      <body>
        <div className="app-shell">
          <aside className="sidebar">
            <a className="brand" href="/">
              <div className="brand-mark">DR</div>
              <div><strong>DevRelOS</strong><span>Operator Console</span></div>
            </a>
            <PrimaryNav items={visibleNav} />
            <div className="sidebar-footer session-footer">
              {sessionsEnabled && workspaceId ? <WorkspaceSwitcher currentWorkspaceId={workspaceId} /> : null}
              <div className="session-identity">
                <span className="status-dot" />
                <div>
                  <strong>{sessionsEnabled ? (displayName || email || "Signed in") : "Operator mode"}</strong>
                  <span>{sessionsEnabled ? role : "Local workspace"}</span>
                </div>
              </div>
              {sessionsEnabled ? <SessionLogout /> : null}
            </div>
          </aside>
          <main className="main-panel">{children}</main>
        </div>
      </body>
    </html>
  );
}
