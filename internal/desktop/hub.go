package desktop

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	vocab "github.com/veryshyjelly/vocabulary_go/sdk"
)

// Hub coordinates the vocabulary_go Client, active learning sessions (Trainer, Bee, Quiz),
// user preferences, and per-operation context cancellation for the Wails desktop application.
//
// Hub is safe for concurrent use by multiple goroutines.
type Hub struct {
	mu         sync.RWMutex
	configDir  string
	prefs      AppPreferences
	client     *vocab.Client
	clientOpts []vocab.Option
	emitEvent  func(name string, data any)

	opMu      sync.Mutex
	opCancels map[string]context.CancelFunc

	trainerMu      sync.Mutex
	session        *vocab.TrainerSession
	sessionActType string
	sessionListID  int64
	lastState      *vocab.ChallengeStateResponse
	lastOutcome    *AnswerFeedbackDTO

	beeMu      sync.Mutex
	beeListID  int64
	beeActID   string
	beeShownAt time.Time
	beeCurrent *vocab.BeeStateResponse
	beePending *vocab.BeeStateResponse

	quizMu    sync.Mutex
	quizState *vocab.QuizStateResponse

	Auth       *AuthService
	Trainer    *TrainerService
	Dictionary *DictionaryService
	Lists      *ListsService
	Discovery  *DiscoveryService
	Progress   *ProgressService
	Settings   *SettingsService
}

// HubOptions configures a desktop Hub instance.
type HubOptions struct {
	ConfigDir  string
	ClientOpts []vocab.Option
	EmitEvent  func(name string, data any)
}

// NewHub initializes the desktop Hub, loads persisted preferences, constructs the
// appropriate SessionStore, and initializes all 7 domain services.
func NewHub(opts HubOptions) (*Hub, error) {
	cfgDir := strings.TrimSpace(opts.ConfigDir)
	if cfgDir == "" {
		userCfg, err := os.UserConfigDir()
		if err != nil || userCfg == "" {
			userCfg = ".scratch"
		}
		cfgDir = filepath.Join(userCfg, "vocabulary_desktop")
	}

	h := &Hub{
		configDir:  cfgDir,
		prefs:      DefaultPreferences(),
		clientOpts: opts.ClientOpts,
		emitEvent:  opts.EmitEvent,
		opCancels:  make(map[string]context.CancelFunc),
	}

	_ = h.loadPreferences()

	client, err := h.buildClientForMode(h.prefs.SessionStoreMode)
	if err != nil {
		return nil, err
	}
	h.client = client

	h.Auth = &AuthService{hub: h}
	h.Trainer = &TrainerService{hub: h}
	h.Dictionary = &DictionaryService{hub: h}
	h.Lists = &ListsService{hub: h}
	h.Discovery = &DiscoveryService{hub: h}
	h.Progress = &ProgressService{hub: h}
	h.Settings = &SettingsService{hub: h}

	return h, nil
}

func (h *Hub) buildSessionStore(mode SessionStoreMode) vocab.SessionStore {
	switch mode {
	case SessionStoreMemory:
		return vocab.NewMemorySessionStore()
	case SessionStoreFile:
		return vocab.NewFileSessionStore(filepath.Join(h.configDir, "session.json"))
	default:
		return vocab.NewKeyringSessionStore("vocabulary_desktop", "default")
	}
}

func (h *Hub) buildClientForMode(mode SessionStoreMode) (*vocab.Client, error) {
	store := h.buildSessionStore(mode)
	allOpts := make([]vocab.Option, 0, len(h.clientOpts)+1)
	allOpts = append(allOpts, vocab.WithSessionStore(store))
	allOpts = append(allOpts, h.clientOpts...)
	return vocab.NewClient(allOpts...)
}

// Client returns the active *vocab.Client.
func (h *Hub) Client() *vocab.Client {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.client
}

// Emit dispatches an asynchronous Wails event if an event emitter is registered.
func (h *Hub) Emit(name string, data any) {
	if h.emitEvent != nil {
		h.emitEvent(name, data)
	}
}

// BeginOperation cancels any previous in-flight operation with the same key and returns a new derived context.
// This guarantees stale autocomplete, lookup, or list-loading requests are canceled immediately.
func (h *Hub) BeginOperation(parent context.Context, key string, timeout time.Duration) (context.Context, context.CancelFunc) {
	if parent == nil {
		parent = context.Background()
	}
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	ctx, cancel := context.WithTimeout(parent, timeout)

	h.opMu.Lock()
	if prev, ok := h.opCancels[key]; ok && prev != nil {
		prev()
	}
	h.opCancels[key] = cancel
	h.opMu.Unlock()

	return ctx, cancel
}

