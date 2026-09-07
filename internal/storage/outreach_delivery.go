package storage

import (
	"context"
	"errors"
	"time"

	domain "github.com/savisaluwadana/DevRelOS/internal/domain/outreach"
	"github.com/jackc/pgx/v5"
)

func (s *Store) QueueOutreachDelivery(ctx context.Context, projectID, outreachID string) (domain.Delivery, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return domain.Delivery{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var delivery domain.Delivery
	var currentStatus, channel string
	var contactID, contactName, email, subject, body string
	var doNotContact bool
	if err := tx.QueryRow(ctx, `
		SELECT o.status, o.channel, COALESCE(o.contact_id::text,''), COALESCE(c.name,''),
		       COALESCE(c.email,''), COALESCE(c.do_not_contact,false), COALESCE(o.subject,''), o.body
		FROM outreach o
		LEFT JOIN contacts c ON c.id=o.contact_id
		WHERE o.id=$1 AND o.project_id=$2
		FOR UPDATE OF o`, outreachID, projectID).
		Scan(&currentStatus, &channel, &contactID, &contactName, &email, &doNotContact, &subject, &body); err != nil {
		return delivery, err
	}
	if currentStatus != "approved" && currentStatus != "queued" && currentStatus != "failed" {
		return delivery, errors.New("outreach must be approved before delivery can be queued")
	}
	if channel != "email" {
		return delivery, errors.New("automatic delivery currently supports email outreach only")
	}
	if contactID == "" || email == "" {
		return delivery, errors.New("email outreach requires a contact with an email address")
	}
	if doNotContact {
		return delivery, errors.New("contact is marked do-not-contact")
	}

	if _, err := tx.Exec(ctx, `UPDATE outreach SET status='queued', updated_at=now() WHERE id=$1`, outreachID); err != nil {
		return delivery, err
	}

	err = tx.QueryRow(ctx, `
		INSERT INTO outreach_deliveries (outreach_id, transport, status, next_attempt_at)
		VALUES ($1,'smtp','queued',now())
		ON CONFLICT (outreach_id)
		DO UPDATE SET status=CASE WHEN outreach_deliveries.status='sent' THEN 'sent' ELSE 'queued' END,
		              last_error=CASE WHEN outreach_deliveries.status='sent' THEN outreach_deliveries.last_error ELSE NULL END,
		              next_attempt_at=CASE WHEN outreach_deliveries.status='sent' THEN outreach_deliveries.next_attempt_at ELSE now() END,
		              updated_at=now()
		RETURNING id::text, outreach_id::text, transport, status, attempt_count,
		          COALESCE(message_id,''), COALESCE(last_error,''), queued_at, next_attempt_at,
		          sent_at, created_at, updated_at`, outreachID).
		Scan(&delivery.ID, &delivery.OutreachID, &delivery.Transport, &delivery.Status, &delivery.AttemptCount,
			&delivery.MessageID, &delivery.LastError, &delivery.QueuedAt, &delivery.NextAttemptAt,
			&delivery.SentAt, &delivery.CreatedAt, &delivery.UpdatedAt)
	if err != nil {
		return delivery, err
	}
	delivery.ProjectID = projectID
	delivery.RecipientName = contactName
	delivery.RecipientEmail = email
	delivery.Subject = subject
	delivery.Body = body
	if err := tx.Commit(ctx); err != nil {
		return delivery, err
	}
	return delivery, nil
}

func (s *Store) ClaimNextOutreachDelivery(ctx context.Context, maxAttempts int) (*domain.Delivery, error) {
	if maxAttempts < 1 {
		maxAttempts = 5
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var item domain.Delivery
	err = tx.QueryRow(ctx, `
		SELECT d.id::text, d.outreach_id::text, o.project_id::text, d.transport, d.status,
		       d.attempt_count, COALESCE(d.message_id,''), COALESCE(d.last_error,''),
		       COALESCE(c.name,''), COALESCE(c.email,''), COALESCE(o.subject,''), o.body,
		       d.queued_at, d.next_attempt_at, d.sent_at, d.created_at, d.updated_at
		FROM outreach_deliveries d
		JOIN outreach o ON o.id=d.outreach_id
		JOIN contacts c ON c.id=o.contact_id
		WHERE d.status IN ('queued','failed')
		  AND d.next_attempt_at <= now()
		  AND d.attempt_count < $1
		  AND o.status IN ('queued','failed')
		  AND c.do_not_contact=false
		ORDER BY d.next_attempt_at, d.queued_at
		FOR UPDATE OF d SKIP LOCKED
		LIMIT 1`, maxAttempts).
		Scan(&item.ID, &item.OutreachID, &item.ProjectID, &item.Transport, &item.Status,
			&item.AttemptCount, &item.MessageID, &item.LastError, &item.RecipientName, &item.RecipientEmail,
			&item.Subject, &item.Body, &item.QueuedAt, &item.NextAttemptAt, &item.SentAt,
			&item.CreatedAt, &item.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	item.AttemptCount++
	now := time.Now().UTC()
	if _, err := tx.Exec(ctx, `
		UPDATE outreach_deliveries
		SET status='sending', attempt_count=$2, updated_at=$3
		WHERE id=$1`, item.ID, item.AttemptCount, now); err != nil {
		return nil, err
	}
	item.Status = "sending"
	item.UpdatedAt = now
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &item, nil
}

func (s *Store) FinishOutreachDelivery(ctx context.Context, item domain.Delivery, sendErr error, maxAttempts int) error {
	if maxAttempts < 1 {
		maxAttempts = 5
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	now := time.Now().UTC()
	if sendErr == nil {
		if _, err := tx.Exec(ctx, `
			UPDATE outreach_deliveries
			SET status='sent', message_id=NULLIF($2,''), last_error=NULL, sent_at=$3, updated_at=$3
			WHERE id=$1`, item.ID, item.MessageID, now); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `
			UPDATE outreach SET status='sent', sent_at=$2, updated_at=$2 WHERE id=$1`, item.OutreachID, now); err != nil {
			return err
		}
		return tx.Commit(ctx)
	}

	backoff := time.Duration(1<<minInt(item.AttemptCount-1, 6)) * time.Minute
	next := now.Add(backoff)
	terminal := item.AttemptCount >= maxAttempts
	if _, err := tx.Exec(ctx, `
		UPDATE outreach_deliveries
		SET status='failed', last_error=$2, next_attempt_at=$3, updated_at=$4
		WHERE id=$1`, item.ID, sendErr.Error(), next, now); err != nil {
		return err
	}
	if terminal {
		if _, err := tx.Exec(ctx, `UPDATE outreach SET status='failed', updated_at=$2 WHERE id=$1`, item.OutreachID, now); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (s *Store) CancelOutreachDelivery(ctx context.Context, projectID, outreachID string) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE outreach_deliveries d
		SET status='cancelled', updated_at=now()
		FROM outreach o
		WHERE d.outreach_id=o.id AND o.id=$1 AND o.project_id=$2 AND d.status <> 'sent'`, outreachID, projectID)
	return err
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
