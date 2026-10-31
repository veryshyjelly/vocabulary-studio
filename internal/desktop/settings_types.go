package desktop

const AppVersion = "1.0.0"

// SessionStoreMode identifies how authenticated sessions are persisted on desktop.
type SessionStoreMode string

const (
	SessionStoreKeyring SessionStoreMode = "keyring"
	SessionStoreFile    SessionStoreMode = "file"
	SessionStoreMemory  SessionStoreMode = "memory"
)

// IsValid reports whether the mode is one of "keyring", "file", or "memory".
func (m SessionStoreMode) IsValid() bool {
	return m == SessionStoreKeyring || m == SessionStoreFile || m == SessionStoreMemory
}

// AppPreferences stores non-sensitive user preferences for the desktop app.
type AppPreferences struct {
	Theme               string           `json:"theme"`                // "light", "dark", "system"
	PronunciationAccent string           `json:"pronunciation_accent"` // "US", "UK"
	DefaultView         string           `json:"default_view"`         // "home", "practice", "dictionary", "lists", "discover", "progress"
	SessionStoreMode    SessionStoreMode `json:"session_store_mode"`   // "keyring", "file", "memory"
	SearchHistory       []string         `json:"search_history"`
	AppVersion          string           `json:"app_version"`
}

// DefaultPreferences returns the default desktop application preferences.
func DefaultPreferences() AppPreferences {
	return AppPreferences{
		Theme:               "system",
		PronunciationAccent: "US",
		DefaultView:         "home",
		SessionStoreMode:    SessionStoreKeyring,
		SearchHistory:       []string{},
		AppVersion:          AppVersion,
	}
}
