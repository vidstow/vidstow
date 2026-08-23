package jobs

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/tejasa97/vidstow/internal/authsource"
	"github.com/tejasa97/vidstow/internal/jobmodel"
	"github.com/tejasa97/vidstow/internal/outputplan"
	"github.com/tejasa97/ytdlp-go/engine"
)

func installAuthBindings(store *v2MemoryStore, defaultRef string) {
	store.mu.Lock()
	defer store.mu.Unlock()
	store.state.AuthSourceBindings = []authsource.Binding{
		{ID: "chrome-binding", Enabled: true, Descriptor: authsource.Descriptor{SchemaVersion: 1, Platform: "darwin", Browser: authsource.BrowserChrome}},
		{ID: "safari-binding", Enabled: true, Descriptor: authsource.Descriptor{SchemaVersion: 1, Platform: "darwin", Browser: authsource.BrowserSafari}},
	}
	store.state.Settings.BrowserAccessEnabled = true
	store.state.Settings.BrowserAccessConsentVersion = authsource.CurrentConsentVersion
	store.state.Settings.DefaultAuthSourceBindingRef = defaultRef
}

func analysisFixtureResult() engine.Result {
	return engine.Result{InfoJSON: []byte(`{"id":"fixture0001","title":"Fixture","formats":[{"format_id":"v1","ext":"mp4","vcodec":"avc1","acodec":"none","height":1080,"width":1920},{"format_id":"a1","ext":"m4a","vcodec":"none","acodec":"mp4a","abr":128}]}`)}
}

func installDarwinBrowserSpecResolver(manager *Manager) {
	manager.resolveBrowserSpec = func(descriptor authsource.Descriptor) (string, error) {
		switch descriptor {
		case (authsource.Descriptor{SchemaVersion: authsource.DescriptorSchemaVersion, Platform: "darwin", Browser: authsource.BrowserChrome}):
			return "chrome", nil
		case (authsource.Descriptor{SchemaVersion: authsource.DescriptorSchemaVersion, Platform: "darwin", Browser: authsource.BrowserSafari}):
			return "safari", nil
		default:
			return "", authsource.NewError("invalid-source")
		}
	}
}

func TestAuthenticatedAnalysisInjectsExactSourceOnceAndBindsAuthority(t *testing.T) {
	store, _, _ := newV2TestStore(t)
	// The Settings default deliberately differs from the explicit analysis
	// binding. It must never be substituted.
	installAuthBindings(store, "safari-binding")
	manager := New(nil, nil)
	installDarwinBrowserSpecResolver(manager)
	t.Cleanup(func() { _ = manager.Close() })
	if err := manager.SetStateStore(store); err != nil {
		t.Fatal(err)
	}
	requests := []engine.Request{}
	manager.cachePlans("fixture0001", []outputplan.Plan{{ID: "stale-public-plan"}})
	manager.runAnalyze = func(_ context.Context, request engine.Request) (engine.Result, error) {
		requests = append(requests, request)
		return analysisFixtureResult(), nil
	}
	summary, err := manager.AnalyzeAuthenticated(context.Background(), "https://www.youtube.com/watch?v=fixture0001", "chrome-binding")
	if err != nil {
		t.Fatal(err)
	}
	if len(requests) != 1 || requests[0].CookiesFromBrowser != "chrome" || requests[0].CookieFile != "" || !requests[0].ScopeBrowserCookiesToURL {
		t.Fatalf("authenticated requests = %#v", requests)
	}
	if summary.BrowserAccess.Mode != "browser-session" || summary.BrowserAccess.Label != "Chrome — Default" || summary.AnalysisAuthority == "" {
		t.Fatalf("authenticated summary = %#v", summary)
	}
	if _, err := manager.ResolvePlan("fixture0001", "stale-public-plan"); err == nil {
		t.Fatal("authenticated analysis left a public admission cache usable")
	}
	plan, intent, err := manager.ResolveAnalysisAuthority(summary.AnalysisAuthority, summary.URL, summary.VideoID, summary.Plans[0].ID)
	if err != nil || plan.ID == "" || !intent.RequiresAuthenticatedExecution || intent.AuthSourceBindingRef != "chrome-binding" {
		t.Fatalf("authority = %#v, %#v, %v", plan, intent, err)
	}
}

func TestForgetWaitsForInFlightAuthenticatedAnalysis(t *testing.T) {
	store, _, _ := newV2TestStore(t)
	installAuthBindings(store, "chrome-binding")
	manager := New(nil, nil)
	installDarwinBrowserSpecResolver(manager)
	t.Cleanup(func() { _ = manager.Close() })
	if err := manager.SetStateStore(store); err != nil {
		t.Fatal(err)
	}
	started := make(chan struct{})
	release := make(chan struct{})
	manager.runAnalyze = func(_ context.Context, request engine.Request) (engine.Result, error) {
		if request.CookiesFromBrowser != "chrome" || !request.ScopeBrowserCookiesToURL {
			t.Fatalf("authenticated request = %#v", request)
		}
		close(started)
		<-release
		return analysisFixtureResult(), nil
	}
	result := make(chan error, 1)
	go func() {
		_, err := manager.AnalyzeAuthenticated(context.Background(), "https://www.youtube.com/watch?v=fixture0001", "chrome-binding")
		result <- err
	}()
	<-started
	preview, err := manager.PreviewForgetAuthSource("chrome-binding")
	if err != nil || preview.Active != 1 {
		t.Fatalf("active operation preview = %#v, %v", preview, err)
	}
	if _, err := manager.ForgetAuthSource("chrome-binding"); err == nil || !strings.Contains(err.Error(), "active browser-source operations") {
		t.Fatalf("forget during analysis error = %v", err)
	}
	close(release)
	if err := <-result; err != nil {
		t.Fatal(err)
	}
	if _, err := manager.ForgetAuthSource("chrome-binding"); err != nil {
		t.Fatalf("forget after analysis: %v", err)
	}
}

