package desktop

import (
	"context"
	"strings"
)

// SettingsService manages non-sensitive desktop preferences and session store backend configuration.
type SettingsService struct {
	hub *Hub
}

// GetPreferences returns the current AppPreferences.
func (s *SettingsService) GetPreferences() AppPreferences {
	s.hub.mu.RLock()
	defer s.hub.mu.RUnlock()
	cp := s.hub.prefs
	cp.AppVersion = AppVersion
	if s.hub.prefs.SearchHistory != nil {
		cp.SearchHistory = make([]string, len(s.hub.prefs.SearchHistory))
		copy(cp.SearchHistory, s.hub.prefs.SearchHistory)
	}
	return cp
}

// UpdatePreferences updates theme, pronunciation accent, default view, and optionally session store mode.
func (s *SettingsService) UpdatePreferences(ctx context.Context, updated AppPreferences) (*AppPreferences, error) {
	if updated.SessionStoreMode != "" && updated.SessionStoreMode.IsValid() {
		if err := s.hub.SwitchSessionStoreMode(ctx, updated.SessionStoreMode); err != nil {
			return nil, err
		}
	}

	s.hub.mu.Lock()
	switch updated.Theme {
	case "light", "dark", "system":
		s.hub.prefs.Theme = updated.Theme
	}
	switch strings.ToUpper(updated.PronunciationAccent) {
	case "US", "UK":
		s.hub.prefs.PronunciationAccent = strings.ToUpper(updated.PronunciationAccent)
	}
	switch updated.DefaultView {
	case "home", "practice", "dictionary", "lists", "discover", "progress":
		s.hub.prefs.DefaultView = updated.DefaultView
	}
	s.hub.prefs.AppVersion = AppVersion
	err := s.hub.savePreferencesLocked()
	prefsCopy := s.hub.prefs
	s.hub.mu.Unlock()

	if err != nil {
		return nil, err
	}
	s.hub.Emit("settings:changed", prefsCopy)
	return &prefsCopy, nil
}
