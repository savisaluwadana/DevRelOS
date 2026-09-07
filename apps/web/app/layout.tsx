import type { Metadata } from "next";
import "./globals.css";
import "./manage.css";
import "./signals.css";

export const metadata: Metadata = {
  title: "DevRelOS",
  description: "Developer Relations operating system"
};

const nav = [
  { label: "Command Center", href: "/" },
  { label: "Signals", href: "/signals" },
  { label: "Events & CFPs", href: "/manage#cfps" },
  { label: "Communities", href: "/manage#communities" },
  { label: "Talk Library", href: "/manage#talks" },
  { label: "Outreach", href: "/#outreach" },
  { label: "Content", href: "/#content" },
  { label: "Feedback", href: "/#feedback" },
  { label: "Analytics", href: "/#analytics" },
  { label: "Automations", href: "/#automations" },
  { label: "Integrations", href: "/integrations" }
];

export default function RootLayout({ children }: Readonly<{ children: React.ReactNode }>) {
  return (
    <html lang="en">
      <body>
        <div className="app-shell">
          <aside className="sidebar">
            <a className="brand" href="/">
              <div className="brand-mark">DR</div>
              <div>
                <strong>DevRelOS</strong>
                <span>Operator Console</span>
              </div>
            </a>
            <nav className="nav-list" aria-label="Primary">
              {nav.map((item, index) => (
                <a className={index === 0 ? "nav-item active" : "nav-item"} href={item.href} key={item.label}>
                  <span className="nav-dot" />
                  {item.label}
                </a>
              ))}
            </nav>
            <div className="sidebar-footer">
              <span className="status-dot" />
              Local workspace
            </div>
          </aside>
          <main className="main-panel">{children}</main>
        </div>
      </body>
    </html>
  );
}
