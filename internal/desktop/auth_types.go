package desktop

import (
	vocab "github.com/veryshyjelly/vocabulary-studio/sdk"
)

// AccountStatusDTO contains strictly non-sensitive user identity and subscription state.
// Secrets (passwords, cookies, GUIDs, Bearer JWTs) are never included.
type AccountStatusDTO struct {
	Authenticated    bool              `json:"authenticated"`
	ValidUser        bool              `json:"valid_user"`
	UID              string            `json:"uid,omitempty"`
	Nickname         string            `json:"nickname,omitempty"`
	FullName         string            `json:"full_name,omitempty"`
	Email            string            `json:"email,omitempty"`
	Role             string            `json:"role,omitempty"`
	Points           int               `json:"points"`
	Level            vocab.UserLevel   `json:"level"`
	Paid             bool              `json:"paid"`
	FullVocabTrainer bool              `json:"full_vocab_trainer"`
	SessionStoreMode SessionStoreMode  `json:"session_store_mode"`
	PersistenceState string            `json:"persistence_state"`
	Bowl             *vocab.BowlSeason `json:"bowl,omitempty"`
}

// SessionPersistenceStatusDTO reports save health without backend error details.
type SessionPersistenceStatusDTO struct {
	State string `json:"state"`
}
