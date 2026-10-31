package vocab

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/zalando/go-keyring"
	"golang.org/x/net/publicsuffix"
)

// SerializableCookie captures a cookie and the URL domain it belongs to so a cookiejar.Jar
// can be reconstructed across desktop/mobile app restarts.
type SerializableCookie struct {
	RawURL   string    `json:"raw_url"`
	Name     string    `json:"name"`
	Value    string    `json:"value"`
	Path     string    `json:"path,omitempty"`
	Domain   string    `json:"domain,omitempty"`
	Expires  time.Time `json:"expires,omitempty"`
	Secure   bool      `json:"secure,omitempty"`
	HttpOnly bool      `json:"http_only,omitempty"`
}

// String returns a redacted representation of SerializableCookie safe for logging.
func (c SerializableCookie) String() string {
	val := ""
	if c.Value != "" {
		val = "[REDACTED]"
	}
	return fmt.Sprintf("SerializableCookie{RawURL:%q, Name:%q, Value:%q, Path:%q}", c.RawURL, c.Name, val, c.Path)
}

// GoString returns a redacted representation of SerializableCookie when formatted with %#v.
func (c SerializableCookie) GoString() string {
	return c.String()
}

// LogValue implements slog.LogValuer with Value redacted.
func (c SerializableCookie) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("raw_url", c.RawURL),
		slog.String("name", c.Name),
		slog.String("value", "[REDACTED]"),
		slog.String("path", c.Path),
	)
}

// IsExpired reports whether the cookie has a non-zero expiration time that is in the past at `now`.
func (c SerializableCookie) IsExpired(now time.Time) bool {
	return !c.Expires.IsZero() && !c.Expires.After(now)
}

