package identity

import "time"

type Session struct {
	ID          string     `json:"id"`
	UserID      string     `json:"userId"`
	WorkspaceID string     `json:"workspaceId"`
	ExpiresAt   time.Time  `json:"expiresAt"`
	LastSeenAt  *time.Time `json:"lastSeenAt,omitempty"`
	RevokedAt   *time.Time `json:"revokedAt,omitempty"`
	CreatedAt   time.Time  `json:"createdAt"`
}

type SessionPrincipal struct {
	Principal
	SessionID   string `json:"sessionId"`
	WorkspaceID string `json:"workspaceId"`
}

type WorkspaceAccess struct {
	WorkspaceID   string `json:"workspaceId"`
	WorkspaceSlug string `json:"workspaceSlug"`
	WorkspaceName string `json:"workspaceName"`
	Role          string `json:"role"`
}

type Invitation struct {
	ID              string     `json:"id"`
	WorkspaceID     string     `json:"workspaceId"`
	Email           string     `json:"email"`
	Role            string     `json:"role"`
	InvitedByUserID string     `json:"invitedByUserId,omitempty"`
	ExpiresAt       time.Time  `json:"expiresAt"`
	AcceptedAt      *time.Time `json:"acceptedAt,omitempty"`
	CreatedAt       time.Time  `json:"createdAt"`
}
