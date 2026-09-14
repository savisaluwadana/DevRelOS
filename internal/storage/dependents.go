package storage

import "context"

// DependentsError means a delete was blocked because rows that depend on the
// target still exist. The API layer maps this to 409 Conflict instead of
// letting a raw foreign-key violation (or a silent cascade) reach the client.
type DependentsError struct{ Message string }

func (e *DependentsError) Error() string { return e.Message }

func dependentsErr(message string) error { return &DependentsError{Message: message} }

// campaignItemReferences reports whether a campaign_items row still points at
// entityType/entityID. campaign_items.entity_id has no foreign key (it is a
// polymorphic reference across ten resource kinds), so the database will not
// catch a dangling reference on its own.
func (s *Store) campaignItemReferences(ctx context.Context, entityType, entityID string) (bool, error) {
	var exists bool
	err := s.pool.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM campaign_items WHERE entity_type=$1 AND entity_id=$2)`,
		entityType, entityID).Scan(&exists)
	return exists, err
}

// workItemSourceReferences reports whether a work_items row was generated
// from sourceType/sourceID (e.g. a pain point or signal). source_id has no
// foreign key either.
func (s *Store) workItemSourceReferences(ctx context.Context, sourceType, sourceID string) (bool, error) {
	var exists bool
	err := s.pool.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM work_items WHERE source_type=$1 AND source_id=$2)`,
		sourceType, sourceID).Scan(&exists)
	return exists, err
}

// feedbackSourceReferences is the feedback_items equivalent of
// workItemSourceReferences.
func (s *Store) feedbackSourceReferences(ctx context.Context, sourceType, sourceID string) (bool, error) {
	var exists bool
	err := s.pool.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM feedback_items WHERE source_type=$1 AND source_id=$2)`,
		sourceType, sourceID).Scan(&exists)
	return exists, err
}
