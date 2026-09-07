package connectors

import "time"

type Secret struct {
	ID              string     `json:"id"`
	WorkspaceID     string     `json:"workspaceId"`
	Provider        string     `json:"provider"`
	Name            string     `json:"name"`
	KeyVersion      int        `json:"keyVersion"`
	CreatedByUserID string     `json:"createdByUserId,omitempty"`
	RotatedAt       *time.Time `json:"rotatedAt,omitempty"`
	CreatedAt       time.Time  `json:"createdAt"`
	UpdatedAt       time.Time  `json:"updatedAt"`
}

type EncryptedSecret struct {
	Secret
	Ciphertext []byte
	Nonce      []byte
}