// TokenInfo stores the Bearer JWT issued by https://api.vocabulary.com/1.0/auth/token.
type TokenInfo struct {
	TokenType    string    `json:"token_type"`
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresIn    int       `json:"expires_in,omitempty"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// Clone returns a deep copy of TokenInfo.
func (t *TokenInfo) Clone() *TokenInfo {
	if t == nil {
		return nil
	}
	cp := *t
	return &cp
}

// IsValid reports whether the token is non-empty and not expiring within the next 30 seconds.
func (t *TokenInfo) IsValid(now time.Time) bool {
	if t == nil || t.AccessToken == "" {
		return false
	}
	return now.Add(30 * time.Second).Before(t.ExpiresAt)
}

// String returns a redacted representation of TokenInfo safe for debug output.
func (t TokenInfo) String() string {
	access := ""
	if t.AccessToken != "" {
		access = "[REDACTED]"
	}
	refresh := ""
	if t.RefreshToken != "" {
		refresh = "[REDACTED]"
	}
	return fmt.Sprintf("TokenInfo{TokenType:%q, AccessToken:%q, RefreshToken:%q, ExpiresAt:%s}",
		t.TokenType, access, refresh, t.ExpiresAt.Format(time.RFC3339))
}

// GoString returns a redacted representation of TokenInfo when formatted with %#v.
func (t TokenInfo) GoString() string {
	return t.String()
}

// LogValue implements slog.LogValuer with tokens redacted.
func (t TokenInfo) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("token_type", t.TokenType),
		slog.String("access_token", "[REDACTED]"),
		slog.String("refresh_token", "[REDACTED]"),
		slog.Time("expires_at", t.ExpiresAt),
	)
}

// SessionSnapshot holds the complete serializable authentication state (both web cookies and API JWT).
type SessionSnapshot struct {
	GUID      string               `json:"guid,omitempty"`
	UserUID   string               `json:"user_uid,omitempty"`
	Username  string               `json:"username,omitempty"`
	Timezone  string               `json:"timezone,omitempty"`
	Token     *TokenInfo           `json:"token,omitempty"`
	Cookies   []SerializableCookie `json:"cookies,omitempty"`
	UpdatedAt time.Time            `json:"updated_at"`
}

// Clone returns an independent deep copy of the SessionSnapshot.
func (s *SessionSnapshot) Clone() *SessionSnapshot {
	if s == nil {
		return nil
	}
	cp := *s
	cp.Token = s.Token.Clone()
	if s.Cookies != nil {
		cp.Cookies = make([]SerializableCookie, len(s.Cookies))
		copy(cp.Cookies, s.Cookies)
	}
	return &cp
}

// String returns a redacted summary of SessionSnapshot safe for logging.
func (s SessionSnapshot) String() string {
	guid := ""
	if s.GUID != "" {
		guid = "[REDACTED]"
	}
	return fmt.Sprintf("SessionSnapshot{UserUID:%q, Username:%q, GUID:%q, Cookies:%d, HasToken:%v, UpdatedAt:%s}",
		s.UserUID, s.Username, guid, len(s.Cookies), s.Token != nil, s.UpdatedAt.Format(time.RFC3339))
}

// GoString returns a redacted summary of SessionSnapshot when formatted with %#v.
func (s SessionSnapshot) GoString() string {
	return s.String()
}

// LogValue implements slog.LogValuer with sensitive fields redacted.
func (s SessionSnapshot) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("user_uid", s.UserUID),
		slog.String("username", s.Username),
		slog.String("timezone", s.Timezone),
		slog.Int("cookie_count", len(s.Cookies)),
		slog.Bool("has_token", s.Token != nil),
		slog.Time("updated_at", s.UpdatedAt),
	)
}

// SessionStore persists and loads a SessionSnapshot across process restarts.
// Desktop and mobile apps can implement this over iOS Keychain, Android EncryptedSharedPreferences,
// macOS Keychain, or an encrypted local file.
type SessionStore interface {
	Load(ctx context.Context) (*SessionSnapshot, error)
	Save(ctx context.Context, snap *SessionSnapshot) error
	Clear(ctx context.Context) error
}

// MemorySessionStore is a thread-safe in-memory SessionStore.
type MemorySessionStore struct {
	mu   sync.RWMutex
	snap *SessionSnapshot
}

// NewMemorySessionStore creates an in-memory session store.
func NewMemorySessionStore() *MemorySessionStore {
	return &MemorySessionStore{}
}

func (m *MemorySessionStore) Load(ctx context.Context) (*SessionSnapshot, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.snap.Clone(), nil
}

func (m *MemorySessionStore) Save(ctx context.Context, snap *SessionSnapshot) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.snap = snap.Clone()
	return nil
}

func (m *MemorySessionStore) Clear(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.snap = nil
	return nil
}

// FileSessionStore persists the SessionSnapshot to a local JSON file with 0600 permissions
// using atomic temporary-file replacement.
type FileSessionStore struct {
	mu   sync.Mutex
	path string
}

// NewFileSessionStore creates a file-backed SessionStore at the given path.
func NewFileSessionStore(path string) *FileSessionStore {
	return &FileSessionStore{path: path}
}

func (f *FileSessionStore) Load(ctx context.Context) (*SessionSnapshot, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.path == "" {
		return nil, nil
	}
	data, err := os.ReadFile(f.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	if len(bytes.TrimSpace(data)) == 0 {
		return nil, fmt.Errorf("%w: session file %q is empty", ErrCorruptedSession, f.path)
	}
	var snap SessionSnapshot
	if err := json.Unmarshal(data, &snap); err != nil {
		return nil, fmt.Errorf("%w: decode %q: %v", ErrCorruptedSession, f.path, err)
	}
	return &snap, nil
}

func (f *FileSessionStore) Save(ctx context.Context, snap *SessionSnapshot) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.path == "" {
		return fmt.Errorf("vocab: FileSessionStore path is empty")
	}
	if snap == nil {
		return f.clearLocked()
	}
	dir := filepath.Dir(f.path)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return err
		}
	} else {
		dir = "."
	}
	data, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		return err
	}

	tmpFile, err := os.CreateTemp(dir, ".vocab-session-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmpFile.Name()
	success := false
	defer func() {
		if !success {
			_ = tmpFile.Close()
			_ = os.Remove(tmpName)
		}
	}()

	if err := tmpFile.Chmod(0o600); err != nil {
		return err
	}
	if _, err := tmpFile.Write(data); err != nil {
		return err
	}
	if err := tmpFile.Sync(); err != nil {
		return err
	}
	if err := tmpFile.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpName, f.path); err != nil {
		return err
	}
	_ = os.Chmod(f.path, 0o600)
	success = true
	return nil
}

func (f *FileSessionStore) Clear(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.clearLocked()
}

func (f *FileSessionStore) clearLocked() error {
	if f.path == "" {
		return nil
	}
	if err := os.Remove(f.path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

// KeyringSessionStore stores the SessionSnapshot in the OS native credential store
// (macOS Keychain, Windows Credential Manager, or Linux Secret Service).
type KeyringSessionStore struct {
	service string
	account string
}

// NewKeyringSessionStore creates an OS-keychain backed SessionStore.
func NewKeyringSessionStore(service, account string) *KeyringSessionStore {
	if service == "" {
		// Persisted storage identifier, independent of the repository/module name.
		service = "vocabulary_go"
	}
	if account == "" {
		account = "default"
	}
	return &KeyringSessionStore{service: service, account: account}
}

func (k *KeyringSessionStore) Load(ctx context.Context) (*SessionSnapshot, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	raw, err := keyring.Get(k.service, k.account)
	if err != nil {
		if errors.Is(err, keyring.ErrNotFound) {
			return nil, nil
		}
		return nil, err
	}
	if len(bytes.TrimSpace([]byte(raw))) == 0 {
		return nil, fmt.Errorf("%w: keyring entry is empty", ErrCorruptedSession)
	}
	var snap SessionSnapshot
	if err := json.Unmarshal([]byte(raw), &snap); err != nil {
		return nil, fmt.Errorf("%w: decode keyring entry: %v", ErrCorruptedSession, err)
	}
	return &snap, nil
}

func (k *KeyringSessionStore) Save(ctx context.Context, snap *SessionSnapshot) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if snap == nil {
		return k.Clear(ctx)
	}
	data, err := json.Marshal(snap)
	if err != nil {
		return err
	}
	return keyring.Set(k.service, k.account, string(data))
}

func (k *KeyringSessionStore) Clear(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	err := keyring.Delete(k.service, k.account)
	if errors.Is(err, keyring.ErrNotFound) {
		return nil
	}
	return err
}

// trackedCookieJar wraps standard net/http/cookiejar.Jar and tracks cookie metadata across known domains
// so they can be exported into a SessionSnapshot without losing Path, Domain, Expires, Secure, or HttpOnly.
type trackedCookieJar struct {
	mu       sync.RWMutex
	jar      *cookiejar.Jar
	tracked  map[string]SerializableCookie // key: origin + "|" + name + "|" + normalizedPath
	allowURL func(*url.URL) bool
}

func newTrackedCookieJar() (*trackedCookieJar, error) {
	jar, err := cookiejar.New(&cookiejar.Options{
		PublicSuffixList: publicsuffix.List,
	})
	if err != nil {
		return nil, err
	}
	return &trackedCookieJar{
		jar:     jar,
		tracked: make(map[string]SerializableCookie),
	}, nil
}

func normalizeCookiePath(p string) string {
	if p == "" {
		return "/"
	}
	return p
}

func cookieMapKey(origin, name, path string) string {
	return origin + "|" + name + "|" + normalizeCookiePath(path)
}

func (t *trackedCookieJar) SetCookies(u *url.URL, cookies []*http.Cookie) {
	if u == nil || len(cookies) == 0 {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.allowURL != nil && !t.allowURL(u) {
		return
	}
	t.jar.SetCookies(u, cookies)
	origin := u.Scheme + "://" + u.Host
	for _, c := range cookies {
		if c == nil || c.Name == "" {
			continue
		}
		path := normalizeCookiePath(c.Path)
		key := cookieMapKey(origin, c.Name, path)
		if c.MaxAge < 0 || c.Value == "" {
			delete(t.tracked, key)
			continue
		}
		expires := c.Expires
		if c.MaxAge > 0 && expires.IsZero() {
			expires = time.Now().UTC().Add(time.Duration(c.MaxAge) * time.Second)
		}
		t.tracked[key] = SerializableCookie{
			RawURL:   origin,
			Name:     c.Name,
			Value:    c.Value,
			Path:     path,
			Domain:   c.Domain,
			Expires:  expires,
			Secure:   c.Secure,
			HttpOnly: c.HttpOnly,
		}
	}
}

func (t *trackedCookieJar) Cookies(u *url.URL) []*http.Cookie {
	if u == nil {
		return nil
	}
	t.mu.RLock()
	defer t.mu.RUnlock()
	if t.allowURL != nil && !t.allowURL(u) {
		return nil
	}
	return t.jar.Cookies(u)
}

func (t *trackedCookieJar) Export(urls []*url.URL, now time.Time) []SerializableCookie {
	t.mu.RLock()
	defer t.mu.RUnlock()
	if now.IsZero() {
		now = time.Now()
	}

	seen := make(map[string]SerializableCookie, len(t.tracked))
	for k, sc := range t.tracked {
		if sc.Value == "" || sc.IsExpired(now) {
			continue
		}
		seen[k] = sc
	}

	for _, u := range urls {
		if u == nil || u.Scheme == "" || u.Host == "" {
			continue
		}
		origin := u.Scheme + "://" + u.Host
		for _, c := range t.jar.Cookies(u) {
			if c == nil || c.Name == "" || c.Value == "" {
				continue
			}
			// Find if we already tracked this cookie under any path on this origin.
			matchedKey := ""
			for k, existing := range seen {
				if existing.RawURL == origin && existing.Name == c.Name {
					matchedKey = k
					break
				}
			}
			if matchedKey != "" {
				existing := seen[matchedKey]
				existing.Value = c.Value
				seen[matchedKey] = existing
			} else {
				key := cookieMapKey(origin, c.Name, "/")
				seen[key] = SerializableCookie{
					RawURL: origin,
					Name:   c.Name,
					Value:  c.Value,
					Path:   "/",
					Secure: u.Scheme == "https",
				}
			}
		}
	}

	out := make([]SerializableCookie, 0, len(seen))
	for _, sc := range seen {
		out = append(out, sc)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].RawURL != out[j].RawURL {
			return out[i].RawURL < out[j].RawURL
		}
		if out[i].Name != out[j].Name {
			return out[i].Name < out[j].Name
		}
		return out[i].Path < out[j].Path
	})
	return out
}

func (t *trackedCookieJar) Import(cookies []SerializableCookie, now time.Time) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if now.IsZero() {
		now = time.Now()
	}
	for _, sc := range cookies {
		if sc.Name == "" || sc.Value == "" || sc.IsExpired(now) {
			continue
		}
		u, err := url.Parse(sc.RawURL)
		if err != nil || u.Scheme == "" || u.Host == "" {
			continue
		}
		path := normalizeCookiePath(sc.Path)
		sc.Path = path
		hc := &http.Cookie{
			Name:     sc.Name,
			Value:    sc.Value,
			Path:     path,
			Domain:   sc.Domain,
			Expires:  sc.Expires,
			Secure:   sc.Secure,
			HttpOnly: sc.HttpOnly,
		}
		t.jar.SetCookies(u, []*http.Cookie{hc})
		key := cookieMapKey(u.Scheme+"://"+u.Host, sc.Name, path)
		t.tracked[key] = sc
	}
}

func (t *trackedCookieJar) CookieValue(u *url.URL, name string) string {
	for _, c := range t.Cookies(u) {
		if c.Name == name && c.Value != "" {
			return c.Value
		}
	}
	return ""
}

func (t *trackedCookieJar) Reset() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	jar, err := cookiejar.New(&cookiejar.Options{
		PublicSuffixList: publicsuffix.List,
	})
	if err != nil {
		return err
	}
	t.jar = jar
	t.tracked = make(map[string]SerializableCookie)
	return nil
}
