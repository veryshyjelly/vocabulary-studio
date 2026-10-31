package vocab

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"
	"golang.org/x/time/rate"
)

var linkHeaderPartRegex = regexp.MustCompile(`<([^>]+)>;\s*rel="([^"]+)"`)

// Client is the top-level Vocabulary.com SDK client.
// It manages dual-tier authentication (web session cookies + Bearer JWT), rate limiting,
// gameplay pacing guards, and provides sub-services for Auth, Trainer, Lists, Dictionary, Progress, Bee, and Quiz.
//
// Client is safe for concurrent use by multiple goroutines.
type Client struct {
	httpClient       *http.Client
	customHTTPClient bool
	jar              *trackedCookieJar
	sessionStore     SessionStore
	limiter          *rate.Limiter
	pacingGuard      *PacingGuard
	clock            Clock
	maxResponseBytes int64
	logger           *slog.Logger
	tokenGroup       singleflight.Group

	webBaseURL    string
	apiBaseURL    string
	corpusBaseURL string
	audioBaseURL  string
	videoBaseURL  string
	userAgent     string
	timezone      string

	mu       sync.RWMutex
	guid     string
	userUID  string
	username string
	token    *TokenInfo

	Auth       *AuthService
	Trainer    *TrainerService
	Lists      *ListsService
	Dictionary *DictionaryService
	Progress   *ProgressService
	Bee        *BeeService
	Quiz       *QuizService
}

// NewClient creates a new Vocabulary.com Client with sensible defaults:
// - Pure-Go tracked CookieJar with publicsuffix list validation
// - Automatic Bearer JWT exchange and singleflight refresh
// - Rate limiting (10 req/s, burst 20)
// - Gameplay pacing guard enabled by default
// - Maximum response body size of 16 MiB
func NewClient(opts ...Option) (*Client, error) {
	jar, err := newTrackedCookieJar()
	if err != nil {
		return nil, err
	}

	discardLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
	clk := realClock{}

	c := &Client{
		httpClient: &http.Client{
			Jar:     jar,
			Timeout: 30 * time.Second,
		},
		jar:              jar,
		sessionStore:     NewMemorySessionStore(),
		limiter:          rate.NewLimiter(rate.Limit(10), 20),
		pacingGuard:      newPacingGuard(true, clk),
		clock:            clk,
		maxResponseBytes: DefaultMaxResponseBytes,
		logger:           discardLogger,
		webBaseURL:       DefaultWebBaseURL,
		apiBaseURL:       DefaultAPIBaseURL,
		corpusBaseURL:    DefaultCorpusBaseURL,
		audioBaseURL:     DefaultAudioBaseURL,
		videoBaseURL:     DefaultVideoBaseURL,
		userAgent:        DefaultUserAgent,
		timezone:         DefaultTimezone,
	}

	for _, opt := range opts {
		if opt == nil {
			continue
		}
		if err := opt(c); err != nil {
			return nil, err
		}
	}

	// Ensure our tracked cookie jar is attached even if caller supplied a custom *http.Client without a Jar.
	c.jar.allowURL = c.isTrustedHost
	if c.httpClient.Jar == nil {
		c.httpClient.Jar = c.jar
	}
	if !c.customHTTPClient && c.httpClient.CheckRedirect == nil {
		c.httpClient.CheckRedirect = c.checkRedirect
	}

	c.ensureTimezoneCookie()

	c.Auth = &AuthService{client: c}
	c.Trainer = &TrainerService{client: c}
	c.Lists = &ListsService{client: c}
	c.Dictionary = &DictionaryService{client: c}
	c.Progress = &ProgressService{client: c}
	c.Bee = &BeeService{client: c}
	c.Quiz = &QuizService{client: c}

	// Best-effort load from sessionStore on startup.
	_ = c.RestoreSession(context.Background())

	return c, nil
}

// checkRedirect strips sensitive headers if a redirect ever leaves trusted Vocabulary.com or configured hosts.
func (c *Client) checkRedirect(req *http.Request, via []*http.Request) error {
	if len(via) >= 10 {
		return errors.New("vocab: stopped after 10 redirects")
	}
	if len(via) > 0 && !c.isTrustedHost(req.URL) {
		req.Header.Del("Authorization")
		req.Header.Del("Cookie")
	}
	return nil
}

