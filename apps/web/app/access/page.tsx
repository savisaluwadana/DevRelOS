import { APIKeyForm, MembershipForm, RevokeKeyButton, UserForm } from "@/components/access-actions";
import { formatDateTime } from "@/lib/api";
import { getAccessData } from "@/lib/identity-api";

export default async function AccessPage() {
  const data = await getAccessData();

  return (
    <div className="workspace-page access-page">
      <div className="page-heading">
        <div><span className="eyebrow">Security</span><h1>Access &amp; identity</h1><p>Bootstrap users, workspace roles, revocable API keys and audit visibility.</p></div>
      </div>

      {!data.connected ? <div className="notice">Identity API is not available yet. Ensure migration 000008 is applied and the API is running.</div> : null}

      <section className="access-grid">
        <article className="access-panel"><h2>Create user</h2><UserForm /></article>
        <article className="access-panel"><h2>Workspace membership</h2><MembershipForm users={data.users} /></article>
        <article className="access-panel"><h2>API key</h2><APIKeyForm users={data.users} /></article>
      </section>

      <section className="access-panel">
        <div className="section-heading"><div><span className="eyebrow">Members</span><h2>Workspace access</h2></div></div>
        <div className="table-wrap"><table><thead><tr><th>User</th><th>Email</th><th>Role</th><th>Status</th></tr></thead><tbody>
          {data.memberships.map((item) => <tr key={`${item.workspaceId}:${item.userId}`}><td>{item.displayName || "—"}</td><td>{item.email}</td><td><span className="role-pill">{item.role}</span></td><td>active</td></tr>)}
          {data.memberships.length === 0 ? <tr><td colSpan={4}>No workspace memberships yet.</td></tr> : null}
        </tbody></table></div>
      </section>

      <section className="access-panel">
        <div className="section-heading"><div><span className="eyebrow">Credentials</span><h2>API keys</h2></div></div>
        <div className="key-list">
          {data.users.flatMap((user) => (data.apiKeys.get(user.id) ?? []).map((key) => (
            <div className="key-row" key={key.id}>
              <div><strong>{key.name}</strong><span>{user.email} · {key.keyPrefix}…</span></div>
              <div><span>{key.revokedAt ? "Revoked" : key.lastUsedAt ? `Used ${formatDateTime(key.lastUsedAt)}` : "Never used"}</span>{!key.revokedAt ? <RevokeKeyButton userId={user.id} keyId={key.id} /> : null}</div>
            </div>
          )))}
          {data.users.every((user) => (data.apiKeys.get(user.id) ?? []).length === 0) ? <p>No API keys yet.</p> : null}
        </div>
      </section>

      <section className="access-panel">
        <div className="section-heading"><div><span className="eyebrow">Audit</span><h2>Recent security events</h2></div></div>
        <div className="audit-list">
          {data.audit.map((event) => <div className="audit-row" key={event.id}><div><strong>{event.action}</strong><span>{event.actorKind}{event.resourceType ? ` · ${event.resourceType}` : ""}</span></div><time>{formatDateTime(event.createdAt)}</time></div>)}
          {data.audit.length === 0 ? <p>No audit events yet.</p> : null}
        </div>
      </section>
    </div>
  );
}
