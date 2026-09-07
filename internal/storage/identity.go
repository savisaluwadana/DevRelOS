package storage

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	identitydomain "github.com/savisaluwadana/DevRelOS/internal/domain/identity"
)

func (s *Store) UserByEmail(ctx context.Context, email string) (identitydomain.User, error) {
	var user identitydomain.User
	err := s.pool.QueryRow(ctx, `
		SELECT id::text, email, display_name, password_hash, status, created_at, updated_at
		FROM users
		WHERE lower(email)=lower($1)
		LIMIT 1`, strings.TrimSpace(email)).
		Scan(&user.ID, &user.Email, &user.DisplayName, &user.PasswordHash, &user.Status, &user.CreatedAt, &user.UpdatedAt)
	return user, err
}

func (s *Store) UserByID(ctx context.Context, userID string) (identitydomain.User, error) {
	var user identitydomain.User
	err := s.pool.QueryRow(ctx, `
		SELECT id::text, email, display_name, password_hash, status, created_at, updated_at
		FROM users WHERE id=$1`, userID).
		Scan(&user.ID, &user.Email, &user.DisplayName, &user.PasswordHash, &user.Status, &user.CreatedAt, &user.UpdatedAt)
	return user, err
}

func (s *Store) CreateUserWithMembership(ctx context.Context, user identitydomain.User, workspaceID, role string) (identitydomain.User, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return user, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	err = tx.QueryRow(ctx, `
		INSERT INTO users (email, display_name, password_hash, status)
		VALUES (lower($1),$2,$3,'active')
		RETURNING id::text, email, display_name, password_hash, status, created_at, updated_at`,
		strings.TrimSpace(user.Email), strings.TrimSpace(user.DisplayName), user.PasswordHash).
		Scan(&user.ID, &user.Email, &user.DisplayName, &user.PasswordHash, &user.Status, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return user, err
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO workspace_memberships (workspace_id, user_id, role)
		VALUES ($1,$2,$3)`, workspaceID, user.ID, role); err != nil {
		return user, err
	}
	if err := tx.Commit(ctx); err != nil {
		return user, err
	}
	return user, nil
}

func (s *Store) EnsureMembership(ctx context.Context, workspaceID, userID, role string) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO workspace_memberships (workspace_id, user_id, role)
		VALUES ($1,$2,$3)
		ON CONFLICT (workspace_id, user_id)
		DO UPDATE SET role=EXCLUDED.role, updated_at=now()`, workspaceID, userID, role)
	return err
}

func (s *Store) CreateSession(ctx context.Context, userID, tokenHash string, expiresAt time.Time) (identitydomain.Session, error) {
	var session identitydomain.Session
	session.UserID = userID
	session.ExpiresAt = expiresAt
	err := s.pool.QueryRow(ctx, `
		INSERT INTO user_sessions (user_id, token_hash, expires_at)
		VALUES ($1,$2,$3)
		RETURNING id::text, created_at`, userID, tokenHash, expiresAt).
		Scan(&session.ID, &session.CreatedAt)
	return session, err
}

func (s *Store) UserBySessionHash(ctx context.Context, tokenHash string) (identitydomain.User, error) {
	var user identitydomain.User
	err := s.pool.QueryRow(ctx, `
		SELECT u.id::text, u.email, u.display_name, u.password_hash, u.status, u.created_at, u.updated_at
		FROM user_sessions s
		JOIN users u ON u.id=s.user_id
		WHERE s.token_hash=$1
		  AND s.revoked_at IS NULL
		  AND s.expires_at > now()
		  AND u.status='active'
		LIMIT 1`, tokenHash).
		Scan(&user.ID, &user.Email, &user.DisplayName, &user.PasswordHash, &user.Status, &user.CreatedAt, &user.UpdatedAt)
	if err == nil {
		_, _ = s.pool.Exec(ctx, `UPDATE user_sessions SET last_seen_at=now() WHERE token_hash=$1`, tokenHash)
	}
	return user, err
}

func (s *Store) RevokeSession(ctx context.Context, tokenHash string) error {
	_, err := s.pool.Exec(ctx, `UPDATE user_sessions SET revoked_at=now() WHERE token_hash=$1 AND revoked_at IS NULL`, tokenHash)
	return err
}

func (s *Store) ListMembershipsForUser(ctx context.Context, userID string) ([]identitydomain.Membership, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT m.workspace_id::text, w.slug, w.name, m.user_id::text,
		       u.email, u.display_name, m.role, m.created_at, m.updated_at
		FROM workspace_memberships m
		JOIN workspaces w ON w.id=m.workspace_id
		JOIN users u ON u.id=m.user_id
		WHERE m.user_id=$1
		ORDER BY w.name`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]identitydomain.Membership, 0)
	for rows.Next() {
		var item identitydomain.Membership
		if err := rows.Scan(&item.WorkspaceID, &item.WorkspaceSlug, &item.WorkspaceName, &item.UserID,
			&item.Email, &item.DisplayName, &item.Role, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) ListWorkspaceMembers(ctx context.Context, workspaceID string) ([]identitydomain.Membership, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT m.workspace_id::text, w.slug, w.name, m.user_id::text,
		       u.email, u.display_name, m.role, m.created_at, m.updated_at
		FROM workspace_memberships m
		JOIN workspaces w ON w.id=m.workspace_id
		JOIN users u ON u.id=m.user_id
		WHERE m.workspace_id=$1
		ORDER BY CASE m.role WHEN 'owner' THEN 1 WHEN 'admin' THEN 2 WHEN 'editor' THEN 3 ELSE 4 END, lower(u.email)`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]identitydomain.Membership, 0)
	for rows.Next() {
		var item identitydomain.Membership
		if err := rows.Scan(&item.WorkspaceID, &item.WorkspaceSlug, &item.WorkspaceName, &item.UserID,
			&item.Email, &item.DisplayName, &item.Role, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) MembershipRole(ctx context.Context, userID, workspaceID string) (string, error) {
	var role string
	err := s.pool.QueryRow(ctx, `SELECT role FROM workspace_memberships WHERE user_id=$1 AND workspace_id=$2`, userID, workspaceID).Scan(&role)
	return role, err
}

func (s *Store) UpdateMembershipRole(ctx context.Context, workspaceID, userID, role string) error {
	result, err := s.pool.Exec(ctx, `
		UPDATE workspace_memberships SET role=$3, updated_at=now()
		WHERE workspace_id=$1 AND user_id=$2`, workspaceID, userID, role)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (s *Store) CountWorkspaceOwners(ctx context.Context, workspaceID string) (int, error) {
	var count int
	err := s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM workspace_memberships WHERE workspace_id=$1 AND role='owner'`, workspaceID).Scan(&count)
	return count, err
}

func (s *Store) ProjectWorkspaceID(ctx context.Context, projectID string) (string, error) {
	var workspaceID string
	err := s.pool.QueryRow(ctx, `SELECT workspace_id::text FROM projects WHERE id=$1`, projectID).Scan(&workspaceID)
	return workspaceID, err
}

func IsNotFound(err error) bool {
	return errors.Is(err, pgx.ErrNoRows)
}
