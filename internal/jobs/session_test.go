package jobs

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/tejasa97/ytdlp-go/engine"
)

func TestNormalizeBrowserSession(t *testing.T) {
	tests := []struct {
		raw     string
		want    string
		wantErr string
	}{
		{raw: "", want: ""},
		{raw: "off", want: ""},
		{raw: "None", want: ""},
		{raw: "chrome", want: "chrome"},
		{raw: "Chrome:Default", want: "chrome:Default"},
		{raw: "firefox:Work::Shopping", want: "firefox:Work::Shopping"},
		{raw: "chrome:.", wantErr: "invalid"},
		{raw: "chrome:foo/bar", wantErr: "invalid"},
		{raw: "netscape", wantErr: "unsupported"},
		{raw: "chrome:cookie=abc", wantErr: "cookie"},
		{raw: "https://example", wantErr: "browser name"},
	}
	for _, test := range tests {
		got, err := NormalizeBrowserSession(test.raw)
		if test.wantErr != "" {
			if err == nil || !strings.Contains(strings.ToLower(err.Error()), test.wantErr) {
				t.Fatalf("NormalizeBrowserSession(%q) error = %v; want %q", test.raw, err, test.wantErr)
			}
			continue
		}
		if err != nil || got != test.want {
			t.Fatalf("NormalizeBrowserSession(%q) = %q, %v; want %q", test.raw, got, err, test.want)
		}
	}
}

func TestNormalizeCookieFileRejectsContents(t *testing.T) {
	got, err := NormalizeCookieFile(" /tmp/cookies.txt ")
	if err != nil || got != "/tmp/cookies.txt" {
		t.Fatalf("path = %q, %v", got, err)
	}
	if _, err := NormalizeCookieFile("youtube.com\tTRUE\t/\tFALSE\t0\tSID\tabc"); err == nil {
		t.Fatal("cookie bytes were accepted as a path")
	}
}

func TestAuthFailureEnvelopeJSON(t *testing.T) {
	err := authFailureFor(&engine.Error{Category: engine.ErrorAuthentication, Op: "youtube extraction", Err: errors.New("login required")}, browserSession{}, false)
	failure, ok := AsAuthFailure(err)
	if !ok {
		t.Fatalf("AsAuthFailure() = %#v", err)
	}
	if failure.Reason != ReasonSigninRequired || !strings.HasPrefix(failure.Error(), AuthFailurePrefix) {
		t.Fatalf("envelope = %#v error = %q", failure, failure.Error())
	}
	payload := strings.TrimPrefix(failure.Error(), AuthFailurePrefix)
	var decoded AuthFailure
	if jsonErr := json.Unmarshal([]byte(payload), &decoded); jsonErr != nil {
		t.Fatal(jsonErr)
	}
	if decoded.Title != "This video needs your YouTube sign-in." {
		t.Fatalf("title = %q", decoded.Title)
	}
	if strings.Contains(failure.Error(), "SID=") || strings.Contains(payload, "cookie") && strings.Contains(payload, "=") {
		t.Fatalf("cookie material leaked: %s", failure.Error())
	}
}

func TestAnalyzeAlwaysAttachesConfiguredSession(t *testing.T) {
	manager := New(nil, nil)
	t.Cleanup(func() { _ = manager.Close() })
	manager.SetBrowserSession("chrome:Default", "/tmp/cookies.txt")
	var seen engine.Request
	manager.runAnalyze = func(_ context.Context, req engine.Request) (engine.Result, error) {
		seen = req
		return engine.Result{InfoJSON: []byte(`{"id":"fixture0001","title":"Fixture","formats":[{"format_id":"v1","ext":"mp4","vcodec":"avc1","acodec":"none","height":1080,"width":1920},{"format_id":"a1","ext":"m4a","vcodec":"none","acodec":"mp4a","abr":128}]}`)}, nil
	}
	summary, _, err := manager.AnalyzeForAdmission(context.Background(), "https://www.youtube.com/watch?v=fixture0001")
	if err != nil {
		t.Fatal(err)
	}
	if seen.CookiesFromBrowser != "chrome:Default" || seen.CookieFile != "/tmp/cookies.txt" {
		t.Fatalf("attached session = %#v", seen)
	}
	if summary.SessionLabel != "Chrome · Default" {
		t.Fatalf("session label = %q", summary.SessionLabel)
	}
}

