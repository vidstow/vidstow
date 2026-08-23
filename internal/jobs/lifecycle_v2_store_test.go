package jobs_test

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/tejasa97/vidstow/internal/authsource"
	"github.com/tejasa97/vidstow/internal/diagnostics"
	"github.com/tejasa97/vidstow/internal/jobmodel"
	"github.com/tejasa97/vidstow/internal/jobs"
	"github.com/tejasa97/vidstow/internal/store"
	"github.com/tejasa97/ytdlp-go/engine"
)

func privateStoreTestDir(t *testing.T) string {
	t.Helper()
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	dir, err := os.MkdirTemp(home, ".vidstow-jobs-store-test-")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(dir, 0o700); err != nil {
		_ = os.RemoveAll(dir)
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	return dir
}

func TestAuthenticatedCanariesAreCuratedAcrossRealStoreCloseAndReopen(t *testing.T) {
	path := filepath.Join(privateStoreTestDir(t), "authenticated-state.json")
	durable, status, err := store.OpenV2(path)
	if err != nil || !status.Healthy() {
		t.Fatalf("OpenV2: %v %#v", err, status)
	}
	root := t.TempDir()
	now := time.Now().UTC()
	const jobID = "job-real-auth-store"
	outputRoot := jobmodel.OutputRootRef{CanonicalPath: root, Identity: "volume-auth-test"}
	job := jobmodel.DurableJob{
		ID: jobID, Revision: 1, AttemptID: "attempt-real-auth-store", SessionID: "abcdef0123456789abcdef0123456789",
		QueueOrdinal: 1, Lifecycle: jobmodel.LifecyclePaused, Phase: jobmodel.PhasePreparing, Desired: jobmodel.DesiredPaused,
		Request:     jobmodel.PersistedRequest{SourceURL: "https://www.youtube.com/watch?v=abc12345678", VideoID: "abc12345678", Title: "Demo", Quality: "best", PlanID: "video-1080-mp4"},
		Plan:        jobmodel.PersistedPlan{ID: "video-1080-mp4", Kind: "video", Label: "1080p", Container: "MP4", PrivateSelector: "137+140"},
		OutputRoot:  outputRoot,
		Reservation: jobmodel.ReservationSet{GroupID: jobID, Directory: outputRoot, Artifacts: []jobmodel.ReservedArtifact{{Kind: string(engine.ArtifactKindPrimary), Identity: "primary", Basename: "Demo [abc123] [1080p].mp4"}}},
		AuthIntent:  jobmodel.AuthIntent{RequiresAuthenticatedExecution: true, AuthSourceBindingRef: "chrome-binding"},
		RetryMode:   jobmodel.RetryModeNone,
		CreatedAt:   now, UpdatedAt: now,
	}
	if err := durable.Transaction(nil, func(state *jobmodel.State) error {
		state.Settings.BrowserAccessEnabled = true
		state.Settings.BrowserAccessConsentVersion = authsource.CurrentConsentVersion
		state.Settings.DefaultAuthSourceBindingRef = "safari-binding"
		state.AuthSourceBindings = []authsource.Binding{
			{ID: "chrome-binding", Enabled: true, Descriptor: authsource.Descriptor{SchemaVersion: 1, Platform: "darwin", Browser: authsource.BrowserChrome}},
			{ID: "safari-binding", Enabled: true, Descriptor: authsource.Descriptor{SchemaVersion: 1, Platform: "darwin", Browser: authsource.BrowserSafari}},
		}
		state.Jobs = []jobmodel.DurableJob{job}
		state.NextQueueOrdinal = 2
		return nil
	}); err != nil {
		t.Fatalf("seed authenticated job: %v", err)
	}
	if err := durable.Close(); err != nil {
		t.Fatal(err)
	}

	reopened, status, err := store.OpenV2(path)
	if err != nil || !status.Healthy() {
		t.Fatalf("reopen State v2: %v %#v", err, status)
	}
	defer reopened.Close()
	var eventMu sync.Mutex
	wailsEvents := []jobs.Event{}
	manager := jobs.New(nil, func(event jobs.Event) {
		eventMu.Lock()
		wailsEvents = append(wailsEvents, event)
		eventMu.Unlock()
	})
	jobs.InstallBrowserSpecResolverForTest(manager, func(descriptor authsource.Descriptor) (string, error) {
		if descriptor == (authsource.Descriptor{SchemaVersion: authsource.DescriptorSchemaVersion, Platform: "darwin", Browser: authsource.BrowserChrome}) {
			return "chrome", nil
		}
		return "", authsource.NewError("unsupported-platform")
	})
	if err := manager.SetStateStore(reopened); err != nil {
		t.Fatal(err)
	}
	if err := manager.RestoreStateV2(reopened.Snapshot()); err != nil {
		t.Fatal(err)
	}
	const eventCanary = "RAW_ENGINE_EVENT_COOKIE_PATH_CANARY"
	const errorCanary = "RAW_ENGINE_ERROR_SIGNED_URL_CANARY"
	rawEngineErr := &engine.Error{Category: engine.ErrorAuthentication, Err: errors.New(errorCanary)}
	if !strings.Contains(rawEngineErr.Error(), errorCanary) {
		t.Fatal("test did not seed the raw engine error canary")
	}
	requests := make(chan engine.Request, 1)
	jobs.InstallDownloadHooksForTest(manager, nil, func(_ context.Context, request engine.Request, handler engine.EventHandler) (engine.Result, error) {
		requests <- request
		if err := handler(context.Background(), engine.Event{Kind: "future-browser-provider-event", Message: eventCanary}); err != nil {
			return engine.Result{}, err
		}
		return engine.Result{}, rawEngineErr
	})
	if err := manager.Resume(jobID); err != nil {
		t.Fatal(err)
	}
	request := <-requests
	if request.CookiesFromBrowser != "chrome" || request.CookieFile != "" {
		t.Fatalf("reopened authenticated request = %#v", request)
	}
	deadline := time.Now().Add(3 * time.Second)
	for reopened.Snapshot().Jobs[0].Lifecycle != jobmodel.LifecycleActionRequired {
		if time.Now().After(deadline) {
			t.Fatalf("authenticated failure did not settle: %#v", reopened.Snapshot().Jobs[0])
		}
		time.Sleep(5 * time.Millisecond)
	}
	snapshot, ok := manager.Find(jobID)
	if !ok {
		t.Fatal("settled job missing from manager")
	}
	view := manager.QueueView()
	if len(view.Rows) != 1 || view.Rows[0].AccessMode != "browser-session" || view.Rows[0].BrowserSourceLabel != "Chrome — Default" {
		t.Fatalf("reopened queue presentation = %#v", view.Rows)
	}
	if err := manager.Close(); err != nil {
		t.Fatal(err)
	}
	eventMu.Lock()
	capturedEvents := append([]jobs.Event(nil), wailsEvents...)
	eventMu.Unlock()
	var terminalDiagnostic *jobs.Diagnostic
	for _, event := range capturedEvents {
		if event.Diagnostic != nil {
			terminalDiagnostic = event.Diagnostic
		}
	}
	if terminalDiagnostic == nil || terminalDiagnostic.Category != "authentication_required" {
		t.Fatalf("curated terminal diagnostic = %#v", terminalDiagnostic)
	}
	if err := reopened.Close(); err != nil {
		t.Fatal(err)
	}
	finalStore, finalStatus, err := store.OpenV2(path)
	if err != nil || !finalStatus.Healthy() {
		t.Fatalf("final OpenV2: %v %#v", err, finalStatus)
	}
	defer finalStore.Close()

	diagnosticsDir := filepath.Join(privateStoreTestDir(t), "diagnostics")
	recorder, err := diagnostics.Open(filepath.Join(diagnosticsDir, "history-v1.json"))
	if err != nil {
		t.Fatal(err)
	}
	outbox, err := diagnostics.OpenOutbox(filepath.Join(diagnosticsDir, "outbox-v1.json"))
	if err != nil {
		t.Fatal(err)
	}
	diagnosticEvent := diagnostics.Event{
		SchemaVersion: diagnostics.SchemaVersion,
		EventID:       "11111111-1111-4111-8111-111111111111",
		SessionID:     "22222222-2222-4222-8222-222222222222",
		OccurredAt:    time.Now().UTC(),
		AppVersion:    "test",
		EngineVersion: "test",
		Platform:      diagnostics.CurrentPlatform(),
		Type:          diagnostics.TypeProblemObserved,
		Problem: &diagnostics.Problem{
			Stage: terminalDiagnostic.Stage, Category: terminalDiagnostic.Category,
			Outcome: "terminal", RetryBucket: "none",
		},
	}
	if err := recorder.Record(diagnosticEvent); err != nil {
		t.Fatal(err)
	}
	if err := outbox.Enqueue(diagnosticEvent); err != nil {
		t.Fatal(err)
	}
	diagnosticHistory, err := recorder.Recent()
	if err != nil {
		t.Fatal(err)
	}
	outboxBatch, err := outbox.Batch()
	if err != nil {
		t.Fatal(err)
	}
	surfaces := map[string]any{
		"state-after-second-reopen": finalStore.Snapshot(),
		"job-snapshot":              snapshot,
		"queue-view":                view,
		"wails-events":              capturedEvents,
		"diagnostic-history":        diagnosticHistory,
		"diagnostic-outbox":         outboxBatch,
		"download-history":          finalStore.History(),
	}
	encoded, err := json.Marshal(surfaces)
	if err != nil {
		t.Fatal(err)
	}
	for _, canary := range []string{eventCanary, errorCanary} {
		if strings.Contains(string(encoded), canary) {
			t.Fatalf("serialized surfaces retained %q: %s", canary, encoded)
		}
	}
	for _, curated := range []string{"authenticated-session-unavailable", "authentication_required", "Browser session needs attention"} {
		if !strings.Contains(string(encoded), curated) {
			t.Fatalf("serialized surfaces lack curated value %q: %s", curated, encoded)
		}
	}
}

// TestV2ZeroProgressRestartCommitsOnRealStore proves the escalated retry
// rotation is valid under V2Store.Transaction: no mid-life tombstone is
// written, so validateState cannot reject the commit the way a retired-session
// tombstone overlapping a live job would.
func TestV2ZeroProgressRestartCommitsOnRealStore(t *testing.T) {
	path := filepath.Join(privateStoreTestDir(t), "state.json")
	durable, status, err := store.OpenV2(path)
	if err != nil || !status.Healthy() {
		t.Fatalf("OpenV2: %v %#v", err, status)
	}
	defer durable.Close()

	root := t.TempDir()
	now := time.Now().UTC()
	const jobID = "job-real-store"
	originalSession := "0123456789abcdef0123456789abcdef"
	outputRoot := jobmodel.OutputRootRef{CanonicalPath: root, Identity: "volume-test"}
	job := jobmodel.DurableJob{
		ID: jobID, Revision: 1, AttemptID: "attempt-real-store", SessionID: originalSession,
		QueueOrdinal: 1, Lifecycle: jobmodel.LifecycleFailed, Phase: jobmodel.PhasePreparing, Desired: jobmodel.DesiredRunning,
		Request: jobmodel.PersistedRequest{
			SourceURL: "https://www.youtube.com/watch?v=abc12345678", VideoID: "abc12345678",
			Title: "Demo", Channel: "Creator", Quality: "best", PlanID: "video-1080-mp4",
		},
		Plan:       jobmodel.PersistedPlan{ID: "video-1080-mp4", Kind: "video", Label: "1080p", Container: "MP4", PrivateSelector: "137+140"},
		OutputRoot: outputRoot,
		Reservation: jobmodel.ReservationSet{
			GroupID: jobID, Directory: outputRoot,
			Artifacts: []jobmodel.ReservedArtifact{{Kind: string(engine.ArtifactKindPrimary), Identity: "primary", Basename: "Demo [abc123] [1080p].mp4"}},
		},
		RetryMode: jobmodel.RetryModeResumeValidated, LastErrorCode: jobs.RetryCodeMediaLinkExpiredForTest,
		LastFailureCommittedBytes: 3145728, ZeroProgressResumes: 1, CreatedAt: now, UpdatedAt: now,
	}
	if err := durable.Transaction(nil, func(state *jobmodel.State) error {
		state.Jobs = []jobmodel.DurableJob{job}
		state.NextQueueOrdinal = 2
		return nil
	}); err != nil {
		t.Fatalf("seed job: %v", err)
	}

	manager := jobs.New(nil, nil)
	defer manager.Close()
	if err := manager.SetStateStore(durable); err != nil {
		t.Fatal(err)
	}
	if err := manager.RestoreStateV2(durable.Snapshot()); err != nil {
		t.Fatal(err)
	}
	started := make(chan string, 1)
	jobs.InstallDownloadHooksForTest(manager,
		func(context.Context, engine.OutputRootRef, string) (engine.ResumeSummary, error) {
			return engine.ResumeSummary{
				HasManifest: true, Classification: "available",
				Components: []engine.ResumeComponent{{ID: "video", Kind: "video", CommittedBytes: 3145728}},
			}, nil
		},
		func(_ context.Context, request engine.Request, _ engine.EventHandler) (engine.Result, error) {
			started <- request.Filesystem.Resume.SessionID
			return engine.Result{Filename: filepath.Join(root, "Demo [abc123] [1080p].mp4")}, nil
		},
	)

	if err := manager.Retry(jobID); err != nil {
		t.Fatal(err)
	}
	restarted := <-started
	if restarted == originalSession || restarted == "" {
		t.Fatalf("real-store restart session = %q; want fresh identity", restarted)
	}

	deadline := time.Now().Add(2 * time.Second)
	var completed jobmodel.DurableJob
	found := false
	for time.Now().Before(deadline) {
		for _, candidate := range durable.Snapshot().Jobs {
			if candidate.ID == jobID && candidate.Lifecycle == jobmodel.LifecycleCompleted {
				completed = candidate
				found = true
				break
			}
		}
		if found {
			break
		}
		time.Sleep(time.Millisecond)
	}
	if !found {
		t.Fatalf("real-store job did not complete; snapshot = %#v", durable.Snapshot().Jobs)
	}
	if completed.SessionID != restarted || completed.RetryMode != jobmodel.RetryModeRestartNewSession || completed.SessionRestarts != 1 {
		t.Fatalf("real-store completed identity = %#v; want session %q restart-new-session restarts=1", completed, restarted)
	}
	if len(durable.Snapshot().Cleanup) != 0 {
		t.Fatalf("real-store cleanup = %#v; want no mid-life tombstone", durable.Snapshot().Cleanup)
	}
}
