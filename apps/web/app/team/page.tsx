import { MemberForm, MemberRoleControl } from "@/components/team-actions";
import { getMembers, roleLabels } from "@/lib/identity-api";

export default async function TeamPage() {
  const data = await getMembers();

  return (
    <div className="page-wrap identity-page">
      <header className="topbar compact-topbar">
        <div><span className="eyebrow">Workspace Access</span><h1>Team and roles.</h1><p>Owners and admins can add workspace users. Owners control elevated roles; viewers are read-only and editors can execute DevRel workflows.</p></div>
        <a className="button ghost" href="/account">My account</a>
      </header>

      {!data.connected && !data.forbidden && <div className="notice"><strong>Identity API is not connected.</strong><span>Apply migration 000008 and start the updated API.</span></div>}
      {data.forbidden && <div className="notice"><strong>Admin access required.</strong><span>Your current workspace role cannot manage members.</span></div>}

      {!data.forbidden && (
        <>
          <section className="panel">
            <div className="panel-head"><div><span className="eyebrow">Add member</span><h2>Workspace access</h2></div><span className="muted panel-note">New users need an initial password of at least 12 characters. Existing DevRelOS users can be attached without changing their password.</span></div>
            <MemberForm />
          </section>
          <section className="panel">
            <div className="panel-head"><div><span className="eyebrow">Members</span><h2>{data.items.length} workspace users</h2></div></div>
            <div className="identity-membership-list">
              {data.items.map((member) => (
                <article className="identity-member-row identity-team-row" key={member.userId}>
                  <div className="identity-member-copy"><strong>{member.displayName || member.email}</strong><span>{member.email}</span><span className="muted">Current role: {roleLabels[member.role]}</span></div>
                  <MemberRoleControl member={member} />
                </article>
              ))}
              {data.items.length === 0 && <p className="empty-copy">No workspace members found.</p>}
            </div>
          </section>
        </>
      )}
    </div>
  );
}
