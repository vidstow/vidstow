package store

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tejasa97/vidstow/internal/authsource"
	"github.com/tejasa97/vidstow/internal/jobmodel"
)

func TestStateV2LegacyRowsNormalizeToPublic(t *testing.T) {
	state := defaultStateV2()
	job := testJob()
	state.Jobs = []jobmodel.DurableJob{job}
	state.NextQueueOrdinal = 2
	data, err := json.Marshal(state)
	if err != nil {
		t.Fatal(err)
	}
	// Simulate a pre-feature State v2 image, whose durable job schema did not
	// contain authIntent at all.
	var legacy map[string]any
	if err := json.Unmarshal(data, &legacy); err != nil {
		t.Fatal(err)
	}
	for _, rawJob := range legacy["jobs"].([]any) {
		delete(rawJob.(map[string]any), "authIntent")
	}
	delete(legacy, "authSourceBindings")
	data, err = json.Marshal(legacy)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "state.json")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	opened, status, err := OpenV2(path)
	if err != nil || !status.Healthy() {
		t.Fatalf("OpenV2() = %#v, %#v, %v", opened, status, err)
	}
	defer opened.Close()
	got := opened.Snapshot()
	if len(got.Jobs) != 1 || got.Jobs[0].AuthIntent != (jobmodel.AuthIntent{}) || got.Settings.BrowserAccessEnabled {
		t.Fatalf("legacy auth normalization = %#v, settings=%#v", got.Jobs, got.Settings)
	}
}

func TestStateV2RejectsUnsafeBrowserDescriptor(t *testing.T) {
	state := defaultStateV2()
	state.Settings.BrowserAccessEnabled = true
	state.Settings.BrowserAccessConsentVersion = authsource.CurrentConsentVersion
	state.Settings.DefaultAuthSourceBindingRef = "binding-1"
	state.AuthSourceBindings = []authsource.Binding{{
		ID: "binding-1", Enabled: true,
		Descriptor: authsource.Descriptor{SchemaVersion: 1, Platform: "darwin", Browser: authsource.BrowserChrome, ProfileRef: "/Users/private-canary/Profile"},
	}}
	path := filepath.Join(t.TempDir(), "state.json")
	if err := os.WriteFile(path, mustJSON(t, state), 0o600); err != nil {
		t.Fatal(err)
	}
	opened, status, err := OpenV2(path)
	if err != nil || opened != nil || status.Mode != StartupRecoveryRequired {
		t.Fatalf("unsafe descriptor OpenV2() = %#v, %#v, %v", opened, status, err)
	}
}

