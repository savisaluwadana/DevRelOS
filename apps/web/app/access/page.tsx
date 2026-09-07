import {
  APIKeyForm,
  ConnectorSecretControl,
  DeleteSecretButton,
  InvitationForm,
  MembershipForm,
  RevokeInvitationButton,
  RevokeKeyButton,
  RotateSecretButton,
  SecretForm,
  UserForm
} from "@/components/access-actions";
import { formatDateTime } from "@/lib/api";
import { getAccessData } from "@/lib/identity-api";

export default async function AccessPage() {
  const data = await getAccessData();

  return (
    <div className="workspace-page access-page">
      <div className="page-heading">
        <div><span className="eyebrow">Security</span><h1>Access &amp; identity</h1><p>Provision workspace users, manage roles, invitations and credentials, and inspect security activity.</p></div>
      </div>

      {!data.connected ? <div className="notice">Security APIs are not fully available. Ensure migrations 000008–000010 are applied and the API is running.</div> : null}

      <section className="access-grid">
        <article className="access-panel"><h2>Invite teammate</h2><InvitationForm /></article>
        <article className="access-panel"><h2>Provision user</h2><UserForm /></article>
        <article className="access-panel"><h2>Workspace membership</h2><MembershipForm users={data.users} /></article>
        <article className="access-panel"><h2>API key</h2><APIKeyForm users={data.users} /></article>
      </section>

      <section className="access-panel">
        <div className="section-heading"><div><span className="eyebrow">Invitations</span><h2>Pending workspace access</h2></div></div>
        <div className="table-wrap"><table><thead><tr><th>Email</th><th>Role</th><th>Expires</th><th>Status</th><th /></tr></thead><tbody>
          {data.invitations.map((item) => <tr key={item.id}><td>{item.email}</td><td><span className="role-pill">{item.role}</span></td><td>{formatDateTime(item.expiresAt)}</td><td>{item.acceptedAt ? "accepted" : "pending"}</td><td>{!item.acceptedAt ? <RevokeInvitationButton invitationId={item.id} /> : null}</td></tr>)}
          {data.invitations.length === 0 ? <tr><td colSpan={5}>No active invitations.</td></tr> : null}
        </tbody></table></div>
      </section>

      <section className="access-panel">
        <div className="section-heading"><div><span className="eyebrow">Members</span><h2>Workspace access</h2></div></div>
        <div className="table-wrap"><table><thead><tr><th>User</th><th>Email</th><th>Role</th><th>Status</th></tr></thead><tbody>
          {data.memberships.map((item) => <tr key={`${item.workspaceId}:${item.userId}`}><td>{item.displayName || "—"}</td><td>{item.email}</td><td><span className="role-pill">{item.role}</span></td><td>active</td></tr>)}
          {data.memberships.length === 0 ? <tr><td colSpan={4}>No workspace memberships yet.</td></tr> : null}
        </tbody></table></div>
      </section>

      <section className="access-grid">
        <article className="access-panel">
          <div className="section-heading"><div><span className="eyebrow">Encrypted credentials</span><h2>Add connector secret</h2></div></div>
          <p>Secret plaintext is encrypted before storage and is never returned by list APIs.</p>
          <SecretForm />
        </article>
        <article className="access-panel">
          <div className="section-heading"><div><span className="eyebrow">Credential binding</span><h2>Attach to connectors</h2></div></div>
          <div className="key-list">
            {data.connectors.map((connector) => <div className="key-row" key={connector.id}><div><strong>{connector.name}</strong><span>{connector.provider}</span></div><ConnectorSecretControl connector={connector} secrets={data.secrets} /></div>)}
            {data.connectors.length === 0 ? <p>No connectors configured.</p> : null}
          </div>
        </article>
      </section>

      <section className="access-panel">
        <div className="section-heading"><div><span className="eyebrow">Secret inventory</span><h2>{data.secrets.length} encrypted connector secret{data.secrets.length === 1 ? "" : "s"}</h2></div></div>
        <div className="key-list">
          {data.secrets.map((secret) => (
            <div className="key-row secret-row" key={secret.id}>
              <div><strong>{secret.name}</strong><span>{secret.provider} · key version {secret.keyVersion}{secret.rotatedAt ? ` · rotated ${formatDateTime(secret.rotatedAt)}` : ""}</span></div>
              <div><RotateSecretButton secretId={secret.id} /><DeleteSecretButton secretId={secret.id} /></div>
            </div>
          ))}
          {data.secrets.length === 0 ? <p>No encrypted connector secrets yet.</p> : null}
        </div>
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
          {data.audit.map((event) => <div className="audit-row" key={event.id}><div><strong>{event.action}</strong><span>{event.actorEmail || event.actorKind}{event.resourceType ? ` · ${event.resourceType}` : ""}</span></div><time>{formatDateTime(event.createdAt)}</time></div>)}
          {data.audit.length === 0 ? <p>No audit events yet.</p> : null}
        </div>
      </section>
    </div>
  );
}
