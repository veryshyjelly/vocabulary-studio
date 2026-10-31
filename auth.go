package vocab

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// AuthService handles login, logout, session state inspection (/auth/me.json),
// and Bearer JWT exchange/refresh (https://api.vocabulary.com/1.0/auth/token).
//
// AuthService is safe for concurrent use by multiple goroutines.
type AuthService struct {
	client *Client
}

// LoginData represents the session/profile payload returned by POST /auth/me.json.
type LoginData struct {
	ValidUser     bool        `json:"validUser"`
	GUID          string      `json:"guid,omitempty"`
	Ima           string      `json:"ima,omitempty"`
	Paid          bool        `json:"paid,omitempty"`
	Plan          string      `json:"plan,omitempty"`
	Stage         string      `json:"stage,omitempty"`
	SubID         string      `json:"subId,omitempty"`
	VisitID       string      `json:"visitid,omitempty"`
	MaxAge        int         `json:"maxage,omitempty"`
	Points        int         `json:"points"`
	LocCn         string      `json:"locCn,omitempty"`
	Level         UserLevel   `json:"level"`
	Auth          AuthDetails `json:"auth"`
	Perms         UserPerms   `json:"perms"`
	Classes       UserClasses `json:"classes,omitempty"`
	CurrentPlan   *UserPlan   `json:"currentPlan,omitempty"`
	Bowl          *BowlSeason `json:"bowl,omitempty"`
	ValidUnitEd   bool        `json:"validUnitEditor,omitempty"`
	Notifications []any       `json:"notifications,omitempty"`
}

// UserLevel represents a learner's level and milestone progress (e.g. L01 Novice).
type UserLevel struct {
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	Milestone int     `json:"milestone"`
	Progress  float64 `json:"progress"`
}

// AuthDetails holds the authenticated user's identity inside LoginData.
type AuthDetails struct {
	LoggedIn  bool        `json:"loggedin"`
	UID       string      `json:"uid,omitempty"`
	Nickname  string      `json:"nickname,omitempty"`
	FullName  string      `json:"fullname,omitempty"`
	Email     string      `json:"email,omitempty"`
	Role      string      `json:"role,omitempty"`
	Method    string      `json:"method,omitempty"`
	IsUnder13 bool        `json:"isUnder13,omitempty"`
	IXL       bool        `json:"ixl,omitempty"`
	School    *SchoolInfo `json:"school,omitempty"`
}

// SchoolInfo holds school association metadata.
type SchoolInfo struct {
	ID      string `json:"id,omitempty"`
	Name    string `json:"n,omitempty"`
	Country string `json:"cn,omitempty"`
	State   string `json:"st,omitempty"`
	City    string `json:"ct,omitempty"`
	URL     string `json:"url,omitempty"`
}

// UserPerms holds feature flags and subscription capabilities for the user.
type UserPerms struct {
	SubscribeEmail     bool `json:"subscribeEmail"`
	FullVocabTrainer   bool `json:"fullVocabTrainer"`
	CreateClass        bool `json:"createClass"`
	CreateQuiz         bool `json:"createQuiz"`
	ViewClassReports   bool `json:"viewClassReports"`
	ViewSchoolReports  bool `json:"viewSchoolReports"`
	ListAccountUsers   bool `json:"listAccountUsers"`
	UnlinkAccountUsers bool `json:"unlinkAccountUsers"`
	EditAuthentication bool `json:"editAuthentication"`
}

// UserClasses holds class membership summary from /auth/me.json.
type UserClasses struct {
	ActiveSubscription        bool     `json:"activeSubscription"`
	ActiveTrial               bool     `json:"activeTrial"`
	AssignmentCount           int      `json:"assignmentCount"`
	IncompleteAssignmentCount int      `json:"incompleteAssignmentCount"`
	Empty                     bool     `json:"empty"`
	Paid                      bool     `json:"paid"`
	MemberOf                  []string `json:"memberOf,omitempty"`
	OwnerOf                   []string `json:"ownerOf,omitempty"`
}

// UserPlan describes the user's active subscription plan.
type UserPlan struct {
	ID          string `json:"_id,omitempty"`
	ForEducator bool   `json:"forEducator"`
	Name        string `json:"name,omitempty"`
	Paid        bool   `json:"paid"`
	ProductCode string `json:"productCode,omitempty"`
}

// BowlSeason describes the active Vocabulary Bowl season.
type BowlSeason struct {
	Period    string `json:"period"`
	Name      string `json:"name"`
	ShortName string `json:"shortName"`
	StartDay  string `json:"startDay"`
	EndDay    string `json:"endDay"`
}

