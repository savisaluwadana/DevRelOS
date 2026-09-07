package storage

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	domain "github.com/savisaluwadana/DevRelOS/internal/domain/identity"
)

func (s *Store) CreateUser(ctx context.Context, user domain.User) (domain.User, error) {
	if user.Status == "" { user.Status = "active" }
	err := s.pool.QueryRow(ctx, `
		INSERT INTO users (email, display_name, status)
		VALUES (lower($1), $2, $3)
		RETURNING id::text, email, display_name, status, created_at, updated_at`,
		user.Email, user.DisplayName, user.Status).
		Scan(&user.ID, &user.Email, &user.DisplayName, &user.Status, &user.CreatedAt, &user.UpdatedAt)
	return user, err
}

func (s *Store) CreateWorkspaceUser(ctx context.Context, workspaceID string, user domain.User, role string) (domain.User, error) {
	if user.Status == "" { user.Status = "active" }
	tx, err := s.pool.Begin(ctx)
	if err != nil { return user, err }
	defer tx.Rollback(ctx)

	err = tx.QueryRow(ctx, `
		INSERT INTO users (email, display_name, status)
		VALUES (lower($1), $2, $3)
		ON CONFLICT DO NOTHING
		RETURNING id::text, email, display_name, status, created_at, updated_at`,
		user.Email, user.DisplayName, user.Status).
		Scan(&user.ID, &user.Email, &user.DisplayName, &user.Status, &user.CreatedAt, &user.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		err = tx.QueryRow(ctx, `
			SELECT id::text, email, display_name, status, created_at, updated_at
			FROM users WHERE lower(email)=lower($1)`, user.Email).
			Scan(&user.ID, &user.Email, &user.DisplayName, &user.Status, &user.CreatedAt, &user.UpdatedAt)
	}
	if err != nil { return user, err }

	if _, err = tx.Exec(ctx, `
		INSERT INTO workspace_memberships (workspace_id, user_id, role)
		VALUES ($1,$2,$3)
		ON CONFLICT (workspace_id, user_id)
		DO UPDATE SET role=EXCLUDED.role, updated_at=now()`, workspaceID, user.ID, role); err != nil {
		return user, err
	}
	if err = tx.Commit(ctx); err != nil { return user, err }
	return user, nil
}

func (s *Store) ListUsers(ctx context.Context) ([]domain.User, error) {
	rows, err := s.pool.Query(ctx, `SELECT id::text, email, display_name, status, created_at, updated_at FROM users ORDER BY created_at DESC`)
	if err != nil { return nil, err }
	defer rows.Close()
	items := make([]domain.User, 0)
	for rows.Next() {
		var item domain.User
		if err := rows.Scan(&item.ID, &item.Email, &item.DisplayName, &item.Status, &item.CreatedAt, &item.UpdatedAt); err != nil { return nil, err }
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) ListWorkspaceUsers(ctx context.Context, workspaceID string) ([]domain.User, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT u.id::text, u.email, u.display_name, u.status, u.created_at, u.updated_at
		FROM users u JOIN workspace_memberships m ON m.user_id=u.id
		WHERE m.workspace_id=$1 ORDER BY u.email`, workspaceID)
	if err != nil { return nil, err }
	defer rows.Close()
	items := make([]domain.User, 0)
	for rows.Next() {
		var item domain.User
		if err := rows.Scan(&item.ID, &item.Email, &item.DisplayName, &item.Status, &item.CreatedAt, &item.UpdatedAt); err != nil { return nil, err }
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) UpsertMembership(ctx context.Context, item domain.Membership) (domain.Membership, error) {
	err := s.pool.QueryRow(ctx, `
		INSERT INTO workspace_memberships (workspace_id, user_id, role)
		VALUES ($1,$2,$3)
		ON CONFLICT (workspace_id, user_id)
		DO UPDATE SET role=EXCLUDED.role, updated_at=now()
		RETURNING created_at, updated_at`, item.WorkspaceID, item.UserID, item.Role).
		Scan(&item.CreatedAt, &item.UpdatedAt)
	return item, err
}

func (s *Store) ListMemberships(ctx context.Context, workspaceID string) ([]domain.Membership, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT m.workspace_id::text, w.name, m.user_id::text, u.email, u.display_name, m.role, m.created_at, m.updated_at
		FROM workspace_memberships m
		JOIN workspaces w ON w.id=m.workspace_id
		JOIN users u ON u.id=m.user_id
		WHERE m.workspace_id=$1
		ORDER BY CASE m.role WHEN 'owner' THEN 1 WHEN 'admin' THEN 2 WHEN 'editor' THEN 3 ELSE 4 END, u.email`, workspaceID)
	if err != nil { return nil, err }
	defer rows.Close()
	items := make([]domain.Membership, 0)
	for rows.Next() {
		var item domain.Membership
		if err := rows.Scan(&item.WorkspaceID, &item.WorkspaceName, &item.UserID, &item.Email, &item.DisplayName, &item.Role, &item.CreatedAt, &item.UpdatedAt); err != nil { return nil, err }
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) RoleForWorkspace(ctx context.Context, userID, workspaceID string) (string, error) {
	var role string
	err := s.pool.QueryRow(ctx, `
		SELECT m.role FROM workspace_memberships m
		JOIN users u ON u.id=m.user_id
		WHERE m.user_id=$1 AND m.workspace_id=$2 AND u.status='active'`, userID, workspaceID).Scan(&role)
	return role, err
}

func (s *Store) UserInWorkspace(ctx context.Context, userID, workspaceID string) (bool, error) {
	var exists bool
	err := s.pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM workspace_memberships m
			JOIN users u ON u.id=m.user_id
			WHERE m.user_id=$1 AND m.workspace_id=$2 AND u.status='active'
		)`, userID, workspaceID).Scan(&exists)
	return exists, err
}

func (s *Store) CreateAPIKey(ctx context.Context, userID, name, prefix, secretHash string, expiresAt *time.Time) (domain.APIKey, error) {
	var item domain.APIKey
	item.UserID, item.Name, item.KeyPrefix, item.ExpiresAt = userID, name, prefix, expiresAt
	err := s.pool.QueryRow(ctx, `
		INSERT INTO api_keys (user_id, name, key_prefix, secret_hash, expires_at)
		VALUES ($1,$2,$3,$4,$5) RETURNING id::text, created_at`, userID, name, prefix, secretHash, expiresAt).
		Scan(&item.ID, &item.CreatedAt)
	return item, err
}

func (s *Store) ListAPIKeys(ctx context.Context, userID string) ([]domain.APIKey, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id::text, user_id::text, name, key_prefix, expires_at, last_used_at, revoked_at, created_at
		FROM api_keys WHERE user_id=$1 ORDER BY created_at DESC`, userID)
	if err != nil { return nil, err }
	defer rows.Close()
	items := make([]domain.APIKey, 0)
	for rows.Next() {
		var item domain.APIKey
		if err := rows.Scan(&item.ID, &item.UserID, &item.Name, &item.KeyPrefix, &item.ExpiresAt, &item.LastUsedAt, &item.RevokedAt, &item.CreatedAt); err != nil { return nil, err }
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) ResolveAPIKey(ctx context.Context, secretHash string) (domain.Principal, error) {
	var principal domain.Principal
	err := s.pool.QueryRow(ctx, `
		SELECT u.id::text, u.email, u.display_name, u.status, k.id::text
		FROM api_keys k JOIN users u ON u.id=k.user_id
		WHERE k.secret_hash=$1 AND k.revoked_at IS NULL
		  AND (k.expires_at IS NULL OR k.expires_at > now()) AND u.status='active'`, secretHash).
		Scan(&principal.UserID, &principal.Email, &principal.DisplayName, &principal.Status, &principal.APIKeyID)
	if err != nil { return principal, err }
	_, _ = s.pool.Exec(ctx, `UPDATE api_keys SET last_used_at=now() WHERE id=$1`, principal.APIKeyID)
	return principal, nil
}