func TestRendererAnalysisAuthorityInvalidatesOnAccessOrSourceChange(t *testing.T) {
	store, _, _ := newV2TestStore(t)
	installAuthBindings(store, "safari-binding")
	manager := New(nil, nil)
	installDarwinBrowserSpecResolver(manager)
	t.Cleanup(func() { _ = manager.Close() })
	if err := manager.SetStateStore(store); err != nil {
		t.Fatal(err)
	}
	manager.runAnalyze = func(_ context.Context, _ engine.Request) (engine.Result, error) {
		return analysisFixtureResult(), nil
	}
	const rawURL = "https://www.youtube.com/watch?v=fixture0001"
	publicSummary, err := manager.Analyze(context.Background(), rawURL)
	if err != nil {
		t.Fatal(err)
	}
	chromeSummary, err := manager.AnalyzeAuthenticated(context.Background(), rawURL, "chrome-binding")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := manager.ResolveAnalysisAuthority(publicSummary.AnalysisAuthority, publicSummary.URL, publicSummary.VideoID, publicSummary.Plans[0].ID); err == nil {
		t.Fatal("public authority survived browser-session analysis")
	}
	safariSummary, err := manager.AnalyzeAuthenticated(context.Background(), rawURL, "safari-binding")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := manager.ResolveAnalysisAuthority(chromeSummary.AnalysisAuthority, chromeSummary.URL, chromeSummary.VideoID, chromeSummary.Plans[0].ID); err == nil {
		t.Fatal("old browser-source authority survived source change")
	}
	latestPublic, err := manager.Analyze(context.Background(), rawURL)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := manager.ResolveAnalysisAuthority(safariSummary.AnalysisAuthority, safariSummary.URL, safariSummary.VideoID, safariSummary.Plans[0].ID); err == nil {
		t.Fatal("browser-session authority survived explicit public analysis")
	}
	if _, _, err := manager.ResolveAnalysisAuthority("", latestPublic.URL, latestPublic.VideoID, latestPublic.Plans[0].ID); err == nil {
		t.Fatal("omitted analysis authority was accepted")
	}
	if _, intent, err := manager.ResolveAnalysisAuthority(latestPublic.AnalysisAuthority, latestPublic.URL, latestPublic.VideoID, latestPublic.Plans[0].ID); err != nil || intent.RequiresAuthenticatedExecution {
		t.Fatalf("latest public authority = %#v, %v", intent, err)
	}
}

func TestForgottenBindingCannotReviveCachedAnalysisAfterFreshReconfiguration(t *testing.T) {
	store, _, _ := newV2TestStore(t)
	installAuthBindings(store, "chrome-binding")
	manager := New(nil, nil)
	installDarwinBrowserSpecResolver(manager)
	t.Cleanup(func() { _ = manager.Close() })
	if err := manager.SetStateStore(store); err != nil {
		t.Fatal(err)
	}
	manager.runAnalyze = func(_ context.Context, request engine.Request) (engine.Result, error) {
		if request.CookiesFromBrowser != "chrome" || !request.ScopeBrowserCookiesToURL {
			t.Fatalf("authenticated request = %#v", request)
		}
		return analysisFixtureResult(), nil
	}
	const rawURL = "https://www.youtube.com/watch?v=fixture0001"
	oldSummary, err := manager.AnalyzeAuthenticated(context.Background(), rawURL, "chrome-binding")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := manager.ForgetAuthSource("chrome-binding"); err != nil {
		t.Fatal(err)
	}
	store.mu.Lock()
	store.state.AuthSourceBindings = append(store.state.AuthSourceBindings, authsource.Binding{
		ID: "chrome-binding-fresh", Enabled: true,
		Descriptor: authsource.Descriptor{SchemaVersion: authsource.DescriptorSchemaVersion, Platform: "darwin", Browser: authsource.BrowserChrome},
	})
	store.state.Settings.BrowserAccessEnabled = true
	store.state.Settings.BrowserAccessConsentVersion = authsource.CurrentConsentVersion
	store.state.Settings.DefaultAuthSourceBindingRef = "chrome-binding-fresh"
	store.mu.Unlock()

	if _, _, err := manager.ResolveAnalysisAuthority(oldSummary.AnalysisAuthority, oldSummary.URL, oldSummary.VideoID, oldSummary.Plans[0].ID); err == nil {
		t.Fatal("forgotten binding's cached authority revived after same-browser reconfiguration")
	}
	freshSummary, err := manager.AnalyzeAuthenticated(context.Background(), rawURL, "chrome-binding-fresh")
	if err != nil {
		t.Fatal(err)
	}
	if _, intent, err := manager.ResolveAnalysisAuthority(freshSummary.AnalysisAuthority, freshSummary.URL, freshSummary.VideoID, freshSummary.Plans[0].ID); err != nil || intent.AuthSourceBindingRef != "chrome-binding-fresh" {
		t.Fatalf("fresh authority = %#v, %v", intent, err)
	}
}

func TestOlderConcurrentAnalysisCannotReplaceNewerSelection(t *testing.T) {
	store, _, _ := newV2TestStore(t)
	installAuthBindings(store, "chrome-binding")
	manager := New(nil, nil)
	installDarwinBrowserSpecResolver(manager)
	t.Cleanup(func() { _ = manager.Close() })
	if err := manager.SetStateStore(store); err != nil {
		t.Fatal(err)
	}
	publicStarted := make(chan struct{})
	releasePublic := make(chan struct{})
	manager.runAnalyze = func(_ context.Context, request engine.Request) (engine.Result, error) {
		if request.CookiesFromBrowser == "" {
			if request.ScopeBrowserCookiesToURL {
				t.Fatalf("public request enabled browser-cookie scoping: %#v", request)
			}
			close(publicStarted)
			<-releasePublic
		} else if !request.ScopeBrowserCookiesToURL {
			t.Fatalf("authenticated request omitted browser-cookie scoping: %#v", request)
		}
		return analysisFixtureResult(), nil
	}
	const rawURL = "https://www.youtube.com/watch?v=fixture0001"
	publicResult := make(chan error, 1)
	go func() {
		_, err := manager.Analyze(context.Background(), rawURL)
		publicResult <- err
	}()
	<-publicStarted
	authenticated, err := manager.AnalyzeAuthenticated(context.Background(), rawURL, "chrome-binding")
	if err != nil {
		t.Fatal(err)
	}
	close(releasePublic)
	if err := <-publicResult; err == nil {
		t.Fatal("older analysis completion was not superseded")
	}
	if _, intent, err := manager.ResolveAnalysisAuthority(authenticated.AnalysisAuthority, authenticated.URL, authenticated.VideoID, authenticated.Plans[0].ID); err != nil || intent.AuthSourceBindingRef != "chrome-binding" {
		t.Fatalf("newer authority = %#v, %v", intent, err)
	}
}

