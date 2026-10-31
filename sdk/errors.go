package vocab

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

var (
	// ErrInvalidCredentials indicates the username or password was rejected by Vocabulary.com.
	ErrInvalidCredentials = errors.New("vocab: invalid username or password")

	// ErrMaxLoginAttempts indicates Vocabulary.com throttled login attempts ("maxattempts").
	ErrMaxLoginAttempts = errors.New("vocab: maximum login attempts exceeded")

	// ErrSessionExpired indicates the session is unauthenticated or has expired.
	ErrSessionExpired = errors.New("vocab: session expired or unauthenticated")

	// ErrTrialOver indicates the unauthenticated/free question limit has been reached ("trialover").
	ErrTrialOver = errors.New("vocab: trial over or subscription required")

	// ErrRestartChallenge indicates the server returned RestartChallengeException.
	ErrRestartChallenge = errors.New("vocab: challenge state out of sync (RestartChallengeException)")

	// ErrConcurrentPlay indicates the server returned ConcurrentPlayException.
	ErrConcurrentPlay = errors.New("vocab: concurrent play detected (ConcurrentPlayException)")

	// ErrCooldownActive indicates the rolling 10-questions-in-30s cooldown guard is active.
	ErrCooldownActive = errors.New("vocab: trainer cooldown active (too many questions answered within 30s)")

	// ErrNotFound indicates the requested word, list, or unit does not exist.
	ErrNotFound = errors.New("vocab: resource not found")

	// ErrRateLimited indicates the server throttled the request (HTTP 429 or rate-limit response).
	ErrRateLimited = errors.New("vocab: rate limited by server")

	// ErrResponseTooLarge indicates the HTTP response body exceeded the client's configured size limit.
	ErrResponseTooLarge = errors.New("vocab: response body exceeded maximum allowed size")

	// ErrUnexpectedResponse indicates the server returned unexpected or malformed content (e.g. HTML instead of JSON).
	ErrUnexpectedResponse = errors.New("vocab: unexpected server response format")

	// ErrCorruptedSession indicates the persisted session data is corrupted and cannot be decoded.
	ErrCorruptedSession = errors.New("vocab: corrupted session snapshot")

	// ErrInvalidStateTransition indicates a TrainerSession method was called in an invalid state.
	ErrInvalidStateTransition = errors.New("vocab: invalid trainer session state transition")
)

const maxErrorBodyPreview = 256

// APIError represents a non-2xx HTTP response or structured JSON error from Vocabulary.com.
type APIError struct {
	StatusCode  int
	Method      string
	URL         string
	ErrorCode   string
	Message     string
	ContentType string
	RetryAfter  time.Duration
	RawBody     string
	Cause       error
}

func (e *APIError) Error() string {
	if e == nil {
		return "<nil>"
	}
	var suffix string
	if e.RetryAfter > 0 {
		suffix = fmt.Sprintf(" [retry-after=%s]", e.RetryAfter)
	}
	if e.ErrorCode != "" && e.Message != "" {
		return fmt.Sprintf("vocab: %s %s returned HTTP %d (%s: %s)%s", e.Method, e.URL, e.StatusCode, e.ErrorCode, e.Message, suffix)
	}
	if e.ErrorCode != "" {
		return fmt.Sprintf("vocab: %s %s returned HTTP %d (%s)%s", e.Method, e.URL, e.StatusCode, e.ErrorCode, suffix)
	}
	if e.Message != "" {
		return fmt.Sprintf("vocab: %s %s returned HTTP %d: %s%s", e.Method, e.URL, e.StatusCode, e.Message, suffix)
	}
	return fmt.Sprintf("vocab: %s %s returned HTTP %d%s", e.Method, e.URL, e.StatusCode, suffix)
}

// BodyPreview returns a truncated, single-line preview of RawBody safe for diagnostic messages.
func (e *APIError) BodyPreview() string {
	if e == nil || e.RawBody == "" {
		return ""
	}
	s := strings.Join(strings.Fields(e.RawBody), " ")
	if len(s) > maxErrorBodyPreview {
		return s[:maxErrorBodyPreview] + "..."
	}
	return s
}

func (e *APIError) Unwrap() error {
	if e == nil {
		return nil
	}
	if e.Cause != nil {
		return e.Cause
	}
	switch e.ErrorCode {
	case "RestartChallengeException":
		return ErrRestartChallenge
	case "ConcurrentPlayException":
		return ErrConcurrentPlay
	case "maxattempts":
		return ErrMaxLoginAttempts
	}
	switch e.StatusCode {
	case http.StatusUnauthorized, http.StatusForbidden:
		return ErrSessionExpired
	case http.StatusNotFound:
		return ErrNotFound
	case http.StatusTooManyRequests:
		return ErrRateLimited
	}
	return nil
}

// parseRetryAfter parses an HTTP Retry-After header value (either delta-seconds or HTTP-date).
func parseRetryAfter(val string, now time.Time) time.Duration {
	val = strings.TrimSpace(val)
	if val == "" {
		return 0
	}
	if secs, err := strconv.Atoi(val); err == nil {
		if secs <= 0 {
			return 0
		}
		return time.Duration(secs) * time.Second
	}
	if t, err := http.ParseTime(val); err == nil {
		if d := t.Sub(now); d > 0 {
			return d
		}
	}
	return 0
}
