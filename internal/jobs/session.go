package jobs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/tejasa97/ytdlp-go/engine"
)

// Auth failure reason keys returned across the Wails boundary. The frontend
// maps these onto proposal copy; it must not classify from free-text messages.
const (
	ReasonSigninRequired    = "signin-required"
	ReasonSessionUnreadable = "session-unreadable"
	AuthFailurePrefix       = "vidstow:auth:"
)

// AuthFailure is the keyed sign-in error envelope. Cookie values never appear
// here — only the reason, optional browser label, and user-facing copy.
type AuthFailure struct {
	Reason           string `json:"reason"`
	Browser          string `json:"browser,omitempty"`
	Title            string `json:"title"`
	Message          string `json:"message"`
	SessionAttempted bool   `json:"sessionAttempted,omitempty"`
}

func (e *AuthFailure) Error() string {
	if e == nil {
		return AuthFailurePrefix + "{}"
	}
	payload, err := json.Marshal(e)
	if err != nil {
		return AuthFailurePrefix + `{"reason":"` + e.Reason + `"}`
	}
	return AuthFailurePrefix + string(payload)
}

// AsAuthFailure extracts an AuthFailure from err.
func AsAuthFailure(err error) (*AuthFailure, bool) {
	var failure *AuthFailure
	if errors.As(err, &failure) {
		return failure, true
	}
	return nil, false
}

// browserSession holds the manager's future-attempt cookie snapshot. It
// mirrors the FFmpeg location pattern: mutex-held, applied only to new work.
type browserSession struct {
	cookiesFromBrowser string
	cookieFile         string
}

func (s browserSession) configured() bool {
	return strings.TrimSpace(s.cookiesFromBrowser) != "" || strings.TrimSpace(s.cookieFile) != ""
}

func (s browserSession) signedInAttempts() []browserSession {
	browser := strings.TrimSpace(s.cookiesFromBrowser)
	file := strings.TrimSpace(s.cookieFile)
	switch {
	case browser != "" && file != "":
		return []browserSession{{cookiesFromBrowser: browser}, {cookieFile: file}}
	case browser != "":
		return []browserSession{{cookiesFromBrowser: browser}}
	case file != "":
		return []browserSession{{cookieFile: file}}
	default:
		return nil
	}
}

func applyBrowserSession(req *engine.Request, session browserSession) {
	if req == nil {
		return
	}
	req.CookiesFromBrowser = strings.TrimSpace(session.cookiesFromBrowser)
	req.CookieFile = strings.TrimSpace(session.cookieFile)
}

func (s browserSession) label() string {
	if !s.configured() {
		return ""
	}
	if name := SessionLabel(s.cookiesFromBrowser); name != "" {
		return name
	}
	if strings.TrimSpace(s.cookieFile) != "" {
		return "Cookie file"
	}
	return ""
}

