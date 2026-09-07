import type { Metadata } from "next";
import "./globals.css";

export const metadata: Metadata = {
  title: "DevRelOS",
  description: "Developer Relations operating system"
};

const nav = [
  "Command Center",
  "Signals",
  "Events & CFPs",
  "Communities",
  "Talk Library",
  "Outreach",
  "Content",
  "Feedback",
  "Analytics",
  "Automations",
  "Integrations"
];

export default function RootLayout({ children }: Readonly<{ children: React.ReactNode }>) {
  return (
    <html lang="en">
      <body>
        <div className="app-shell">
          <aside className="sidebar">
            <div className="brand">
              <div className="brand-mark">DR</div>
              <div>
                <strong>DevRelOS</strong>
                <span>Operator Console</span>
              </div>
            </div>
            <nav className="nav-list" aria-label="Primary">
              {nav.map((item, index) => (
                <a className={index === 0 ? "nav-item active" : "nav-item"} href="#" key={item}>
                  <span className="nav-dot" />
                  {item}
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