func TestAuthenticatedManagerRejectsUnsupportedPlatformResolutionWithoutEngineFallback(t *testing.T) {
	store, _, _ := newV2TestStore(t)
	installAuthBindings(store, "chrome-binding")
	manager := New(nil, nil)
	manager.resolveBrowserSpec = func(authsource.Descriptor) (string, error) {
		return "", authsource.NewError("unsupported-platform")
	}
	t.Cleanup(func() { _ = manager.Close() })
	if err := manager.SetStateStore(store); err != nil {
		t.Fatal(err)
	}
	calls := 0
	manager.runAnalyze = func(context.Context, engine.Request) (engine.Result, error) {
		calls++
		return analysisFixtureResult(), nil
	}
	if _, err := manager.AnalyzeAuthenticated(context.Background(), "https://www.youtube.com/watch?v=fixture0001", "chrome-binding"); err == nil {
		t.Fatal("unsupported browser source execution succeeded")
	}
	if calls != 0 {
		t.Fatalf("unsupported source reached engine %d times", calls)
	}
}

func TestAuthenticatedAnalysisFailureHasNoCookieFreeFallback(t *testing.T) {
	store, _, _ := newV2TestStore(t)
	installAuthBindings(store, "chrome-binding")
	manager := New(nil, nil)
	installDarwinBrowserSpecResolver(manager)
	t.Cleanup(func() { _ = manager.Close() })
	if err := manager.SetStateStore(store); err != nil {
		t.Fatal(err)
	}
	calls := 0
	manager.runAnalyze = func(_ context.Context, request engine.Request) (engine.Result, error) {
		calls++
		if request.CookiesFromBrowser != "chrome" || request.CookieFile != "" || !request.ScopeBrowserCookiesToURL {
			t.Fatalf("request = %#v", request)
		}
		return engine.Result{}, &engine.Error{Category: engine.ErrorAuthentication, Err: errors.New("KEYCHAIN_CANARY")}
	}
	if _, err := manager.AnalyzeAuthenticated(context.Background(), "https://www.youtube.com/watch?v=fixture0001", "chrome-binding"); err == nil {
		t.Fatal("authenticated analysis unexpectedly succeeded")
	}
	if calls != 1 {
		t.Fatalf("engine calls = %d, want exactly one cookie-configured attempt", calls)
	}
}

func TestPublicAnalysisRemainsAnonymous(t *testing.T) {
	store, _, _ := newV2TestStore(t)
	installAuthBindings(store, "chrome-binding")
	manager := New(nil, nil)
	installDarwinBrowserSpecResolver(manager)
	t.Cleanup(func() { _ = manager.Close() })
	if err := manager.SetStateStore(store); err != nil {
		t.Fatal(err)
	}
	manager.runAnalyze = func(_ context.Context, request engine.Request) (engine.Result, error) {
		if request.CookiesFromBrowser != "" || request.CookieFile != "" || request.ScopeBrowserCookiesToURL {
			t.Fatalf("public request inspected browser authority: %#v", request)
		}
		return analysisFixtureResult(), nil
	}
	summary, err := manager.Analyze(context.Background(), "https://www.youtube.com/watch?v=fixture0001")
	if err != nil || summary.BrowserAccess.Mode != "public" || summary.AnalysisAuthority == "" {
		t.Fatalf("public summary = %#v, %v", summary, err)
	}
}

func TestAuthenticatedDownloadFailureBecomesActionRequiredWithoutFallback(t *testing.T) {
	store, root, plan := newV2TestStore(t, "job-auth")
	installAuthBindings(store, "safari-binding")
	store.mu.Lock()
	store.state.Jobs[0].AuthIntent = jobmodel.AuthIntent{RequiresAuthenticatedExecution: true, AuthSourceBindingRef: "chrome-binding"}
	store.mu.Unlock()
	manager := New(nil, nil)
	installDarwinBrowserSpecResolver(manager)
	t.Cleanup(func() { _ = manager.Close() })
	if err := manager.SetStateStore(store); err != nil {
		t.Fatal(err)
	}
	calls := 0
	manager.runDownload = func(_ context.Context, request engine.Request, handler engine.EventHandler) (engine.Result, error) {
		calls++
		if request.CookiesFromBrowser != "chrome" || request.CookieFile != "" || !request.ScopeBrowserCookiesToURL {
			t.Fatalf("download request = %#v", request)
		}
		if err := handler(context.Background(), engine.Event{Kind: engine.EventBrowserCookies, Message: "imported 7 of 9 browser cookies"}); err != nil {
			t.Fatal(err)
		}
		return engine.Result{}, &engine.Error{Category: engine.ErrorAuthentication, Err: errors.New("DB_ROW_CANARY")}
	}
	if _, err := manager.SubmitAdmitted("job-auth", Request{URL: "https://www.youtube.com/watch?v=abc123", VideoID: "abc123", Title: "Demo", PlanID: plan.ID, OutputDir: root}, &plan, AdmittedOutput{Basename: "Demo [abc123] [1080p].mp4"}); err != nil {
		t.Fatal(err)
	}
	job := waitForV2Job(t, store, "job-auth", jobmodel.LifecycleActionRequired)
	if calls != 1 || job.ActionRequiredCode != authenticatedSessionUnavailableCode {
		t.Fatalf("calls=%d job=%#v", calls, job)
	}
	snapshot, _ := manager.Find("job-auth")
	if snapshot.Message == "imported 7 of 9 browser cookies" || snapshot.ErrorReason != authenticatedSessionUnavailableCode {
		t.Fatalf("renderer snapshot leaked browser event or wrong recovery: %#v", snapshot)
	}
	view := manager.QueueView()
	review, err := manager.QueueActionRequiredReview("job-auth", view.Rows[0].CommandToken)
	if err != nil || !review.CanRetryFreshLink || review.RetryFreshLabel != "Refresh Chrome — Default and retry" {
		t.Fatalf("available bound-source review = %#v, %v", review, err)
	}
}