// NormalizeBrowserSession validates and normalizes a browser[:profile][::container]
// choice. Empty, "off", and "none" mean signed out. Cookie values are rejected.
func NormalizeBrowserSession(raw string) (string, error) {
	value := strings.TrimSpace(raw)
	if value == "" || strings.EqualFold(value, "off") || strings.EqualFold(value, "none") {
		return "", nil
	}
	if strings.ContainsAny(value, "\n\r\t") || strings.Contains(value, "://") {
		return "", errors.New("browser session must be a browser name and optional profile")
	}
	lower := strings.ToLower(value)
	if strings.Contains(lower, "cookie") && strings.Contains(lower, "=") {
		return "", errors.New("browser session must not contain cookie values")
	}
	browser, rest, hasRest := strings.Cut(value, ":")
	browser = strings.ToLower(strings.TrimSpace(browser))
	switch browser {
	case "chrome", "chromium", "brave", "edge", "vivaldi", "opera", "firefox", "safari":
	default:
		return "", fmt.Errorf("unsupported browser %q", browser)
	}
	if !hasRest {
		return browser, nil
	}
	profile, container, hasContainer := strings.Cut(rest, "::")
	profile = strings.TrimSpace(profile)
	if profile == "" || profile == "." || strings.ContainsAny(profile, `/\`) {
		return "", errors.New("browser profile is invalid")
	}
	if !hasContainer {
		return browser + ":" + profile, nil
	}
	if !strings.EqualFold(browser, "firefox") {
		return "", errors.New("browser containers are only supported with Firefox")
	}
	container = strings.TrimSpace(container)
	if container == "" || strings.ContainsAny(container, `/\`) {
		return "", errors.New("browser container is invalid")
	}
	return browser + ":" + profile + "::" + container, nil
}

// NormalizeCookieFile keeps a path-only cookie fallback. Existence is not
// required at save time — the engine reports missing files at use time.
func NormalizeCookieFile(raw string) (string, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", nil
	}
	if strings.ContainsAny(value, "\n\r\x00") {
		return "", errors.New("cookie file path is invalid")
	}
	if strings.Contains(value, "\t") || (strings.Contains(value, "TRUE") && strings.Contains(value, "FALSE")) {
		return "", errors.New("cookie file must be a path, not cookie contents")
	}
	return value, nil
}

// SessionLabel formats the Home/Queue session line, e.g. "Chrome · Default".
func SessionLabel(spec string) string {
	spec = strings.TrimSpace(spec)
	if spec == "" {
		return ""
	}
	browser, rest, hasRest := strings.Cut(spec, ":")
	name := browserDisplayName(browser)
	if !hasRest {
		return name
	}
	profile, _, _ := strings.Cut(rest, "::")
	profile = strings.TrimSpace(profile)
	if profile == "" {
		return name
	}
	return name + " · " + profile
}

func browserFamilyName(spec string) string {
	spec = strings.TrimSpace(spec)
	if spec == "" {
		return ""
	}
	browser, _, _ := strings.Cut(spec, ":")
	browser = strings.ToLower(strings.TrimSpace(browser))
	if browser == "" || browser == "off" || browser == "none" {
		return ""
	}
	return browserDisplayName(browser)
}

func sessionFromJob(state *jobState) browserSession {
	if state == nil {
		return browserSession{}
	}
	return browserSession{
		cookiesFromBrowser: strings.TrimSpace(state.durable.Request.BrowserSession),
		cookieFile:         strings.TrimSpace(state.durable.Request.CookieFile),
	}
}

func sessionUnreadableCopy(session browserSession) (title, message string) {
	if name := browserFamilyName(session.cookiesFromBrowser); name != "" {
		return "Couldn't use " + name + ".", "Close " + name + ", then retry."
	}
	if strings.TrimSpace(session.cookieFile) != "" {
		return "Couldn't use the cookie file.", "Choose another file, then retry."
	}
	return "Couldn't use the browser.", "Close it, then retry."
}

func browserDisplayName(browser string) string {
	switch strings.ToLower(strings.TrimSpace(browser)) {
	case "chrome", "chromium":
		return "Chrome"
	case "firefox":
		return "Firefox"
	case "safari":
		return "Safari"
	case "edge":
		return "Edge"
	case "brave":
		return "Brave"
	case "vivaldi":
		return "Vivaldi"
	case "opera":
		return "Opera"
	default:
		return strings.TrimSpace(browser)
	}
}

func isSessionUnreadableError(err error) bool {
	var typed *engine.Error
	if !errors.As(err, &typed) {
		return false
	}
	if typed.Category != engine.ErrorAuthentication {
		return false
	}
	switch typed.Op {
	case "parse browser cookie source", "import browser cookies", "load browser cookies", "load cookie file":
		return true
	default:
		return false
	}
}

func isSigninRequiredError(err error) bool {
	var typed *engine.Error
	if !errors.As(err, &typed) {
		return false
	}
	return typed.Category == engine.ErrorAuthentication && !isSessionUnreadableError(err)
}

func authFailureFor(err error, session browserSession, attempted bool) error {
	browser := SessionLabel(session.cookiesFromBrowser)
	if isSessionUnreadableError(err) {
		title, message := sessionUnreadableCopy(session)
		return &AuthFailure{
			Reason:           ReasonSessionUnreadable,
			Browser:          browser,
			Title:            title,
			Message:          message,
			SessionAttempted: attempted,
		}
	}
	if !isSigninRequiredError(err) {
		return err
	}
	if attempted && session.configured() {
		return &AuthFailure{
			Reason:           ReasonSigninRequired,
			Browser:          browser,
			Title:            "This video isn't available to your account.",
			Message:          "Your browser session tried, and YouTube still said no. Double-check the link, or sign in somewhere else and update Settings.",
			SessionAttempted: true,
		}
	}
	return &AuthFailure{
		Reason:  ReasonSigninRequired,
		Browser: browser,
		Title:   "This video needs your YouTube sign-in.",
		Message: "Sign in to YouTube in your browser, pick that browser in Settings, then try again. Only videos your account can already watch.",
	}
}

// playlistAuthFailure keeps session-unreadable copy as-is and rewrites
// sign-in-required titles so playlist Analyze talks about the playlist,
// not a single video. Child AnalyzeForAdmission still uses video copy.
func playlistAuthFailure(err error) error {
	failure, ok := AsAuthFailure(err)
	if !ok || failure.Reason != ReasonSigninRequired {
		return err
	}
	rewritten := *failure
	if failure.SessionAttempted {
		rewritten.Title = "This playlist isn't available to your account."
		rewritten.Message = "Your browser session tried, and YouTube still said no. Double-check the link, or sign in somewhere else and update Settings."
	} else {
		rewritten.Title = "This playlist needs your YouTube sign-in."
		rewritten.Message = "Sign in to YouTube in your browser, pick that browser in Settings, then try again. Only playlists your account can already open."
	}
	return &rewritten
}

// SetBrowserSession updates the cookie snapshot for future Analyze attempts
// and for in-memory downloads that are not State v2 jobs. Admitted downloads
// reuse the browser/cookie-file choice stored on the job.
func (m *Manager) SetBrowserSession(cookiesFromBrowser, cookieFile string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closing || m.closed {
		return
	}
	m.session = browserSession{
		cookiesFromBrowser: strings.TrimSpace(cookiesFromBrowser),
		cookieFile:         strings.TrimSpace(cookieFile),
	}
}

func (m *Manager) browserSessionSnapshot() browserSession {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.session
}

func (m *Manager) sessionForDownload(state *jobState) browserSession {
	if state != nil && state.fromStateV2 {
		return browserSession{
			cookiesFromBrowser: strings.TrimSpace(state.durable.Request.BrowserSession),
			cookieFile:         strings.TrimSpace(state.durable.Request.CookieFile),
		}
	}
	return m.browserSessionSnapshot()
}

// runAnalyzeWithSession attaches a configured session. Browser import runs
// first; a cookie file is a fallback only when the browser store cannot be
// read. If every signed-in source is unreadable, it retries once signed out
// so public videos still work.
func (m *Manager) runAnalyzeWithSession(ctx context.Context, req engine.Request, runner analyzeRunner) (engine.Result, browserSession, error) {
	session := m.browserSessionSnapshot()
	var lastUnreadable error
	for _, attempt := range session.signedInAttempts() {
		next := req
		applyBrowserSession(&next, attempt)
		result, err := runner(ctx, next)
		if err == nil {
			return result, attempt, nil
		}
		if isSessionUnreadableError(err) {
			lastUnreadable = err
			continue
		}
		if isSigninRequiredError(err) {
			return engine.Result{}, session, authFailureFor(err, attempt, true)
		}
		return engine.Result{}, session, err
	}
	unsigned := req
	applyBrowserSession(&unsigned, browserSession{})
	if lastUnreadable != nil {
		retry, retryErr := runner(ctx, unsigned)
		if retryErr == nil {
			return retry, browserSession{}, nil
		}
		return engine.Result{}, session, authFailureFor(lastUnreadable, session, true)
	}
	result, err := runner(ctx, unsigned)
	if err == nil {
		return result, session, nil
	}
	if isSigninRequiredError(err) || isSessionUnreadableError(err) {
		return engine.Result{}, session, authFailureFor(err, session, false)
	}
	return engine.Result{}, session, err
}

// runDownloadWithSession attaches the job's stored session (or the live
// Settings snapshot for in-memory jobs). Cookie-file fallback and one
// signed-out retry match Analyze.
func (m *Manager) runDownloadWithSession(ctx context.Context, req engine.Request, handler engine.EventHandler, runner downloadRunner, session browserSession) (engine.Result, error) {
	var lastUnreadable error
	for _, attempt := range session.signedInAttempts() {
		next := req
		applyBrowserSession(&next, attempt)
		result, err := runner(ctx, next, handler)
		if err == nil {
			return result, nil
		}
		if isSessionUnreadableError(err) {
			lastUnreadable = err
			continue
		}
		if isSigninRequiredError(err) {
			return engine.Result{}, authFailureFor(err, attempt, true)
		}
		return engine.Result{}, err
	}
	unsigned := req
	applyBrowserSession(&unsigned, browserSession{})
	if lastUnreadable != nil {
		retry, retryErr := runner(ctx, unsigned, handler)
		if retryErr == nil {
			return retry, nil
		}
		return engine.Result{}, authFailureFor(lastUnreadable, session, true)
	}
	result, err := runner(ctx, unsigned, handler)
	if err == nil {
		return result, nil
	}
	if isSigninRequiredError(err) || isSessionUnreadableError(err) {
		return engine.Result{}, authFailureFor(err, session, session.configured())
	}
	return engine.Result{}, err
}