func TestWholeSettingsUpdateCannotRewriteBrowserDefault(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	state, status, err := OpenV2(path)
	if err != nil || !status.Healthy() {
		t.Fatalf("OpenV2() = %#v, %#v, %v", state, status, err)
	}
	defer state.Close()
	binding := authsource.Binding{ID: "binding-1", Enabled: true, Descriptor: authsource.Descriptor{SchemaVersion: 1, Platform: "darwin", Browser: authsource.BrowserChrome}}
	if err := state.Transaction(nil, func(document *jobmodel.State) error {
		document.AuthSourceBindings = []authsource.Binding{binding}
		document.Settings.BrowserAccessEnabled = true
		document.Settings.BrowserAccessConsentVersion = authsource.CurrentConsentVersion
		document.Settings.DefaultAuthSourceBindingRef = binding.ID
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	settings := state.Settings()
	settings.DownloadConcurrency = 3
	settings.BrowserAccessEnabled = false
	settings.DefaultAuthSourceBindingRef = "renderer-substitution"
	settings.BrowserAccessConsentVersion = 0
	if err := state.SetSettings(settings); err != nil {
		t.Fatal(err)
	}
	got := state.Settings()
	if got.DownloadConcurrency != 3 || !got.BrowserAccessEnabled || got.DefaultAuthSourceBindingRef != binding.ID || got.BrowserAccessConsentVersion != authsource.CurrentConsentVersion {
		t.Fatalf("settings replacement = %#v", got)
	}
}

func TestReconfigureForgottenBrowserSourceCreatesFreshImmutableBinding(t *testing.T) {
	state, status, err := OpenV2(filepath.Join(t.TempDir(), "state.json"))
	if err != nil || !status.Healthy() {
		t.Fatalf("OpenV2() = %#v, %#v, %v", state, status, err)
	}
	defer state.Close()
	descriptor := authsource.Descriptor{SchemaVersion: authsource.DescriptorSchemaVersion, Platform: "darwin", Browser: authsource.BrowserChrome}
	first, err := state.configureBrowserSource(descriptor, authsource.CurrentConsentVersion)
	if err != nil {
		t.Fatal(err)
	}
	if err := state.Transaction(nil, func(document *jobmodel.State) error {
		document.AuthSourceBindings[0].Enabled = false
		document.Settings.BrowserAccessEnabled = false
		document.Settings.BrowserAccessConsentVersion = 0
		document.Settings.DefaultAuthSourceBindingRef = ""
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	second, err := state.configureBrowserSource(descriptor, authsource.CurrentConsentVersion)
	if err != nil {
		t.Fatal(err)
	}
	if first.BindingRef == "" || second.BindingRef == "" || first.BindingRef == second.BindingRef {
		t.Fatalf("binding identities were reused: first=%#v second=%#v", first, second)
	}
	snapshot := state.Snapshot()
	if len(snapshot.AuthSourceBindings) != 2 || snapshot.AuthSourceBindings[0].Enabled || !snapshot.AuthSourceBindings[1].Enabled || snapshot.Settings.DefaultAuthSourceBindingRef != second.BindingRef {
		t.Fatalf("fresh reconfiguration state = %#v, settings=%#v", snapshot.AuthSourceBindings, snapshot.Settings)
	}
}

func TestStateV2RestartsAuthenticatedPlaylistWithExactBindingAndOccurrenceOrder(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	state, status, err := OpenV2(path)
	if err != nil || !status.Healthy() {
		t.Fatalf("OpenV2() = %#v, %#v, %v", state, status, err)
	}
	binding := authsource.Binding{ID: "binding-1", Enabled: true, Descriptor: authsource.Descriptor{SchemaVersion: 1, Platform: "darwin", Browser: authsource.BrowserChrome}}
	intent := jobmodel.AuthIntent{RequiresAuthenticatedExecution: true, AuthSourceBindingRef: binding.ID}
	now := testJob().CreatedAt
	first := testJob()
	first.CollectionID, first.CollectionIndex, first.SourceIndex, first.SourceOccurrenceID = "collection-1", 1, 2, "11111111-1111-4111-8111-111111111111"
	first.AuthIntent = intent
	second := testJob()
	second.ID, second.AttemptID, second.SessionID, second.QueueOrdinal = "job-2", "attempt-2", "abcdef0123456789abcdef0123456789", 2
	second.CollectionID, second.CollectionIndex, second.SourceIndex, second.SourceOccurrenceID = "collection-1", 2, 7, "22222222-2222-4222-8222-222222222222"
	second.Request.VideoID, second.Request.SourceURL = "abc12345679", "https://www.youtube.com/watch?v=abc12345679"
	second.Reservation.GroupID = second.ID
	second.Reservation.Artifacts[0].Basename = "second.mp4"
	second.AuthIntent = intent
	collection := jobmodel.DurableCollection{
		ID: "collection-1", Revision: 1, Kind: jobmodel.CollectionKindPlaylist, PlaylistID: "PLfixture", SourceURL: "https://www.youtube.com/playlist?list=PLfixture",
		Title: "Authenticated playlist", Policy: "video:1080p", AuthIntent: intent, Discovered: 8, Ready: 4, AuthRequired: 1, Unavailable: 2, Invalid: 1, Approved: 2,
		ChildJobIDs: []string{first.ID, second.ID}, CreatedAt: now, UpdatedAt: now,
	}
	if err := state.Transaction(nil, func(document *jobmodel.State) error {
		document.AuthSourceBindings = []authsource.Binding{binding}
		document.Settings.BrowserAccessEnabled = true
		document.Settings.BrowserAccessConsentVersion = authsource.CurrentConsentVersion
		document.Settings.DefaultAuthSourceBindingRef = binding.ID
		document.Jobs = []jobmodel.DurableJob{first, second}
		document.Collections = []jobmodel.DurableCollection{collection}
		document.NextQueueOrdinal = 3
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if err := state.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, reopenedStatus, err := OpenV2(path)
	if err != nil || !reopenedStatus.Healthy() {
		t.Fatalf("restart OpenV2() = %#v, %#v, %v", reopened, reopenedStatus, err)
	}
	defer reopened.Close()
	got := reopened.Snapshot()
	if len(got.Collections) != 1 || got.Collections[0].AuthIntent != intent || got.Collections[0].Approved != 2 || len(got.Jobs) != 2 || got.Jobs[0].SourceIndex != 2 || got.Jobs[1].SourceIndex != 7 || got.Jobs[1].AuthIntent != intent {
		t.Fatalf("restarted authenticated playlist = %#v, %#v", got.Collections, got.Jobs)
	}
}

func TestStateV2AuthenticatedIntentContainsNoSecretFields(t *testing.T) {
	state := defaultStateV2()
	binding := authsource.Binding{ID: "binding-1", Enabled: true, Descriptor: authsource.Descriptor{SchemaVersion: 1, Platform: "darwin", Browser: authsource.BrowserSafari}}
	state.AuthSourceBindings = []authsource.Binding{binding}
	state.Settings.BrowserAccessEnabled = true
	state.Settings.BrowserAccessConsentVersion = authsource.CurrentConsentVersion
	state.Settings.DefaultAuthSourceBindingRef = binding.ID
	job := testJob()
	job.AuthIntent = jobmodel.AuthIntent{RequiresAuthenticatedExecution: true, AuthSourceBindingRef: binding.ID}
	state.Jobs = []jobmodel.DurableJob{job}
	state.NextQueueOrdinal = 2
	if err := validateState(state); err != nil {
		t.Fatalf("validateState(): %v", err)
	}
	data, err := json.Marshal(state)
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"COOKIE_CANARY_7f2", "AUTH_CANARY", "SIGNED_URL_CANARY", "KEYCHAIN_CANARY", "/Users/private-canary", "cookieFile", "cookiesFromBrowser", "authorization"} {
		if strings.Contains(strings.ToLower(string(data)), strings.ToLower(forbidden)) {
			t.Fatalf("serialized State v2 contains forbidden %q: %s", forbidden, data)
		}
	}
}