// CancelOperation explicitly cancels an in-flight operation by key.
func (h *Hub) CancelOperation(key string) {
	h.opMu.Lock()
	if prev, ok := h.opCancels[key]; ok && prev != nil {
		prev()
		delete(h.opCancels, key)
	}
	h.opMu.Unlock()
}

// Shutdown cancels all in-flight operations, flushes preferences, and closes idle HTTP connections.
func (h *Hub) Shutdown() {
	h.opMu.Lock()
	for k, cancel := range h.opCancels {
		if cancel != nil {
			cancel()
		}
		delete(h.opCancels, k)
	}
	h.opMu.Unlock()

	c := h.Client()
	if c != nil {
		c.CloseIdleConnections()
	}
}

func (h *Hub) preferencesPath() string {
	return filepath.Join(h.configDir, "preferences.json")
}

func (h *Hub) loadPreferences() error {
	data, err := os.ReadFile(h.preferencesPath())
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	var loaded AppPreferences
	if err := json.Unmarshal(data, &loaded); err != nil {
		return err
	}
	if loaded.Theme == "" {
		loaded.Theme = "system"
	}
	if loaded.PronunciationAccent == "" {
		loaded.PronunciationAccent = "US"
	}
	if loaded.DefaultView == "" {
		loaded.DefaultView = "home"
	}
	if !loaded.SessionStoreMode.IsValid() {
		loaded.SessionStoreMode = SessionStoreKeyring
	}
	if loaded.SearchHistory == nil {
		loaded.SearchHistory = []string{}
	}
	loaded.AppVersion = AppVersion

	h.mu.Lock()
	h.prefs = loaded
	h.mu.Unlock()
	return nil
}

func (h *Hub) savePreferencesLocked() error {
	if h.configDir == "" {
		return nil
	}
	if err := os.MkdirAll(h.configDir, 0o700); err != nil {
		return err
	}
	h.prefs.AppVersion = AppVersion
	data, err := json.MarshalIndent(h.prefs, "", "  ")
	if err != nil {
		return err
	}
	path := h.preferencesPath()
	tmp, err := os.CreateTemp(h.configDir, ".prefs-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	ok := false
	defer func() {
		if !ok {
			_ = tmp.Close()
			_ = os.Remove(tmpName)
		}
	}()
	if err := tmp.Chmod(0o600); err != nil {
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		return err
	}
	if err := tmp.Sync(); err != nil {
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpName, path); err != nil {
		return err
	}
	ok = true
	return nil
}

func (h *Hub) recordSearchWord(word string) {
	word = strings.TrimSpace(word)
	if word == "" {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()

	filtered := make([]string, 0, len(h.prefs.SearchHistory)+1)
	filtered = append(filtered, word)
	for _, existing := range h.prefs.SearchHistory {
		if !strings.EqualFold(existing, word) {
			filtered = append(filtered, existing)
		}
		if len(filtered) >= 25 {
			break
		}
	}
	h.prefs.SearchHistory = filtered
	_ = h.savePreferencesLocked()
}

// SwitchSessionStoreMode changes the session persistence backend ("keyring", "file", or "memory")
// without silently downgrading on failure. If an active session exists, it is migrated to the new store
// and cleared from the previous store.
func (h *Hub) SwitchSessionStoreMode(ctx context.Context, newMode SessionStoreMode) error {
	if !newMode.IsValid() {
		return fmt.Errorf("vocab: invalid session store mode %q (expected keyring, file, or memory)", newMode)
	}

	h.mu.Lock()
	oldMode := h.prefs.SessionStoreMode
	oldClient := h.client
	h.mu.Unlock()

	if oldMode == newMode {
		return nil
	}

	snap := oldClient.ExportSession()
	newStore := h.buildSessionStore(newMode)

	// Explicitly verify the new store can save before switching so we never silently downgrade.
	if snap != nil && (snap.GUID != "" || len(snap.Cookies) > 0) {
		if err := newStore.Save(ctx, snap); err != nil {
			return fmt.Errorf("vocab: failed to persist session to %s store: %w", newMode, err)
		}
	}

	newClient, err := h.buildClientForMode(newMode)
	if err != nil {
		return err
	}
	if snap != nil {
		newClient.ImportSession(snap)
	}

	// Best-effort clear old store
	oldStore := h.buildSessionStore(oldMode)
	_ = oldStore.Clear(ctx)

	h.mu.Lock()
	h.client = newClient
	h.prefs.SessionStoreMode = newMode
	err = h.savePreferencesLocked()
	h.mu.Unlock()
	return err
}