func (c *Client) isTrustedHost(u *url.URL) bool {
	if u == nil {
		return false
	}
	host := strings.ToLower(u.Hostname())
	if host == "vocabulary.com" || strings.HasSuffix(host, ".vocabulary.com") {
		return true
	}
	for _, base := range []string{c.webBaseURL, c.apiBaseURL, c.corpusBaseURL, c.audioBaseURL} {
		if bu, err := url.Parse(base); err == nil && bu.Host != "" && strings.EqualFold(bu.Host, u.Host) {
			return true
		}
	}
	return false
}

func (c *Client) ensureTimezoneCookie() {
	u, err := url.Parse(c.webBaseURL)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return
	}
	c.mu.RLock()
	tz := c.timezone
	c.mu.RUnlock()
	if tz == "" {
		tz = DefaultTimezone
	}
	if c.jar.CookieValue(u, "tz") != tz {
		c.jar.SetCookies(u, []*http.Cookie{
			{
				Name:   "tz",
				Value:  tz,
				Path:   "/",
				Secure: u.Scheme == "https",
			},
		})
	}
}

// RestoreSession loads the persisted SessionSnapshot from the configured SessionStore into memory.
func (c *Client) RestoreSession(ctx context.Context) error {
	if c.sessionStore == nil {
		return nil
	}
	snap, err := c.sessionStore.Load(ctx)
	if err != nil || snap == nil {
		return err
	}
	c.ImportSession(snap)
	return nil
}

// ImportSession populates the client's CookieJar, GUID, and cached Bearer JWT from a SessionSnapshot.
// The provided snapshot is deep-copied so subsequent caller mutations cannot affect client state.
func (c *Client) ImportSession(snap *SessionSnapshot) {
	if snap == nil {
		return
	}
	cloned := snap.Clone()
	c.jar.Import(cloned.Cookies, c.clock.Now())
	c.mu.Lock()
	c.guid = cloned.GUID
	c.userUID = cloned.UserUID
	c.username = cloned.Username
	if cloned.Timezone != "" {
		c.timezone = cloned.Timezone
	}
	c.token = cloned.Token
	c.mu.Unlock()
	c.ensureTimezoneCookie()
}

// ExportSession exports a deep copy of the client's current cookies, GUID, and Bearer JWT
// as a serializable SessionSnapshot.
func (c *Client) ExportSession() *SessionSnapshot {
	webU, _ := url.Parse(c.webBaseURL)
	apiU, _ := url.Parse(c.apiBaseURL)

	c.mu.RLock()
	guid := c.guid
	uid := c.userUID
	username := c.username
	tz := c.timezone
	tokCopy := c.token.Clone()
	now := c.clock.Now()
	c.mu.RUnlock()

	if guid == "" && webU != nil {
		guid = c.jar.CookieValue(webU, "guid")
	}

	return &SessionSnapshot{
		GUID:      guid,
		UserUID:   uid,
		Username:  username,
		Timezone:  tz,
		Token:     tokCopy,
		Cookies:   c.jar.Export([]*url.URL{webU, apiU}, now),
		UpdatedAt: now.UTC(),
	}
}

// FlushSession persists the current session state to the configured SessionStore.
func (c *Client) FlushSession(ctx context.Context) error {
	if c.sessionStore == nil {
		return nil
	}
	return c.sessionStore.Save(ctx, c.ExportSession())
}

// CloseIdleConnections closes any idle keep-alive connections on the underlying HTTP transport
// (useful on mobile when the app enters the background).
func (c *Client) CloseIdleConnections() {
	if c.httpClient != nil {
		c.httpClient.CloseIdleConnections()
	}
}

// do executes an HTTP request after waiting on the client rate limiter and applying standard headers.
func (c *Client) do(req *http.Request) (*http.Response, error) {
	if err := req.Context().Err(); err != nil {
		return nil, err
	}
	if c.limiter != nil {
		if err := c.limiter.Wait(req.Context()); err != nil {
			return nil, err
		}
	}
	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", c.userAgent)
	}
	c.logger.DebugContext(req.Context(), "vocab http request",
		slog.String("method", req.Method),
		slog.String("url", redactURLString(req.URL)),
	)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	// Capture guid cookie if updated by server.
	if webU, err := url.Parse(c.webBaseURL); err == nil && webU != nil {
		if g := c.jar.CookieValue(webU, "guid"); g != "" {
			c.mu.Lock()
			c.guid = g
			c.mu.Unlock()
		}
	}
	return resp, nil
}

