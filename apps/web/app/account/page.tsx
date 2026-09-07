import { LogoutButton } from "@/components/login-form";
import { getPrincipal, roleLabels } from "@/lib/identity-api";

export default async function AccountPage() {
  const principal = await getPrincipal();

  return (
    <div className="page-wrap identity-page">
      <header className="topbar compact-topbar">
        <div><span className="eyebrow">Account</span><h1>Identity and workspace access.</h1><p>See which workspaces this session can access and the role attached to each membership.</p></div>
        {principal?.user && <LogoutButton />}
      </header>

      {!principal && <div className="notice"><strong>You are not signed in.</strong><span><a href="/login">Open the sign-in page →</a></span></div>}

      {principal?.system && !principal.user && (
        <section className="panel"><div className="panel-head"><div><span className="eyebrow">System principal</span><h2>Operator credential</h2></div></div><p>The web server is currently using the deployment-level API token. This principal bypasses workspace RBAC and should be reserved for bootstrap, automation and break-glass administration.</p><a className="button ghost" href="/login">Sign in as a user</a></section>
      )}

      {principal?.user && (
        <>
          <section className="panel identity-card">
            <span className="eyebrow">Signed in</span>
            <h2>{principal.user.displayName || principal.user.email}</h2>
            <p>{principal.user.email}</p>
          </section>
          <section className="panel">
            <div className="panel-head"><div><span className="eyebrow">Memberships</span><h2>Workspace roles</h2></div></div>
            <div className="identity-membership-list">
              {principal.memberships.map((membership) => (
                <article className="identity-member-row" key={membership.workspaceId}>
                  <div><strong>{membership.workspaceName}</strong><span className="muted">{membership.workspaceSlug}</span></div>
                  <span className="pill neutral">{roleLabels[membership.role]}</span>
                </article>
              ))}
              {principal.memberships.length === 0 && <p className="empty-copy">This user is not a member of any workspace.</p>}
            </div>
          </section>
        </>
      )}
    </div>
  );
}