// Login authenticates with Vocabulary.com via POST /login/, stores the session cookies
// (SESSION, guid), verifies authentication via POST /auth/me.json, exchanges the guid
// for a Bearer JWT, and persists the session to the configured SessionStore.
//
// If authentication fails or is canceled, the client's previous session state is restored.
func (a *AuthService) Login(ctx context.Context, username, password string) (*LoginData, error) {
	username = strings.TrimSpace(username)
	if username == "" || password == "" {
		return nil, ErrInvalidCredentials
	}

	prevSnap := a.client.ExportSession()
	restoreOnFailure := func() {
		_ = a.client.jar.Reset()
		a.client.ImportSession(prevSnap)
	}

	form := url.Values{
		"username":      {username},
		"password":      {password},
		"autoLogon":     {"true"},
		".cb-autoLogon": {"1"},
	}

	loginURL := a.client.resolveWebURL("/login/")
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, loginURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Origin", a.client.webBaseURL)
	req.Header.Set("Referer", loginURL)

	resp, err := a.client.do(req)
	if err != nil {
		restoreOnFailure()
		return nil, err
	}
	bodyBytes, readErr := a.client.readResponseBody(resp)
	resp.Body.Close()
	if readErr != nil {
		restoreOnFailure()
		return nil, readErr
	}

	// Verify session state via /auth/me.json
	me, err := a.Me(ctx)
	if err != nil {
		restoreOnFailure()
		return nil, err
	}
	if !me.ValidUser || !me.Auth.LoggedIn {
		restoreOnFailure()
		bodyStr := string(bodyBytes)
		if strings.Contains(bodyStr, "maxattempts") {
			return nil, ErrMaxLoginAttempts
		}
		return nil, ErrInvalidCredentials
	}

	a.commitAuthenticatedUser(ctx, username, me)
	return me, nil
}

// LoginVocabTrainer authenticates via the inline VocabTrainer endpoint (POST /login/vocabtrainer).
// If authentication fails or is canceled, the client's previous session state is restored.
func (a *AuthService) LoginVocabTrainer(ctx context.Context, username, password string) (*LoginData, error) {
	username = strings.TrimSpace(username)
	if username == "" || password == "" {
		return nil, ErrInvalidCredentials
	}

	prevSnap := a.client.ExportSession()
	restoreOnFailure := func() {
		_ = a.client.jar.Reset()
		a.client.ImportSession(prevSnap)
	}

	form := url.Values{
		"username":  {username},
		"password":  {password},
		"autoLogon": {"true"},
	}
	loginURL := a.client.resolveWebURL("/login/vocabtrainer")
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, loginURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
	req.Header.Set("Accept", "application/json, text/html, */*")
	req.Header.Set("Origin", a.client.webBaseURL)
	req.Header.Set("Referer", a.client.resolveWebURL("/vocabtrainer/"))

	resp, err := a.client.do(req)
	if err != nil {
		restoreOnFailure()
		return nil, err
	}
	rawBytes, readErr := a.client.readResponseBody(resp)
	resp.Body.Close()
	if readErr != nil {
		restoreOnFailure()
		return nil, readErr
	}

	var failResp struct {
		Reason string `json:"reason"`
	}
	if json.Unmarshal(rawBytes, &failResp) == nil && failResp.Reason != "" {
		restoreOnFailure()
		if failResp.Reason == "maxattempts" {
			return nil, ErrMaxLoginAttempts
		}
		return nil, ErrInvalidCredentials
	}

	me, err := a.Me(ctx)
	if err != nil {
		restoreOnFailure()
		return nil, err
	}
	if !me.ValidUser || !me.Auth.LoggedIn {
		restoreOnFailure()
		return nil, ErrInvalidCredentials
	}

	a.commitAuthenticatedUser(ctx, username, me)
	return me, nil
}

func (a *AuthService) commitAuthenticatedUser(ctx context.Context, username string, me *LoginData) {
	a.client.mu.Lock()
	a.client.username = username
	if me.GUID != "" {
		a.client.guid = me.GUID
	}
	if me.Auth.UID != "" {
		a.client.userUID = me.Auth.UID
	}
	a.client.token = nil
	a.client.mu.Unlock()

	// Exchange the user's GUID for an authenticated Bearer JWT on api.vocabulary.com.
	_, _ = a.EnsureToken(ctx, true)
	_ = a.client.FlushSession(ctx)
}

// Me fetches the current session and subscription status from POST /auth/me.json.
// Notice we intentionally never send the browser honeypot `fp` parameter.
func (a *AuthService) Me(ctx context.Context) (*LoginData, error) {
	var data LoginData
	if err := a.client.doWebForm(ctx, "/auth/me.json", nil, &data); err != nil {
		return nil, err
	}
	a.client.mu.Lock()
	if data.GUID != "" {
		a.client.guid = data.GUID
	}
	if data.Auth.UID != "" {
		a.client.userUID = data.Auth.UID
	}
	a.client.mu.Unlock()
	return &data, nil
}