func TestAnalyzeSessionUnreadableRetriesOnceSignedOut(t *testing.T) {
	manager := New(nil, nil)
	t.Cleanup(func() { _ = manager.Close() })
	manager.SetBrowserSession("chrome", "")
	var calls []engine.Request
	manager.runAnalyze = func(_ context.Context, req engine.Request) (engine.Result, error) {
		calls = append(calls, req)
		if req.CookiesFromBrowser != "" {
			return engine.Result{}, &engine.Error{Category: engine.ErrorAuthentication, Op: "import browser cookies", Err: errors.New("keychain denied")}
		}
		return engine.Result{InfoJSON: []byte(`{"id":"fixture0001","title":"Public","formats":[{"format_id":"v1","ext":"mp4","vcodec":"avc1","acodec":"none","height":1080,"width":1920},{"format_id":"a1","ext":"m4a","vcodec":"none","acodec":"mp4a","abr":128}]}`)}, nil
	}
	summary, _, err := manager.AnalyzeForAdmission(context.Background(), "https://www.youtube.com/watch?v=fixture0001")
	if err != nil {
		t.Fatal(err)
	}
	if len(calls) != 2 {
		t.Fatalf("calls = %d; want signed-in then signed-out", len(calls))
	}
	if calls[0].CookiesFromBrowser != "chrome" || calls[1].CookiesFromBrowser != "" || calls[1].CookieFile != "" {
		t.Fatalf("retry sequence = %#v", calls)
	}
	if summary.SessionLabel != "" {
		t.Fatalf("signed-out success still labeled %q", summary.SessionLabel)
	}
}

func TestAnalyzeSessionUnreadableStillDeniedReturnsEnvelope(t *testing.T) {
	manager := New(nil, nil)
	t.Cleanup(func() { _ = manager.Close() })
	manager.SetBrowserSession("safari", "")
	manager.runAnalyze = func(_ context.Context, req engine.Request) (engine.Result, error) {
		if req.CookiesFromBrowser != "" {
			return engine.Result{}, &engine.Error{Category: engine.ErrorAuthentication, Op: "parse browser cookie source", Err: errors.New("no such browser")}
		}
		return engine.Result{}, &engine.Error{Category: engine.ErrorAuthentication, Op: "youtube extraction", Err: errors.New("login required")}
	}
	_, _, err := manager.AnalyzeForAdmission(context.Background(), "https://www.youtube.com/watch?v=fixture0001")
	failure, ok := AsAuthFailure(err)
	if !ok || failure.Reason != ReasonSessionUnreadable {
		t.Fatalf("err = %#v", err)
	}
	if failure.Title != "Could not read the browser session." {
		t.Fatalf("title = %q", failure.Title)
	}
}

func TestDownloadRequestUsesConfiguredBrowserSession(t *testing.T) {
	manager := New(nil, nil)
	t.Cleanup(func() { _ = manager.Close() })
	manager.SetBrowserSession("firefox:Work", "")
	started := make(chan engine.Request, 1)
	manager.runDownload = func(ctx context.Context, req engine.Request, _ engine.EventHandler) (engine.Result, error) {
		started <- req
		<-ctx.Done()
		return engine.Result{}, ctx.Err()
	}
	id, err := manager.Submit(Request{URL: "https://example.invalid/video", OutputDir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	select {
	case req := <-started:
		if req.CookiesFromBrowser != "firefox:Work" || req.CookieFile != "" {
			t.Fatalf("download session = %#v", req)
		}
	case <-time.After(time.Second):
		t.Fatal("download runner did not receive a request")
	}
	manager.Cancel(id)
}

func TestSigninRequiredDoesNotRetrySignedOut(t *testing.T) {
	manager := New(nil, nil)
	t.Cleanup(func() { _ = manager.Close() })
	manager.SetBrowserSession("chrome", "")
	calls := 0
	manager.runAnalyze = func(context.Context, engine.Request) (engine.Result, error) {
		calls++
		return engine.Result{}, &engine.Error{Category: engine.ErrorAuthentication, Op: "youtube extraction", Err: errors.New("login required")}
	}
	_, _, err := manager.AnalyzeForAdmission(context.Background(), "https://www.youtube.com/watch?v=fixture0001")
	failure, ok := AsAuthFailure(err)
	if !ok || failure.Reason != ReasonSigninRequired || calls != 1 {
		t.Fatalf("err=%v calls=%d", err, calls)
	}
	if !failure.SessionAttempted || failure.Title != "This video isn't available to your account." {
		t.Fatalf("failure = %#v", failure)
	}
}