func (s *Store) RevokeAPIKey(ctx context.Context, userID, keyID string) error {
	command, err := s.pool.Exec(ctx, `UPDATE api_keys SET revoked_at=COALESCE(revoked_at, now()) WHERE id=$1 AND user_id=$2`, keyID, userID)
	if err != nil { return err }
	if command.RowsAffected() == 0 { return pgx.ErrNoRows }
	return nil
}

func (s *Store) AppendAuditEvent(ctx context.Context, event domain.AuditEvent) error {
	if event.Metadata == nil { event.Metadata = map[string]any{} }
	metadata, err := json.Marshal(event.Metadata)
	if err != nil { return err }
	_, err = s.pool.Exec(ctx, `
		INSERT INTO audit_events (workspace_id, actor_user_id, actor_kind, action, resource_type, resource_id, metadata)
		VALUES (NULLIF($1,'')::uuid, NULLIF($2,'')::uuid, $3,$4,$5,$6,$7::jsonb)`,
		event.WorkspaceID, event.ActorUserID, event.ActorKind, event.Action, event.ResourceType, event.ResourceID, metadata)
	return err
}

func (s *Store) ListAuditEvents(ctx context.Context, workspaceID string, limit int) ([]domain.AuditEvent, error) {
	if limit < 1 { limit = 100 }
	if limit > 500 { limit = 500 }
	rows, err := s.pool.Query(ctx, `
		SELECT a.id::text, COALESCE(a.workspace_id::text,''), COALESCE(a.actor_user_id::text,''), a.actor_kind,
		       COALESCE(u.email,''), a.action, a.resource_type, a.resource_id, a.metadata, a.created_at
		FROM audit_events a LEFT JOIN users u ON u.id=a.actor_user_id
		WHERE a.workspace_id=$1
		ORDER BY a.created_at DESC LIMIT $2`, workspaceID, limit)
	if err != nil { return nil, err }
	defer rows.Close()
	items := make([]domain.AuditEvent, 0)
	for rows.Next() {
		var item domain.AuditEvent
		var raw []byte
		if err := rows.Scan(&item.ID, &item.WorkspaceID, &item.ActorUserID, &item.ActorKind, &item.ActorEmail,
			&item.Action, &item.ResourceType, &item.ResourceID, &raw, &item.CreatedAt); err != nil { return nil, err }
		item.Metadata = map[string]any{}
		if len(raw) > 0 {
			if err := json.Unmarshal(raw, &item.Metadata); err != nil { return nil, err }
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func IsNotFound(err error) bool { return errors.Is(err, pgx.ErrNoRows) }