func (c *Client) readResponseBody(resp *http.Response) ([]byte, error) {
	if resp == nil || resp.Body == nil {
		return nil, nil
	}
	limit := c.maxResponseBytes
	if limit <= 0 {
		limit = DefaultMaxResponseBytes
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("%w (limit %d bytes)", ErrResponseTooLarge, limit)
	}
	return data, nil
}

// doWebForm POSTs application/x-www-form-urlencoded data to www.vocabulary.com and decodes JSON (if dst != nil).
func (c *Client) doWebForm(ctx context.Context, path string, form url.Values, dst any) error {
	targetURL := c.resolveWebURL(path)
	var encoded string
	if form != nil {
		encoded = form.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, targetURL, strings.NewReader(encoded))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
	req.Header.Set("Accept", "application/json, text/javascript, */*; q=0.01")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	req.Header.Set("Origin", c.webBaseURL)
	req.Header.Set("Referer", c.webBaseURL+"/")

	resp, err := c.do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	rawBytes, err := c.readResponseBody(resp)
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return c.parseAPIError(http.MethodPost, targetURL, resp, rawBytes)
	}
	// Check if JSON contains structured error field (e.g. RestartChallengeException)
	if err := checkChallengeErrorJSON(http.MethodPost, targetURL, resp.StatusCode, rawBytes); err != nil {
		return err
	}
	if dst != nil {
		return decodeJSONBody(http.MethodPost, targetURL, resp, rawBytes, dst)
	}
	return nil
}

// doWebGet performs a GET request to www.vocabulary.com and returns the bounded response body.
func (c *Client) doWebGet(ctx context.Context, path string, query url.Values, accept string) ([]byte, *http.Response, error) {
	targetURL := appendQuery(c.resolveWebURL(path), query)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
	if err != nil {
		return nil, nil, err
	}
	if accept != "" {
		req.Header.Set("Accept", accept)
	}
	resp, err := c.do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	rawBytes, err := c.readResponseBody(resp)
	if err != nil {
		return nil, resp, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return rawBytes, resp, c.parseAPIError(http.MethodGet, targetURL, resp, rawBytes)
	}
	return rawBytes, resp, nil
}

// doWebJSONGet performs a GET request to www.vocabulary.com and decodes the JSON response into dst.
func (c *Client) doWebJSONGet(ctx context.Context, path string, query url.Values, dst any) error {
	rawBytes, resp, err := c.doWebGet(ctx, path, query, "application/json")
	if err != nil {
		return err
	}
	targetURL := appendQuery(c.resolveWebURL(path), query)
	return decodeJSONBody(http.MethodGet, targetURL, resp, rawBytes, dst)
}

// doCorpusGet performs a GET request against corpus.vocabulary.com and decodes the JSON response into dst.
func (c *Client) doCorpusGet(ctx context.Context, path string, query url.Values, dst any) error {
	targetURL := appendQuery(c.resolveCorpusURL(path), query)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Origin", c.webBaseURL)
	req.Header.Set("Referer", c.webBaseURL+"/")

	resp, err := c.do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	rawBytes, err := c.readResponseBody(resp)
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return c.parseAPIError(http.MethodGet, targetURL, resp, rawBytes)
	}
	return decodeJSONBody(http.MethodGet, targetURL, resp, rawBytes, dst)
}

