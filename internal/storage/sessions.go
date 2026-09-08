package storage

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	domain "github.com/savisaluwadana/DevRelOS/internal/domain/identity"
)

func (s *Store) ListUserWorkspaces(ctx context.Context, userID string) ([]domain.WorkspaceAccess, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT w.id::text, w.slug, w.name, m.role
		FROM workspace_memberships m
		JOIN workspaces w ON w.id=m.workspace_id
		JOIN users u ON u.id=m.user_id
		WHERE m.user_id=$1 AND u.status='active'
		ORDER BY CASE m.role WHEN 'owner' THEN 1 WHEN 'admin' THEN 2 WHEN 'editor' THEN 3 ELSE 4 END, w.name`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]domain.WorkspaceAccess, 0)
	for rows.Next() {
		var item domain.WorkspaceAccess
		if err := rows.Scan(&item.WorkspaceID, &item.WorkspaceSlug, &item.WorkspaceName, &item.Role); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) CreateSession(ctx context.Context, userID, workspaceID, tokenHash, userAgent string, expiresAt time.Time) (domain.Session, error) {
	var item domain.Session
	item.UserID = userID
	item.WorkspaceID = workspaceID
	item.ExpiresAt = expiresAt
	err := s.pool.QueryRow(ctx, `
		INSERT INTO user_sessions (user_id, workspace_id, token_hash, expires_at, user_agent)
		SELECT $1::uuid, $2::uuid, $3, $4, $5
		WHERE EXISTS (
		  SELECT 1 FROM workspace_memberships m JOIN users u ON u.id=m.user_id
		  WHERE m.user_id=$1 AND m.workspace_id=$2 AND u.status='active'
		)
		RETURNING id::text, created_at`, userID, workspaceID, tokenHash, expiresAt, strings.TrimSpace(userAgent)).
		Scan(&item.ID, &item.CreatedAt)
	return item, err
}

func (s *Store) ResolveSession(ctx context.Context, tokenHash string) (domain.SessionPrincipal, error) {
	var item domain.SessionPrincipal
	err := s.pool.QueryRow(ctx, `
		SELECT u.id::text, u.email, u.display_name, u.status, '' AS api_key_id,
		       s.id::text, s.workspace_id::text
		FROM user_sessions s
		JOIN users u ON u.id=s.user_id
		JOIN workspace_memberships m ON m.user_id=s.user_id AND m.workspace_id=s.workspace_id
		WHERE s.token_hash=$1
		  AND s.revoked_at IS NULL
		  AND s.expires_at > now()
		  AND u.status='active'`, tokenHash).
		Scan(&item.UserID, &item.Email, &item.DisplayName, &item.Status, &item.APIKeyID, &item.SessionID, &item.WorkspaceID)
	if err != nil {
		return item, err
	}
	_, _ = s.pool.Exec(ctx, `
		UPDATE user_sessions SET last_seen_at=now()
		WHERE id=$1 AND (last_seen_at IS NULL OR last_seen_at < now() - interval '5 minutes')`, item.SessionID)
	return item, nil
}

func (s *Store) SwitchSessionWorkspace(ctx context.Context, sessionID, userID, workspaceID string) error {
	command, err := s.pool.Exec(ctx, `
		UPDATE user_sessions s
		SET workspace_id=$3, last_seen_at=now()
		WHERE s.id=$1 AND s.user_id=$2 AND s.revoked_at IS NULL AND s.expires_at>now()
		  AND EXISTS (SELECT 1 FROM workspace_memberships m WHERE m.user_id=$2 AND m.workspace_id=$3)`,
		sessionID, userID, workspaceID)
	if err != nil {
		return err
	}
	if command.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (s *Store) RevokeSession(ctx context.Context, sessionID, userID string) error {
	command, err := s.pool.Exec(ctx, `
		UPDATE user_sessions SET revoked_at=COALESCE(revoked_at, now())
		WHERE id=$1 AND user_id=$2`, sessionID, userID)
	if err != nil {
		return err
	}
	if command.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (s *Store) RevokeAllUserSessions(ctx context.Context, userID string) error {
	_, err := s.pool.Exec(ctx, `UPDATE user_sessions SET revoked_at=COALESCE(revoked_at, now()) WHERE user_id=$1 AND revoked_at IS NULL`, userID)
	return err
}

func (s *Store) ListInvitations(ctx context.Context, workspaceID string) ([]domain.Invitation, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id::text, workspace_id::text, email, role, COALESCE(invited_by_user_id::text,''), expires_at, accepted_at, created_at
		FROM workspace_invitations
		WHERE workspace_id=$1
		ORDER BY created_at DESC LIMIT 200`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]domain.Invitation, 0)
	for rows.Next() {
		var item domain.Invitation
		if err := rows.Scan(&item.ID, &item.WorkspaceID, &item.Email, &item.Role, &item.InvitedByUserID, &item.ExpiresAt, &item.AcceptedAt, &item.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) CreateInvitation(ctx context.Context, item domain.Invitation, tokenHash string) (domain.Invitation, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return item, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	_, err = tx.Exec(ctx, `DELETE FROM workspace_invitations WHERE workspace_id=$1 AND lower(email)=lower($2) AND accepted_at IS NULL`, item.WorkspaceID, item.Email)
	if err != nil {
		return item, err
	}
	err = tx.QueryRow(ctx, `
		INSERT INTO workspace_invitations (workspace_id, email, role, token_hash, invited_by_user_id, expires_at)
		VALUES ($1,lower($2),$3,$4,NULLIF($5,'')::uuid,$6)
		RETURNING id::text, email, created_at`, item.WorkspaceID, item.Email, item.Role, tokenHash, item.InvitedByUserID, item.ExpiresAt).
		Scan(&item.ID, &item.Email, &item.CreatedAt)
	if err != nil {
		return item, err
	}
	if err := tx.Commit(ctx); err != nil {
		return item, err
	}
	return item, nil
}

func (s *Store) RevokeInvitation(ctx context.Context, workspaceID, invitationID string) error {
	command, err := s.pool.Exec(ctx, `DELETE FROM workspace_invitations WHERE id=$1 AND workspace_id=$2 AND accepted_at IS NULL`, invitationID, workspaceID)
	if err != nil {
		return err
	}
	if command.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (s *Store) AcceptInvitation(ctx context.Context, tokenHash, displayName string) (domain.User, domain.Invitation, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return domain.User{}, domain.Invitation{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var invitation domain.Invitation
	err = tx.QueryRow(ctx, `
		SELECT id::text, workspace_id::text, email, role, COALESCE(invited_by_user_id::text,''), expires_at, accepted_at, created_at
		FROM workspace_invitations
		WHERE token_hash=$1 AND accepted_at IS NULL AND expires_at>now()
		FOR UPDATE`, tokenHash).
		Scan(&invitation.ID, &invitation.WorkspaceID, &invitation.Email, &invitation.Role, &invitation.InvitedByUserID,
			&invitation.ExpiresAt, &invitation.AcceptedAt, &invitation.CreatedAt)
	if err != nil {
		return domain.User{}, invitation, err
	}

	var user domain.User
	err = tx.QueryRow(ctx, `
		SELECT id::text, email, display_name, status, created_at, updated_at
		FROM users WHERE lower(email)=lower($1)`, invitation.Email).
		Scan(&user.ID, &user.Email, &user.DisplayName, &user.Status, &user.CreatedAt, &user.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		user.Status = "active"
		user.Email = strings.ToLower(invitation.Email)
		user.DisplayName = strings.TrimSpace(displayName)
		err = tx.QueryRow(ctx, `
			INSERT INTO users (email, display_name, status) VALUES (lower($1),$2,'active')
			RETURNING id::text, email, display_name, status, created_at, updated_at`, user.Email, user.DisplayName).
			Scan(&user.ID, &user.Email, &user.DisplayName, &user.Status, &user.CreatedAt, &user.UpdatedAt)
	}
	if err != nil {
		return user, invitation, err
	}
	if user.Status != "active" {
		return user, invitation, errors.New("invited user is disabled")
	}
	if user.DisplayName == "" && strings.TrimSpace(displayName) != "" {
		user.DisplayName = strings.TrimSpace(displayName)
		_, _ = tx.Exec(ctx, `UPDATE users SET display_name=$2, updated_at=now() WHERE id=$1`, user.ID, user.DisplayName)
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO workspace_memberships (workspace_id, user_id, role)
		VALUES ($1,$2,$3)
		ON CONFLICT (workspace_id,user_id) DO UPDATE SET role=EXCLUDED.role, updated_at=now()`,
		invitation.WorkspaceID, user.ID, invitation.Role)
	if err != nil {
		return user, invitation, err
	}
	now := time.Now().UTC()
	_, err = tx.Exec(ctx, `UPDATE workspace_invitations SET accepted_at=$2 WHERE id=$1`, invitation.ID, now)
	if err != nil {
		return user, invitation, err
	}
	invitation.AcceptedAt = &now
	if err := tx.Commit(ctx); err != nil {
		return user, invitation, err
	}
	return user, invitation, nil
}