func TestInitialAuthenticatedDownloadSucceedsWithoutLoginAttestation(t *testing.T) {
	store, root, plan := newV2TestStore(t, "job-auth-success")
	installAuthBindings(store, "safari-binding")
	store.mu.Lock()
	store.state.Jobs[0].AuthIntent = jobmodel.AuthIntent{RequiresAuthenticatedExecution: true, AuthSourceBindingRef: "chrome-binding"}
	store.mu.Unlock()
	manager := New(nil, nil)
	installDarwinBrowserSpecResolver(manager)
	t.Cleanup(func() { _ = manager.Close() })
	if err := manager.SetStateStore(store); err != nil {
		t.Fatal(err)
	}
	calls := 0
	manager.runDownload = func(_ context.Context, request engine.Request, _ engine.EventHandler) (engine.Result, error) {
		calls++
		if request.CookiesFromBrowser != "chrome" || request.CookieFile != "" || !request.ScopeBrowserCookiesToURL {
			t.Fatalf("download request = %#v", request)
		}
		return engine.Result{Filename: filepath.Join(root, "Demo [abc123] [1080p].mp4")}, nil
	}
	if _, err := manager.SubmitAdmitted("job-auth-success", Request{URL: "https://www.youtube.com/watch?v=abc123", VideoID: "abc123", Title: "Demo", PlanID: plan.ID, OutputDir: root}, &plan, AdmittedOutput{Basename: "Demo [abc123] [1080p].mp4"}); err != nil {
		t.Fatal(err)
	}
	waitForV2Job(t, store, "job-auth-success", jobmodel.LifecycleCompleted)
	if calls != 1 {
		t.Fatalf("engine calls = %d, want one cookie-configured success", calls)
	}
}

func TestEngineTextCanariesNeverReachPresentationOrState(t *testing.T) {
	const canary = "COOKIE_CANARY_signed_url_profile_path"
	store, root, plan := newV2TestStore(t, "job-auth-canary")
	installAuthBindings(store, "chrome-binding")
	store.mu.Lock()
	store.state.Jobs[0].AuthIntent = jobmodel.AuthIntent{RequiresAuthenticatedExecution: true, AuthSourceBindingRef: "chrome-binding"}
	store.mu.Unlock()
	var eventMu sync.Mutex
	events := []Event{}
	manager := New(nil, func(event Event) {
		eventMu.Lock()
		events = append(events, event)
		eventMu.Unlock()
	})
	installDarwinBrowserSpecResolver(manager)
	if err := manager.SetStateStore(store); err != nil {
		t.Fatal(err)
	}
	manager.runDownload = func(_ context.Context, request engine.Request, handler engine.EventHandler) (engine.Result, error) {
		if request.CookiesFromBrowser != "chrome" || !request.ScopeBrowserCookiesToURL {
			t.Fatalf("request = %#v", request)
		}
		if err := handler(context.Background(), engine.Event{Kind: "future-provider-event", Message: canary}); err != nil {
			t.Fatal(err)
		}
		return engine.Result{}, errors.New(canary)
	}
	if _, err := manager.SubmitAdmitted("job-auth-canary", Request{URL: "https://www.youtube.com/watch?v=abc123", VideoID: "abc123", Title: "Demo", PlanID: plan.ID, OutputDir: root}, &plan, AdmittedOutput{Basename: "Demo [abc123] [1080p].mp4"}); err != nil {
		t.Fatal(err)
	}
	waitForV2Job(t, store, "job-auth-canary", jobmodel.LifecycleFailed)
	snapshot, _ := manager.Find("job-auth-canary")
	view := manager.QueueView()
	if err := manager.Close(); err != nil {
		t.Fatal(err)
	}
	eventMu.Lock()
	capturedEvents := append([]Event(nil), events...)
	eventMu.Unlock()
	for name, value := range map[string]any{
		"State v2": store.Snapshot(), "JobSnapshot": snapshot, "QueueView": view, "Wails events": capturedEvents,
	} {
		raw, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(raw), canary) {
			t.Fatalf("%s leaked engine canary: %s", name, raw)
		}
	}
	if snapshot.Message != "Download failed" || snapshot.ErrorReason != "internal" {
		t.Fatalf("safe terminal projection = %#v", snapshot)
	}
}

func TestDisabledBoundSourceFailsClosedWithoutUsingSettingsDefault(t *testing.T) {
	store, root, plan := newV2TestStore(t, "job-disabled-auth")
	installAuthBindings(store, "safari-binding")
	store.mu.Lock()
	store.state.AuthSourceBindings[0].Enabled = false
	store.state.Jobs[0].AuthIntent = jobmodel.AuthIntent{RequiresAuthenticatedExecution: true, AuthSourceBindingRef: "chrome-binding"}
	store.mu.Unlock()
	manager := New(nil, nil)
	installDarwinBrowserSpecResolver(manager)
	t.Cleanup(func() { _ = manager.Close() })
	if err := manager.SetStateStore(store); err != nil {
		t.Fatal(err)
	}
	calls := 0
	manager.runDownload = func(context.Context, engine.Request, engine.EventHandler) (engine.Result, error) {
		calls++
		return engine.Result{}, nil
	}
	if _, err := manager.SubmitAdmitted("job-disabled-auth", Request{URL: "https://www.youtube.com/watch?v=abc123", VideoID: "abc123", Title: "Demo", PlanID: plan.ID, OutputDir: root}, &plan, AdmittedOutput{Basename: "Demo [abc123] [1080p].mp4"}); err != nil {
		t.Fatal(err)
	}
	job := waitForV2Job(t, store, "job-disabled-auth", jobmodel.LifecycleActionRequired)
	if calls != 0 || job.ActionRequiredCode != authenticatedSessionUnavailableCode {
		t.Fatalf("runner calls=%d job=%#v", calls, job)
	}
}