// Logout terminates the session via GET /auth/logout.json and clears the local CookieJar,
// cached Bearer token, and persistent SessionStore.
func (a *AuthService) Logout(ctx context.Context) error {
	_, _, _ = a.client.doWebGet(ctx, "/auth/logout.json", nil, "application/json")
	a.client.mu.Lock()
	a.client.guid = ""
	a.client.userUID = ""
	a.client.username = ""
	a.client.token = nil
	a.client.mu.Unlock()
	_ = a.client.jar.Reset()
	a.client.ensureTimezoneCookie()
	if a.client.sessionStore != nil {
		return a.client.sessionStore.Clear(ctx)
	}
	return nil
}

// InvalidateToken clears the cached Bearer JWT so the next API request refreshes it.
func (a *AuthService) InvalidateToken() {
	a.client.mu.Lock()
	a.client.token = nil
	a.client.mu.Unlock()
}

// EnsureToken returns a valid Bearer JWT for https://api.vocabulary.com/1.0/...,
// refreshing it via singleflight if missing, expired, or forceRefresh is true.
//
// EnsureToken uses singleflight.DoChan so that if one concurrent caller cancels its context,
// other concurrent callers waiting on the same token refresh are not aborted.
func (a *AuthService) EnsureToken(ctx context.Context, forceRefresh bool) (*TokenInfo, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if !forceRefresh {
		if tok := a.validCachedToken(); tok != nil {
			return tok, nil
		}
	}

	ch := a.client.tokenGroup.DoChan("token", func() (any, error) {
		if !forceRefresh {
			if tok := a.validCachedToken(); tok != nil {
				return tok, nil
			}
		}
		fetchCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 30*time.Second)
		defer cancel()
		return a.fetchToken(fetchCtx, forceRefresh)
	})

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case res := <-ch:
		if res.Err != nil {
			return nil, res.Err
		}
		tok, _ := res.Val.(*TokenInfo)
		return tok.Clone(), nil
	}
}

func (a *AuthService) validCachedToken() *TokenInfo {
	a.client.mu.RLock()
	tok := a.client.token
	guid := a.client.guid
	now := a.client.clock.Now()
	a.client.mu.RUnlock()

	if tok != nil && tok.IsValid(now) && (guid == "" || tok.RefreshToken == guid) {
		return tok.Clone()
	}
	return nil
}

func (a *AuthService) fetchToken(ctx context.Context, forceRefresh bool) (*TokenInfo, error) {
	a.client.mu.RLock()
	guid := a.client.guid
	prevTok := a.client.token
	now := a.client.clock.Now()
	a.client.mu.RUnlock()

	if guid == "" {
		if webU, err := url.Parse(a.client.webBaseURL); err == nil && webU != nil {
			guid = a.client.jar.CookieValue(webU, "guid")
		}
	}

	form := url.Values{}
	if guid != "" {
		form.Set("refresh_token", guid)
	} else if !forceRefresh && prevTok != nil && prevTok.RefreshToken != "" && prevTok.IsValid(now) {
		form.Set("refresh_token", prevTok.RefreshToken)
	}

	tokenURL := a.client.resolveAPIURL("/auth/token")
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Origin", a.client.webBaseURL)
	req.Header.Set("Referer", a.client.webBaseURL+"/")

	resp, err := a.client.do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	rawBytes, err := a.client.readResponseBody(resp)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, a.client.parseAPIError(http.MethodPost, tokenURL, resp, rawBytes)
	}

	var tok TokenInfo
	if err := decodeJSONBody(http.MethodPost, tokenURL, resp, rawBytes, &tok); err != nil {
		return nil, fmt.Errorf("vocab: decode /auth/token response: %w", err)
	}
	if strings.TrimSpace(tok.AccessToken) == "" {
		return nil, fmt.Errorf("%w: /auth/token returned empty access_token", ErrUnexpectedResponse)
	}
	expiresSec := tok.ExpiresIn
	if expiresSec <= 0 {
		expiresSec = 3600
	}
	// Subtract 10 seconds safety margin just like vcom/api in module-zcv3kc.js.
	tok.ExpiresAt = a.client.clock.Now().Add(time.Duration(expiresSec)*time.Second - 10*time.Second)

	a.client.mu.Lock()
	a.client.token = tok.Clone()
	if guid != "" {
		a.client.guid = guid
	}
	a.client.mu.Unlock()

	_ = a.client.FlushSession(ctx)
	return tok.Clone(), nil
}

// ParseJWTClaims decodes the payload section of a Vocabulary.com Bearer JWT for inspection.
func ParseJWTClaims(accessToken string) (map[string]any, error) {
	parts := strings.Split(accessToken, ".")
	if len(parts) < 2 {
		return nil, fmt.Errorf("vocab: invalid JWT format")
	}
	b, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, err
	}
	var claims map[string]any
	if err := json.Unmarshal(b, &claims); err != nil {
		return nil, err
	}
	return claims, nil
}
