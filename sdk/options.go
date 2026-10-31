package vocab

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"golang.org/x/time/rate"
)

const (
	DefaultWebBaseURL       = "https://www.vocabulary.com"
	DefaultAPIBaseURL       = "https://api.vocabulary.com/1.0"
	DefaultCorpusBaseURL    = "https://corpus.vocabulary.com/api/1.0"
	DefaultAudioBaseURL     = "https://audio.vocabulary.com/1.0/us"
	DefaultVideoBaseURL     = "https://sd-pronunciation-processed-videos.sdcdns.com"
	DefaultUserAgent        = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/142.0.0.0 Safari/537.36"
	DefaultTimezone         = "UTC"
	DefaultMaxResponseBytes = 16 << 20 // 16 MiB

	// Protocol-specific gameplay pacing constants derived from Vocabulary.com's frontend bundles:
	// - vcom/challenge/questions.js: < 900ms first try or < 450ms subsequent try triggers UI countdown blocker.
	// - vcom/quiz/quiz.js: < 1500ms triggers 5-second slowdown blocker.
	// - js3/assets/vocab-trainer.js: 10 answers within 30,000ms triggers a 5-minute cooldown lockout.
	MinChallengeFirstTryMS    int64 = 950
	MinChallengeRetryMS       int64 = 475
	MinQuizSubmitMS           int64 = 1550
	MinBeeSubmitMS            int64 = 550
	TrainerRollingWindow            = 30500 * time.Millisecond
	TrainerMaxAnswersInWindow       = 9
)

// Option configures a Vocabulary.com Client.
type Option func(*Client) error

// WithHTTPClient sets a custom *http.Client (for example, wrapping a go-vcr recorder or custom proxy).
// If the provided client does not have a CookieJar, the SDK attaches its tracked CookieJar automatically.
func WithHTTPClient(hc *http.Client) Option {
	return func(c *Client) error {
		if hc != nil {
			c.httpClient = hc
			c.customHTTPClient = true
		}
		return nil
	}
}

// WithTransport sets the underlying http.RoundTripper on the Client's HTTP client.
func WithTransport(rt http.RoundTripper) Option {
	return func(c *Client) error {
		if rt != nil {
			c.httpClient.Transport = rt
		}
		return nil
	}
}

// WithSessionStore attaches a persistent SessionStore (e.g. FileSessionStore or KeyringSessionStore).
func WithSessionStore(store SessionStore) Option {
	return func(c *Client) error {
		if store != nil {
			c.sessionStore = store
		}
		return nil
	}
}

// WithBaseURLs overrides the default Vocabulary.com host URLs (useful for testing against mock servers).
func WithBaseURLs(webURL, apiURL, corpusURL, audioURL string) Option {
	return func(c *Client) error {
		if webURL != "" {
			c.webBaseURL = webURL
		}
		if apiURL != "" {
			c.apiBaseURL = apiURL
		}
		if corpusURL != "" {
			c.corpusBaseURL = corpusURL
		}
		if audioURL != "" {
			c.audioBaseURL = audioURL
		}
		return nil
	}
}

// WithUserAgent sets a custom User-Agent header.
func WithUserAgent(ua string) Option {
	return func(c *Client) error {
		if ua != "" {
			c.userAgent = ua
		}
		return nil
	}
}

// WithTimezone sets the IANA timezone (e.g. "America/New_York" or "Asia/Kolkata") sent in the
// "tz" cookie and "X-Time-Zone" header on api.vocabulary.com requests.
func WithTimezone(tz string) Option {
	return func(c *Client) error {
		if tz != "" {
			c.timezone = tz
		}
		return nil
	}
}

// WithRateLimit configures the token-bucket rate limiter applied to outgoing requests.
// Pass rps <= 0 or burst <= 0 to disable rate limiting.
func WithRateLimit(rps float64, burst int) Option {
	return func(c *Client) error {
		if rps > 0 && burst > 0 {
			c.limiter = rate.NewLimiter(rate.Limit(rps), burst)
		} else {
			c.limiter = nil
		}
		return nil
	}
}

// WithPacingGuard enables or disables automatic gameplay pacing guards
// (preventing <900ms fast-click blockers in v=3, <1500ms in Quiz, and 10-answers-in-30s cooldowns in v=4).
func WithPacingGuard(enabled bool) Option {
	return func(c *Client) error {
		c.pacingGuard.SetEnabled(enabled)
		return nil
	}
}

// WithClock sets a custom Clock for deterministic testing of pacing guards and token expiration.
func WithClock(clock Clock) Option {
	return func(c *Client) error {
		if clock == nil {
			return fmt.Errorf("vocab: clock cannot be nil")
		}
		c.clock = clock
		c.pacingGuard.SetClock(clock)
		return nil
	}
}

// WithMaxResponseBytes sets the maximum allowed HTTP response body size in bytes (defaults to 16 MiB).
func WithMaxResponseBytes(maxBytes int64) Option {
	return func(c *Client) error {
		if maxBytes <= 0 {
			return fmt.Errorf("vocab: maxResponseBytes must be > 0")
		}
		c.maxResponseBytes = maxBytes
		return nil
	}
}

// WithLogger sets a structured slog.Logger for debug diagnostics (credentials and cookies are redacted).
func WithLogger(logger *slog.Logger) Option {
	return func(c *Client) error {
		if logger != nil {
			c.logger = logger
		}
		return nil
	}
}