func TestAuthenticatedValidatedRetryUsesDurableBinding(t *testing.T) {
	store, root, _ := newV2TestStore(t, "job-auth-retry")
	installAuthBindings(store, "safari-binding")
	store.mu.Lock()
	store.state.Jobs[0].Lifecycle = jobmodel.LifecycleFailed
	store.state.Jobs[0].Desired = jobmodel.DesiredRunning
	store.state.Jobs[0].AuthIntent = jobmodel.AuthIntent{RequiresAuthenticatedExecution: true, AuthSourceBindingRef: "chrome-binding"}
	store.mu.Unlock()
	manager := New(nil, nil)
	installDarwinBrowserSpecResolver(manager)
	t.Cleanup(func() { _ = manager.Close() })
	if err := manager.SetStateStore(store); err != nil {
		t.Fatal(err)
	}
	if err := manager.RestoreStateV2(store.Snapshot()); err != nil {
		t.Fatal(err)
	}
	manager.inspectResume = func(context.Context, engine.OutputRootRef, string) (engine.ResumeSummary, error) {
		return engine.ResumeSummary{HasManifest: true, Classification: "available"}, nil
	}
	requests := make(chan engine.Request, 1)
	manager.runDownload = func(_ context.Context, request engine.Request, _ engine.EventHandler) (engine.Result, error) {
		requests <- request
		return engine.Result{Filename: filepath.Join(root, "Demo [abc123] [1080p].mp4")}, nil
	}
	if err := manager.Retry("job-auth-retry"); err != nil {
		t.Fatal(err)
	}
	request := <-requests
	if request.CookiesFromBrowser != "chrome" || request.CookieFile != "" || !request.ScopeBrowserCookiesToURL {
		t.Fatalf("retry request = %#v", request)
	}
	waitForV2Job(t, store, "job-auth-retry", jobmodel.LifecycleCompleted)
}

func TestForgetSourceTombstonesBindingAndMovesDependentJobToActionRequired(t *testing.T) {
	store, _, _ := newV2TestStore(t, "job-forgotten-auth")
	installAuthBindings(store, "chrome-binding")
	store.mu.Lock()
	store.state.Jobs[0].Lifecycle = jobmodel.LifecyclePaused
	store.state.Jobs[0].Desired = jobmodel.DesiredPaused
	store.state.Jobs[0].AuthIntent = jobmodel.AuthIntent{RequiresAuthenticatedExecution: true, AuthSourceBindingRef: "chrome-binding"}
	store.mu.Unlock()
	manager := New(nil, nil)
	installDarwinBrowserSpecResolver(manager)
	t.Cleanup(func() { _ = manager.Close() })
	if err := manager.SetStateStore(store); err != nil {
		t.Fatal(err)
	}
	if err := manager.RestoreStateV2(store.Snapshot()); err != nil {
		t.Fatal(err)
	}
	preview, err := manager.PreviewForgetAuthSource("chrome-binding")
	if err != nil || preview.Label != "Chrome — Default" || preview.Jobs != 1 || preview.Collections != 0 || preview.Active != 0 {
		t.Fatalf("forget preview = %#v, %v", preview, err)
	}
	view := manager.QueueView()
	if len(view.Rows) != 1 || view.Rows[0].AccessMode != "browser-session" || view.Rows[0].BrowserSourceLabel != "Chrome — Default" {
		t.Fatalf("authenticated queue presentation = %#v", view.Rows)
	}
	count, err := manager.ForgetAuthSource("chrome-binding")
	if err != nil || count != 1 {
		t.Fatalf("ForgetAuthSource() = %d, %v", count, err)
	}
	state := store.Snapshot()
	if state.Settings.DefaultAuthSourceBindingRef != "" || !state.Settings.BrowserAccessEnabled {
		t.Fatalf("settings after forget = %#v", state.Settings)
	}
	if state.AuthSourceBindings[0].Enabled || state.Jobs[0].Lifecycle != jobmodel.LifecycleActionRequired || state.Jobs[0].AuthIntent.AuthSourceBindingRef != "chrome-binding" {
		t.Fatalf("state after forget = %#v, %#v", state.AuthSourceBindings, state.Jobs[0])
	}
	view = manager.QueueView()
	review, err := manager.QueueActionRequiredReview("job-forgotten-auth", view.Rows[0].CommandToken)
	if err != nil {
		t.Fatal(err)
	}
	if review.AccessMode != "browser-session" || review.BrowserSourceLabel != "Chrome — Default" || review.RetryFreshLabel != "Refresh Chrome — Default and retry" || review.StartOverLabel != "Start over publicly from Home" || review.CanRetryFreshLink || !strings.Contains(review.Message, "will not switch") {
		t.Fatalf("authenticated recovery review = %#v", review)
	}
	beforeRetry := store.Snapshot().Jobs[0]
	if err := manager.QueueActionRequiredRetryFreshLink("job-forgotten-auth", view.Rows[0].CommandToken); err == nil {
		t.Fatal("forgotten source unexpectedly authorized a fresh retry")
	}
	afterRetry := store.Snapshot().Jobs[0]
	if afterRetry.SessionID != beforeRetry.SessionID || afterRetry.AttemptID != beforeRetry.AttemptID || afterRetry.SessionRestarts != beforeRetry.SessionRestarts || len(store.Snapshot().Cleanup) != 0 {
		t.Fatalf("rejected retry mutated durable evidence: before=%#v after=%#v cleanup=%#v", beforeRetry, afterRetry, store.Snapshot().Cleanup)
	}
	view = manager.QueueView()
	if err := manager.QueueRemove("job-forgotten-auth", view.Rows[0].CommandToken); err != nil {
		t.Fatalf("dismiss settled authenticated row: %v", err)
	}
	afterDismiss := store.Snapshot()
	if len(afterDismiss.Jobs) != 0 || len(afterDismiss.AuthSourceBindings) != 2 || afterDismiss.AuthSourceBindings[0].Enabled {
		t.Fatalf("dismiss changed browser tombstone authority: jobs=%#v bindings=%#v", afterDismiss.Jobs, afterDismiss.AuthSourceBindings)
	}
}