// doAPI executes a Bearer-authenticated request against https://api.vocabulary.com/1.0/...
// with automatic token acquisition and singleflight refresh on HTTP 401/403.
func (c *Client) doAPI(ctx context.Context, method, path string, query url.Values, jsonBody any, formBody url.Values, dst any) (*http.Response, error) {
	targetURL := appendQuery(c.resolveAPIURL(path), query)

	var encodedBody []byte
	var contentType string
	if jsonBody != nil {
		b, err := json.Marshal(jsonBody)
		if err != nil {
			return nil, err
		}
		encodedBody = b
		contentType = "application/json"
	} else if formBody != nil {
		encodedBody = []byte(formBody.Encode())
		contentType = "application/x-www-form-urlencoded; charset=UTF-8"
	}

	var lastAuthErr error
	for attempt := 0; attempt < 2; attempt++ {
		tok, err := c.Auth.EnsureToken(ctx, attempt > 0)
		if err != nil {
			return nil, err
		}

		var bodyReader io.Reader
		if encodedBody != nil {
			bodyReader = bytes.NewReader(encodedBody)
		}
		req, err := http.NewRequestWithContext(ctx, method, targetURL, bodyReader)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+tok.AccessToken)
		req.Header.Set("Accept", "application/json")
		c.mu.RLock()
		tz := c.timezone
		c.mu.RUnlock()
		if tz != "" {
			req.Header.Set("X-Time-Zone", tz)
		}
		if contentType != "" {
			req.Header.Set("Content-Type", contentType)
		}
		req.Header.Set("Origin", c.webBaseURL)
		req.Header.Set("Referer", c.webBaseURL+"/")

		resp, err := c.do(req)
		if err != nil {
			return nil, err
		}
		rawBytes, readErr := c.readResponseBody(resp)
		resp.Body.Close()
		if readErr != nil {
			return resp, readErr
		}

		if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
			lastAuthErr = c.parseAPIError(method, targetURL, resp, rawBytes)
			if attempt == 0 {
				c.Auth.InvalidateToken()
				continue
			}
			return resp, lastAuthErr
		}

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return resp, c.parseAPIError(method, targetURL, resp, rawBytes)
		}

		if dst != nil {
			if err := decodeJSONBody(method, targetURL, resp, rawBytes, dst); err != nil {
				return resp, err
			}
		}
		return resp, nil
	}

	if lastAuthErr != nil {
		return nil, lastAuthErr
	}
	return nil, ErrSessionExpired
}

func decodeJSONBody(method, targetURL string, resp *http.Response, rawBytes []byte, dst any) error {
	trimmed := bytes.TrimSpace(rawBytes)
	if len(trimmed) == 0 {
		return &APIError{
			StatusCode: statusCodeOf(resp),
			Method:     method,
			URL:        targetURL,
			Message:    "empty response body where JSON was expected",
			Cause:      ErrUnexpectedResponse,
		}
	}
	if looksLikeHTML(trimmed, resp) {
		cause := ErrUnexpectedResponse
		if resp != nil && resp.Request != nil && resp.Request.URL != nil && strings.HasPrefix(resp.Request.URL.Path, "/login") {
			cause = ErrSessionExpired
		}
		return &APIError{
			StatusCode:  statusCodeOf(resp),
			Method:      method,
			URL:         targetURL,
			ContentType: headerOf(resp, "Content-Type"),
			Message:     "received HTML response where JSON was expected",
			RawBody:     string(rawBytes),
			Cause:       cause,
		}
	}
	if err := json.Unmarshal(trimmed, dst); err != nil {
		return &APIError{
			StatusCode:  statusCodeOf(resp),
			Method:      method,
			URL:         targetURL,
			ContentType: headerOf(resp, "Content-Type"),
			Message:     fmt.Sprintf("decode JSON: %v", err),
			RawBody:     string(rawBytes),
			Cause:       fmt.Errorf("%w: %v", ErrUnexpectedResponse, err),
		}
	}
	return nil
}

func looksLikeHTML(trimmed []byte, resp *http.Response) bool {
	if len(trimmed) > 0 && trimmed[0] == '<' {
		prefix := strings.ToLower(string(trimmed[:min(len(trimmed), 32)]))
		if strings.HasPrefix(prefix, "<!doctype") || strings.HasPrefix(prefix, "<html") || strings.HasPrefix(prefix, "<head") || strings.HasPrefix(prefix, "<body") {
			return true
		}
	}
	if resp != nil {
		ct := strings.ToLower(resp.Header.Get("Content-Type"))
		if strings.Contains(ct, "text/html") && len(trimmed) > 0 && trimmed[0] != '{' && trimmed[0] != '[' {
			return true
		}
	}
	return false
}

func statusCodeOf(resp *http.Response) int {
	if resp == nil {
		return 0
	}
	return resp.StatusCode
}

func headerOf(resp *http.Response, key string) string {
	if resp == nil {
		return ""
	}
	return resp.Header.Get(key)
}

func appendQuery(baseURL string, query url.Values) string {
	if len(query) == 0 {
		return baseURL
	}
	sep := "?"
	if strings.Contains(baseURL, "?") {
		sep = "&"
	}
	return baseURL + sep + query.Encode()
}

