package storage

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	domain "github.com/savisaluwadana/DevRelOS/internal/domain/connectors"
)

var ErrSecretInUse = errors.New("secret is still attached to one or more connectors")

func (s *Store) ListConnectorSecrets(ctx context.Context, workspaceID string) ([]domain.Secret, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id::text, workspace_id::text, provider, name, key_version,
		       COALESCE(created_by_user_id::text,''), rotated_at, created_at, updated_at
		FROM connector_secrets
		WHERE workspace_id=$1
		ORDER BY provider, name`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]domain.Secret, 0)
	for rows.Next() {
		var item domain.Secret
		if err := rows.Scan(&item.ID, &item.WorkspaceID, &item.Provider, &item.Name, &item.KeyVersion,
			&item.CreatedByUserID, &item.RotatedAt, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) CreateConnectorSecret(ctx context.Context, item domain.EncryptedSecret) (domain.Secret, error) {
	if item.KeyVersion < 1 {
		item.KeyVersion = 1
	}
	err := s.pool.QueryRow(ctx, `
		INSERT INTO connector_secrets
		  (workspace_id, provider, name, ciphertext, nonce, key_version, created_by_user_id)
		VALUES ($1,$2,$3,$4,$5,$6,NULLIF($7,'')::uuid)
		RETURNING id::text, created_at, updated_at`,
		item.WorkspaceID, item.Provider, item.Name, item.Ciphertext, item.Nonce, item.KeyVersion, item.CreatedByUserID).
		Scan(&item.ID, &item.CreatedAt, &item.UpdatedAt)
	return item.Secret, err
}

func (s *Store) RotateConnectorSecret(ctx context.Context, workspaceID, secretID string, ciphertext, nonce []byte, keyVersion int) (domain.Secret, error) {
	var item domain.Secret
	if keyVersion < 1 {
		keyVersion = 1
	}
	now := time.Now().UTC()
	err := s.pool.QueryRow(ctx, `
		UPDATE connector_secrets
		SET ciphertext=$3, nonce=$4, key_version=$5, rotated_at=$6, updated_at=$6
		WHERE id=$1 AND workspace_id=$2
		RETURNING id::text, workspace_id::text, provider, name, key_version,
		          COALESCE(created_by_user_id::text,''), rotated_at, created_at, updated_at`,
		secretID, workspaceID, ciphertext, nonce, keyVersion, now).
		Scan(&item.ID, &item.WorkspaceID, &item.Provider, &item.Name, &item.KeyVersion,
			&item.CreatedByUserID, &item.RotatedAt, &item.CreatedAt, &item.UpdatedAt)
	return item, err
}

func (s *Store) GetEncryptedConnectorSecret(ctx context.Context, workspaceID, secretID string) (domain.EncryptedSecret, error) {
	var item domain.EncryptedSecret
	err := s.pool.QueryRow(ctx, `
		SELECT id::text, workspace_id::text, provider, name, ciphertext, nonce, key_version,
		       COALESCE(created_by_user_id::text,''), rotated_at, created_at, updated_at
		FROM connector_secrets
		WHERE id=$1 AND workspace_id=$2`, secretID, workspaceID).
		Scan(&item.ID, &item.WorkspaceID, &item.Provider, &item.Name, &item.Ciphertext, &item.Nonce,
			&item.KeyVersion, &item.CreatedByUserID, &item.RotatedAt, &item.CreatedAt, &item.UpdatedAt)
	return item, err
}

func (s *Store) DeleteConnectorSecret(ctx context.Context, workspaceID, secretID string) error {
	var references int
	if err := s.pool.QueryRow(ctx, `SELECT count(*) FROM connectors WHERE workspace_id=$1 AND secret_id=$2`, workspaceID, secretID).Scan(&references); err != nil {
		return err
	}
	if references > 0 {
		return ErrSecretInUse
	}
	command, err := s.pool.Exec(ctx, `DELETE FROM connector_secrets WHERE id=$1 AND workspace_id=$2`, secretID, workspaceID)
	if err != nil {
		return err
	}
	if command.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (s *Store) ConnectorSecretBelongs(ctx context.Context, workspaceID, secretID, provider string) (bool, error) {
	if secretID == "" {
		return true, nil
	}
	var exists bool
	err := s.pool.QueryRow(ctx, `
		SELECT EXISTS(
		  SELECT 1 FROM connector_secrets
		  WHERE id=$1 AND workspace_id=$2 AND (provider=$3 OR provider='generic')
		)`, secretID, workspaceID, provider).Scan(&exists)
	return exists, err
}