func TestForgetPreviewCountsOnlyCollectionsWithAffectedNonTerminalChildren(t *testing.T) {
	store, _, _ := newV2TestStore(t)
	installAuthBindings(store, "chrome-binding")
	intent := jobmodel.AuthIntent{RequiresAuthenticatedExecution: true, AuthSourceBindingRef: "chrome-binding"}
	store.mu.Lock()
	store.state.Jobs = []jobmodel.DurableJob{
		{ID: "affected", CollectionID: "collection-active", Lifecycle: jobmodel.LifecyclePaused, AuthIntent: intent},
		{ID: "completed-sibling", CollectionID: "collection-active", Lifecycle: jobmodel.LifecycleCompleted, AuthIntent: intent},
		{ID: "completed-only", CollectionID: "collection-terminal", Lifecycle: jobmodel.LifecycleCompleted, AuthIntent: intent},
		{ID: "canceled-only", CollectionID: "collection-canceled", Lifecycle: jobmodel.LifecycleCanceled, AuthIntent: intent},
	}
	store.mu.Unlock()
	manager := New(nil, nil)
	t.Cleanup(func() { _ = manager.Close() })
	if err := manager.SetStateStore(store); err != nil {
		t.Fatal(err)
	}
	preview, err := manager.PreviewForgetAuthSource("chrome-binding")
	if err != nil || preview.Jobs != 1 || preview.Collections != 1 || preview.Active != 0 {
		t.Fatalf("forget impact = %#v, %v", preview, err)
	}
}