func (c *Client) resolveWebURL(path string) string {
	return resolveHostPath(c.webBaseURL, path)
}

func (c *Client) resolveAPIURL(path string) string {
	return resolveHostPath(c.apiBaseURL, path)
}

func (c *Client) resolveCorpusURL(path string) string {
	return resolveHostPath(c.corpusBaseURL, path)
}

func resolveHostPath(base, path string) string {
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return path
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return strings.TrimRight(base, "/") + path
}

func (c *Client) parseAPIError(method, targetURL string, resp *http.Response, rawBytes []byte) error {
	statusCode := statusCodeOf(resp)
	apiErr := &APIError{
		StatusCode:  statusCode,
		Method:      method,
		URL:         targetURL,
		ContentType: headerOf(resp, "Content-Type"),
		RetryAfter:  parseRetryAfter(headerOf(resp, "Retry-After"), c.clock.Now()),
		RawBody:     string(rawBytes),
	}
	var payload struct {
		Error   string `json:"error"`
		Message string `json:"message"`
		Reason  string `json:"reason"`
	}
	if json.Unmarshal(rawBytes, &payload) == nil {
		apiErr.ErrorCode = payload.Error
		if apiErr.ErrorCode == "" {
			apiErr.ErrorCode = payload.Reason
		}
		apiErr.Message = payload.Message
	}
	if statusCode == http.StatusServiceUnavailable && apiErr.RetryAfter > 0 {
		apiErr.Cause = ErrRateLimited
	}
	return apiErr
}

func checkChallengeErrorJSON(method, targetURL string, statusCode int, rawBytes []byte) error {
	trimmed := bytes.TrimSpace(rawBytes)
	if len(trimmed) == 0 || trimmed[0] != '{' {
		return nil
	}
	var errPayload struct {
		Error   string `json:"error"`
		Message string `json:"message"`
	}
	if json.Unmarshal(trimmed, &errPayload) == nil && errPayload.Error != "" {
		return &APIError{
			StatusCode: statusCode,
			Method:     method,
			URL:        targetURL,
			ErrorCode:  errPayload.Error,
			Message:    errPayload.Message,
			RawBody:    string(rawBytes),
		}
	}
	return nil
}

// ParseLinkHeader parses RFC 5988 Link headers (e.g. `<https://api.vocabulary.com/1.0/units?skip=10>; rel="next"`)
// into a map of relation name -> relative URI path+query so it can be passed back to API methods directly.
func ParseLinkHeader(linkHeader, apiBaseURL string) map[string]string {
	if strings.TrimSpace(linkHeader) == "" {
		return nil
	}
	matches := linkHeaderPartRegex.FindAllStringSubmatch(linkHeader, -1)
	if len(matches) == 0 {
		return nil
	}
	out := make(map[string]string, len(matches))
	trimmedBase := strings.TrimRight(apiBaseURL, "/")
	for _, m := range matches {
		if len(m) < 3 {
			continue
		}
		uri := strings.TrimSpace(m[1])
		rel := strings.TrimSpace(m[2])
		if trimmedBase != "" && strings.HasPrefix(uri, trimmedBase) {
			uri = strings.TrimPrefix(uri, trimmedBase)
		} else if strings.HasPrefix(uri, DefaultAPIBaseURL) {
			uri = strings.TrimPrefix(uri, DefaultAPIBaseURL)
		} else if idx := strings.Index(uri, "/1.0/"); idx >= 0 {
			uri = uri[idx+len("/1.0"):]
		}
		out[rel] = uri
	}
	return out
}

func redactURLString(u *url.URL) string {
	if u == nil {
		return ""
	}
	if u.RawQuery == "" {
		return u.String()
	}
	q := u.Query()
	redacted := false
	for _, secretParam := range []string{"access_token", "refresh_token", "secret", "password"} {
		if q.Has(secretParam) {
			q.Set(secretParam, "REDACTED")
			redacted = true
		}
	}
	if !redacted {
		return u.String()
	}
	cp := *u
	cp.RawQuery = q.Encode()
	return cp.String()
}

func parseTotalCountHeader(resp *http.Response) int {
	if resp == nil {
		return 0
	}
	if tc := resp.Header.Get("X-Total-Count"); tc != "" {
		if n, err := strconv.Atoi(tc); err == nil {
			return n
		}
	}
	return 0
}
