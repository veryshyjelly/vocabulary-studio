package desktop

import (
	"context"
	"errors"
	"strings"
	"time"

	vocab "github.com/veryshyjelly/vocabulary-studio/sdk"
)

// AuthService exposes authentication and session lifecycle methods to the Wails frontend.
// It never returns passwords, cookies, GUIDs, or JWTs.
type AuthService struct {
	hub *Hub
}

// GetPersistenceStatus reads local save health without account verification or IO.
func (s *AuthService) GetPersistenceStatus() *SessionPersistenceStatusDTO {
	return &SessionPersistenceStatusDTO{State: s.hub.Client().PersistenceStatus().State}
}

// RetryPersistence retries saving to the selected backend without signing in again.
func (s *AuthService) RetryPersistence(ctx context.Context) (*SessionPersistenceStatusDTO, error) {
	ctx, cancel := s.hub.BeginOperation(ctx, "auth:persistence", 15*time.Second)
	defer cancel()
	if err := s.hub.Client().FlushSession(ctx); err != nil {
		// Store errors may include paths or credential-bearing backend payloads.
		return nil, errors.New("Your session could not be saved. Retry saving or choose another session storage option in Settings.")
	}
	return s.GetPersistenceStatus(), nil
}

// GetStatus checks the current session via POST /auth/me.json and returns non-sensitive account info.
func (s *AuthService) GetStatus(ctx context.Context) (*AccountStatusDTO, error) {
	ctx, cancel := s.hub.BeginOperation(ctx, "auth:status", 20*time.Second)
	defer cancel()

	me, err := s.hub.Client().Auth.Me(ctx)
	if err != nil {
		return s.unauthenticatedStatus(), err
	}
	return s.mapLoginData(me), nil
}

// RestoreSession attempts to restore a persisted session from the configured SessionStore
// and verifies it against POST /auth/me.json.
func (s *AuthService) RestoreSession(ctx context.Context) (*AccountStatusDTO, error) {
	ctx, cancel := s.hub.BeginOperation(ctx, "auth:restore", 20*time.Second)
	defer cancel()

	client := s.hub.Client()
	if err := client.RestoreSession(ctx); err != nil {
		// If keyring isn't populated or returns not found, return unauthenticated status cleanly
		return s.unauthenticatedStatus(), nil
	}
	me, err := client.Auth.Me(ctx)
	if err != nil {
		return s.unauthenticatedStatus(), nil
	}
	status := s.mapLoginData(me)
	s.hub.Emit("auth:changed", status)
	return status, nil
}

// Login authenticates with Vocabulary.com using username/email and password,
// optionally setting the preferred SessionStoreMode ("keyring", "file", or "memory") first.
func (s *AuthService) Login(ctx context.Context, username, password, storeMode string) (*AccountStatusDTO, error) {
	ctx, cancel := s.hub.BeginOperation(ctx, "auth:login", 30*time.Second)
	defer cancel()

	if mode := SessionStoreMode(strings.TrimSpace(storeMode)); mode != "" && mode.IsValid() {
		if err := s.hub.SwitchSessionStoreMode(ctx, mode); err != nil {
			return nil, err
		}
	}

	me, err := s.hub.Client().Auth.Login(ctx, username, password)
	if err != nil {
		return nil, err
	}
	status := s.mapLoginData(me)
	s.hub.Emit("auth:changed", status)
	return status, nil
}

// Logout terminates the session on Vocabulary.com and clears local cookies, tokens, and persistent storage.
func (s *AuthService) Logout(ctx context.Context) (*AccountStatusDTO, error) {
	ctx, cancel := s.hub.BeginOperation(ctx, "auth:logout", 20*time.Second)
	defer cancel()

	if err := s.hub.Client().Auth.Logout(ctx); err != nil {
		return nil, err
	}
	s.hub.Trainer.EndSession()
	status := s.unauthenticatedStatus()
	s.hub.Emit("auth:changed", status)
	return status, nil
}

func (s *AuthService) unauthenticatedStatus() *AccountStatusDTO {
	s.hub.mu.RLock()
	mode := s.hub.prefs.SessionStoreMode
	s.hub.mu.RUnlock()
	return &AccountStatusDTO{
		Authenticated:    false,
		ValidUser:        false,
		SessionStoreMode: mode,
		PersistenceState: s.hub.Client().PersistenceStatus().State,
		Level:            vocab.UserLevel{ID: "L01", Name: "Novice"},
	}
}

func (s *AuthService) mapLoginData(me *vocab.LoginData) *AccountStatusDTO {
	if me == nil {
		return s.unauthenticatedStatus()
	}
	s.hub.mu.RLock()
	mode := s.hub.prefs.SessionStoreMode
	s.hub.mu.RUnlock()

	return &AccountStatusDTO{
		Authenticated:    me.ValidUser && me.Auth.LoggedIn,
		ValidUser:        me.ValidUser,
		UID:              me.Auth.UID,
		Nickname:         me.Auth.Nickname,
		FullName:         me.Auth.FullName,
		Email:            me.Auth.Email,
		Role:             me.Auth.Role,
		Points:           me.Points,
		Level:            me.Level,
		Paid:             me.Paid,
		FullVocabTrainer: me.Perms.FullVocabTrainer,
		SessionStoreMode: mode,
		PersistenceState: s.hub.Client().PersistenceStatus().State,
		Bowl:             me.Bowl,
	}
}
