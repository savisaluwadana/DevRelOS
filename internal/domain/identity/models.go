package identity

import "time"

type User struct {
	ID           string    `json:"id"`
	Email        string    `json:"email"`
	DisplayName  string    `json:"displayName"`
	PasswordHash string    `json:"-"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

type Membership struct {
	WorkspaceID   string    `json:"workspaceId"`
	WorkspaceSlug string    `json:"workspaceSlug"`
	WorkspaceName string    `json:"workspaceName"`
	UserID        string    `json:"userId"`
	Email         string    `json:"email"`
	DisplayName   string    `json:"displayName"`
	Role          string    `json:"role"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

type Principal struct {
	System      bool         `json:"system"`
	User        *User        `json:"user,omitempty"`
	Memberships []Membership `json:"memberships"`
}

type Session struct {
	ID        string    `json:"id"`
	UserID    string    `json:"userId"`
	ExpiresAt time.Time `json:"expiresAt"`
	CreatedAt time.Time `json:"createdAt"`
}