func TestSupersededActivationReprojectsDurableActionRequiredWithoutEngineRun(t *testing.T) {
	store, root, plan := newV2TestStore(t, "job-auth-race")
	installAuthBindings(store, "chrome-binding")
	store.mu.Lock()
	store.state.Jobs[0].AuthIntent = jobmodel.AuthIntent{RequiresAuthenticatedExecution: true, AuthSourceBindingRef: "chrome-binding"}
	store.mu.Unlock()
	manager := New(nil, nil)
	installDarwinBrowserSpecResolver(manager)
	t.Cleanup(func() { _ = manager.Close() })
	if err := manager.SetStateStore(store); err != nil {
		t.Fatal(err)
	}
	manager.mu.Lock()
	manager.concurrency = 0
	manager.mu.Unlock()
	called := make(chan struct{}, 1)
	manager.runDownload = func(context.Context, engine.Request, engine.EventHandler) (engine.Result, error) {
		called <- struct{}{}
		return engine.Result{}, nil
	}
	if _, err := manager.SubmitAdmitted("job-auth-race", Request{URL: "https://www.youtube.com/watch?v=abc123", VideoID: "abc123", Title: "Demo", PlanID: plan.ID, OutputDir: root}, &plan, AdmittedOutput{Basename: "Demo [abc123] [1080p].mp4"}); err != nil {
		t.Fatal(err)
	}
	if err := store.Transaction(nil, func(document *jobmodel.State) error {
		job := &document.Jobs[0]
		job.Lifecycle = jobmodel.LifecycleActionRequired
		job.Desired = jobmodel.DesiredPaused
		job.ActionRequiredCode = authenticatedSessionUnavailableCode
		job.LastErrorCode = authenticatedSessionUnavailableCode
		job.Revision++
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	manager.mu.Lock()
	manager.concurrency = 1
	manager.maybeStartNextLocked()
	manager.mu.Unlock()
	deadline := time.Now().Add(2 * time.Second)
	for {
		snapshot, ok := manager.Find("job-auth-race")
		if ok && snapshot.Status == StatusActionRequired {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("runtime did not reproject durable winner: %#v", snapshot)
		}
		time.Sleep(5 * time.Millisecond)
	}
	select {
	case <-called:
		t.Fatal("engine ran after durable activation was superseded")
	default:
	}
	view := manager.QueueView()
	if len(view.Rows) != 1 || view.Rows[0].Lifecycle != jobmodel.LifecycleActionRequired || view.Rows[0].AccessMode != "browser-session" {
		t.Fatalf("runtime/durable projection mismatch: %#v", view.Rows)
	}
}

func TestAuthenticatedPlaylistReviewPreservesEveryOutcomeAndStableOccurrenceOrder(t *testing.T) {
	store, _, _ := newV2TestStore(t)
	installAuthBindings(store, "safari-binding")
	manager := New(nil, nil)
	installDarwinBrowserSpecResolver(manager)
	t.Cleanup(func() { _ = manager.Close() })
	if err := manager.SetStateStore(store); err != nil {
		t.Fatal(err)
	}
	ready := func(id, title string) *engine.Result {
		return &engine.Result{InfoJSON: []byte(`{"id":"` + id + `","title":"` + title + `","channel":"Owner","formats":[{"format_id":"v1","ext":"mp4","vcodec":"avc1","acodec":"none","height":1080,"width":1920},{"format_id":"a1","ext":"m4a","vcodec":"none","acodec":"mp4a","abr":128}]}`)}
	}
	calls := 0
	manager.runAnalyze = func(_ context.Context, request engine.Request) (engine.Result, error) {
		calls++
		if request.CookiesFromBrowser != "chrome" || !request.ScopeBrowserCookiesToURL || !request.PlaylistReview || request.Playlist.Flat || request.Playlist.End != playlistReviewProbeLimit {
			t.Fatalf("playlist review request = %#v", request)
		}
		return engine.Result{
			InfoJSON: []byte(`{"id":"PLfixture","title":"Mixed playlist","channel":"Owner"}`),
			PlaylistReview: &engine.PlaylistReview{Complete: true, Discovered: 5, Occurrences: []engine.PlaylistReviewOccurrence{
				{SourceIndex: 3, Outcome: engine.PlaylistReviewUnavailable, ReasonCode: engine.PlaylistReviewReasonUnsupported, InfoJSON: []byte(`{"title":"Removed"}`)},
				{SourceIndex: 1, Outcome: engine.PlaylistReviewReady, ReasonCode: engine.PlaylistReviewReasonReady, Result: ready("fixture0001", "First ready")},
				{SourceIndex: 5, Outcome: engine.PlaylistReviewReady, ReasonCode: engine.PlaylistReviewReasonReady, Result: ready("fixture0001", "Duplicate occurrence")},
				{SourceIndex: 2, Outcome: engine.PlaylistReviewAuthRequired, ReasonCode: engine.PlaylistReviewReasonAuthentication, InfoJSON: []byte(`{"title":"Members item"}`)},
				{SourceIndex: 4, Outcome: engine.PlaylistReviewInvalid, ReasonCode: engine.PlaylistReviewReasonInvalid, InfoJSON: []byte(`{"title":"Invalid item"}`)},
			}},
		}, nil
	}
	const playlistURL = "https://www.youtube.com/playlist?list=PLfixture"
	summary, err := manager.AnalyzePlaylistAuthenticated(context.Background(), playlistURL, "chrome-binding")
	if err != nil {
		t.Fatal(err)
	}
	if calls != 1 || summary.EntryCount != 5 || summary.Ready != 2 || summary.AuthRequired != 1 || summary.Unavailable != 1 || summary.Invalid != 1 || !summary.Admissible || summary.ReviewAuthority == "" {
		t.Fatalf("authenticated playlist summary = %#v; calls=%d", summary, calls)
	}
	if summary.BrowserAccess.Mode != "browser-session" || summary.BrowserAccess.Label != "Chrome — Default" {
		t.Fatalf("browser access = %#v", summary.BrowserAccess)
	}
	for index, entry := range summary.Entries {
		if entry.Index != index+1 || entry.OccurrenceID == "" {
			t.Fatalf("entry %d = %#v", index, entry)
		}
	}
	if summary.Entries[0].VideoID != summary.Entries[4].VideoID || summary.Entries[0].OccurrenceID == summary.Entries[4].OccurrenceID {
		t.Fatalf("duplicate video occurrences lost identity: %#v %#v", summary.Entries[0], summary.Entries[4])
	}
	preview, selected, intent, err := manager.ResolveAuthenticatedPlaylistSelection(summary.ReviewAuthority, []string{summary.Entries[4].OccurrenceID, summary.Entries[0].OccurrenceID})
	if err != nil {
		t.Fatal(err)
	}
	if preview.ID != "PLfixture" || len(selected) != 2 || selected[0].Entry.Index != 1 || selected[1].Entry.Index != 5 || intent.AuthSourceBindingRef != "chrome-binding" {
		t.Fatalf("stable authenticated selection = %#v, %#v, %#v", preview, selected, intent)
	}
}

func TestAuthenticatedPlaylistReviewRejectsSilentLimitTruncation(t *testing.T) {
	store, _, _ := newV2TestStore(t)
	installAuthBindings(store, "chrome-binding")
	manager := New(nil, nil)
	installDarwinBrowserSpecResolver(manager)
	t.Cleanup(func() { _ = manager.Close() })
	if err := manager.SetStateStore(store); err != nil {
		t.Fatal(err)
	}
	occurrences := make([]engine.PlaylistReviewOccurrence, playlistReviewProbeLimit)
	for index := range occurrences {
		occurrences[index] = engine.PlaylistReviewOccurrence{
			SourceIndex: index + 1, Outcome: engine.PlaylistReviewInvalid,
			ReasonCode: engine.PlaylistReviewReasonInvalid, InfoJSON: []byte(`{"title":"Unavailable entry"}`),
		}
	}
	manager.runAnalyze = func(_ context.Context, request engine.Request) (engine.Result, error) {
		if request.Playlist.End != playlistReviewProbeLimit {
			t.Fatalf("playlist review probe end = %d, want %d", request.Playlist.End, playlistReviewProbeLimit)
		}
		return engine.Result{
			InfoJSON:       []byte(`{"id":"PLfixture","title":"Oversized playlist"}`),
			PlaylistReview: &engine.PlaylistReview{Complete: true, Discovered: len(occurrences), Occurrences: occurrences},
		}, nil
	}
	_, err := manager.AnalyzePlaylistAuthenticated(context.Background(), "https://www.youtube.com/playlist?list=PLfixture", "chrome-binding")
	if !errors.Is(err, ErrAuthenticatedPlaylistLimit) {
		t.Fatalf("oversized authenticated playlist error = %v", err)
	}
	if len(manager.playlistReviewCache) != 0 {
		t.Fatalf("oversized playlist created admissible authority: %#v", manager.playlistReviewCache)
	}
}

func TestAuthenticatedPlaylistReviewNeverFallsBackAndInvalidatesOldAuthority(t *testing.T) {
	store, _, _ := newV2TestStore(t)
	installAuthBindings(store, "chrome-binding")
	manager := New(nil, nil)
	installDarwinBrowserSpecResolver(manager)
	t.Cleanup(func() { _ = manager.Close() })
	if err := manager.SetStateStore(store); err != nil {
		t.Fatal(err)
	}
	const playlistURL = "https://www.youtube.com/playlist?list=PLfixture"
	calls := 0
	manager.runAnalyze = func(_ context.Context, request engine.Request) (engine.Result, error) {
		calls++
		if request.CookiesFromBrowser != "chrome" || !request.ScopeBrowserCookiesToURL {
			t.Fatalf("playlist request silently became public: %#v", request)
		}
		return engine.Result{}, &engine.Error{Category: engine.ErrorAuthentication, Err: errors.New("COOKIE_CANARY")}
	}
	if _, err := manager.AnalyzePlaylistAuthenticated(context.Background(), playlistURL, "chrome-binding"); err == nil {
		t.Fatal("authenticated playlist failure unexpectedly succeeded")
	}
	if calls != 1 {
		t.Fatalf("engine calls = %d, want one authenticated attempt", calls)
	}

	result := engine.Result{InfoJSON: []byte(`{"id":"PLfixture","title":"Playlist"}`)}
	result.PlaylistReview = &engine.PlaylistReview{Complete: true, Discovered: 1, Occurrences: []engine.PlaylistReviewOccurrence{{
		SourceIndex: 1, Outcome: engine.PlaylistReviewReady, ReasonCode: engine.PlaylistReviewReasonReady, Result: &engine.Result{InfoJSON: analysisFixtureResult().InfoJSON},
	}}}
	manager.runAnalyze = func(context.Context, engine.Request) (engine.Result, error) { return result, nil }
	first, err := manager.AnalyzePlaylistAuthenticated(context.Background(), playlistURL, "chrome-binding")
	if err != nil {
		t.Fatal(err)
	}
	second, err := manager.AnalyzePlaylistAuthenticated(context.Background(), playlistURL, "chrome-binding")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, _, err := manager.ResolveAuthenticatedPlaylistSelection(first.ReviewAuthority, []string{first.Entries[0].OccurrenceID}); err == nil {
		t.Fatal("superseded playlist review authority remained usable")
	}
	if _, err := manager.ForgetAuthSource("chrome-binding"); err != nil {
		t.Fatal(err)
	}
	if _, _, _, err := manager.ResolveAuthenticatedPlaylistSelection(second.ReviewAuthority, []string{second.Entries[0].OccurrenceID}); err == nil {
		t.Fatal("forgotten source left playlist review authority usable")
	}
}

func TestBrowserSourceCheckReturnsOnlyBoundedStatus(t *testing.T) {
	store, _, _ := newV2TestStore(t)
	installAuthBindings(store, "chrome-binding")
	manager := New(nil, nil)
	installDarwinBrowserSpecResolver(manager)
	t.Cleanup(func() { _ = manager.Close() })
	if err := manager.SetStateStore(store); err != nil {
		t.Fatal(err)
	}
	checkedSpec := ""
	manager.checkBrowserCookies = func(_ context.Context, spec string) (engine.BrowserCookieCheck, error) {
		checkedSpec = spec
		return engine.BrowserCookieCheck{Usable: true, Partial: true}, nil
	}
	check, err := manager.CheckBrowserSource(context.Background(), "chrome-binding")
	if err != nil || checkedSpec != "chrome" || check.Status != "partial" || !check.Ready || !check.Partial || strings.Contains(strings.ToLower(check.Message), "cookie") {
		t.Fatalf("browser source check = %#v, spec=%q, err=%v", check, checkedSpec, err)
	}
	manager.checkBrowserCookies = func(context.Context, string) (engine.BrowserCookieCheck, error) {
		return engine.BrowserCookieCheck{}, &engine.BrowserCookieCheckError{Code: "permission-denied"}
	}
	check, err = manager.CheckBrowserSource(context.Background(), "chrome-binding")
	if err != nil || check.Status != "permission-denied" || check.Ready || strings.Contains(check.Message, "KEYCHAIN_CANARY") {
		t.Fatalf("permission check = %#v, %v", check, err)
	}
	manager.checkBrowserCookies = func(context.Context, string) (engine.BrowserCookieCheck, error) {
		return engine.BrowserCookieCheck{}, &engine.BrowserCookieCheckError{Code: "source-empty"}
	}
	check, err = manager.CheckBrowserSource(context.Background(), "chrome-binding")
	if err != nil || check.Status != "source-empty" || check.Ready || !strings.Contains(check.Message, "cannot determine whether YouTube is signed in") {
		t.Fatalf("empty source check = %#v, %v", check, err)
	}
}

func TestManagerCloseCancelsAndJoinsBrowserSourceCheck(t *testing.T) {
	store, _, _ := newV2TestStore(t)
	installAuthBindings(store, "chrome-binding")
	manager := New(nil, nil)
	installDarwinBrowserSpecResolver(manager)
	if err := manager.SetStateStore(store); err != nil {
		t.Fatal(err)
	}
	started := make(chan struct{})
	manager.checkBrowserCookies = func(ctx context.Context, _ string) (engine.BrowserCookieCheck, error) {
		close(started)
		<-ctx.Done()
		return engine.BrowserCookieCheck{}, ctx.Err()
	}
	result := make(chan BrowserSourceCheck, 1)
	errs := make(chan error, 1)
	go func() {
		check, err := manager.CheckBrowserSource(context.Background(), "chrome-binding")
		result <- check
		errs <- err
	}()
	<-started
	if err := manager.Close(); err != nil {
		t.Fatal(err)
	}
	if err := <-errs; err != nil {
		t.Fatal(err)
	}
	if check := <-result; check.Status != "canceled" || check.Ready {
		t.Fatalf("check after manager close = %#v", check)
	}
}

func TestRestoredAuthenticatedJobUsesDurableBindingNotChangedDefault(t *testing.T) {
	store, root, _ := newV2TestStore(t, "job-restored-auth")
	installAuthBindings(store, "safari-binding")
	store.mu.Lock()
	store.state.Jobs[0].Lifecycle = jobmodel.LifecyclePaused
	store.state.Jobs[0].Desired = jobmodel.DesiredPaused
	store.state.Jobs[0].AuthIntent = jobmodel.AuthIntent{RequiresAuthenticatedExecution: true, AuthSourceBindingRef: "chrome-binding"}
	store.mu.Unlock()
	manager := New(nil, nil)
	installDarwinBrowserSpecResolver(manager)
	t.Cleanup(func() { _ = manager.Close() })
	if err := manager.SetStateStore(store); err != nil {
		t.Fatal(err)
	}
	if err := manager.RestoreStateV2(store.Snapshot()); err != nil {
		t.Fatal(err)
	}
	requests := make(chan engine.Request, 1)
	manager.runDownload = func(_ context.Context, request engine.Request, _ engine.EventHandler) (engine.Result, error) {
		requests <- request
		return engine.Result{Filename: filepath.Join(root, "Demo [abc123] [1080p].mp4")}, nil
	}
	if err := manager.Resume("job-restored-auth"); err != nil {
		t.Fatal(err)
	}
	request := <-requests
	if request.CookiesFromBrowser != "chrome" || request.CookieFile != "" || !request.ScopeBrowserCookiesToURL {
		t.Fatalf("restored request = %#v", request)
	}
	waitForV2Job(t, store, "job-restored-auth", jobmodel.LifecycleCompleted)
}
