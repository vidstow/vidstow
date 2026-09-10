package jobs

import (
	"fmt"
	"os"
	"syscall"
	"testing"

	"github.com/tejasa97/vidstow/internal/jobmodel"
)

func TestQueueFailureProjectionAndCapabilities(t *testing.T) {
	tests := []struct {
		name         string
		code         string
		httpStatus   int
		category     string
		retry        bool
		startAgain   bool
		openSource   bool
		changeFolder bool
		heading      string
		messageKey   string
	}{
		{name: "network", code: "network", category: "network_interrupted", retry: true, messageKey: "queue.failure.network_interrupted"},
		{name: "rate limited", code: "network", httpStatus: 429, category: "rate_limited", retry: true, heading: "YouTube asked VidStow to slow down", messageKey: "queue.failure.rate_limited"},
		{name: "authentication", code: "authentication", category: "authentication_required", retry: true, openSource: true, heading: "Download was refused", messageKey: "queue.failure.authentication_required"},
		{name: "unavailable", code: "unsupported", category: "resource_unavailable", retry: true, openSource: true, messageKey: "queue.failure.resource_unavailable"},
		{name: "invalid input", code: "invalid_input", category: "could_not_start", retry: true, heading: "Download could not start", messageKey: "queue.failure.could_not_start"},
		{name: "destination exists", code: "destination-exists", category: "destination_exists", retry: true, heading: "File already in the folder", messageKey: "queue.failure.destination_exists"},
		{name: "disk full", code: "disk_full", category: "disk_full", startAgain: true, changeFolder: true, heading: "Not enough disk space", messageKey: "queue.failure.disk_full"},
		{name: "permission", code: "permission_denied", category: "permission_denied", startAgain: true, changeFolder: true, heading: "Folder is not writable", messageKey: "queue.failure.permission_denied"},
		{name: "missing folder", code: "output-root-unavailable", category: "folder_unavailable", retry: true, changeFolder: true, heading: "Save folder is missing", messageKey: "queue.failure.folder_unavailable"},
		{name: "security", code: "security", category: "security_blocked", messageKey: "queue.failure.security_blocked"},
		{name: "exhausted", code: retryCodeFreshDownloadRequired, category: "retry_exhausted", messageKey: "queue.failure.retry_exhausted"},
		{name: "internal", code: "internal", category: "internal", retry: true, messageKey: "queue.failure.internal"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			state := &jobState{
				snap:        JobSnapshot{Status: StatusFailed, URL: "https://www.youtube.com/watch?v=dQw4w9WgXcQ", ErrorReason: test.code, Bytes: 10},
				durable:     jobmodel.DurableJob{LastErrorCode: test.code, LastFailureCommittedBytes: 10},
				fromStateV2: true,
			}
			if test.httpStatus != 0 {
				evidence := &jobmodel.FailureEvidence{Stage: "download", Code: test.code, HTTPStatus: test.httpStatus}
				state.snap.FailureEvidence = evidence
				state.durable.LastFailure = evidence
			}
			failure := queueFailureFor(state, state.snap)
			if failure.Category != test.category || failure.MessageKey != test.messageKey || failure.Retryable != test.retry || !failure.PartialOutput {
				t.Fatalf("failure = %#v", failure)
			}
			if failure.Heading == "" || failure.Message == "" || failure.RecommendedAction == "" {
				t.Fatalf("failure copy is incomplete: %#v", failure)
			}
			if test.heading != "" && failure.Heading != test.heading {
				t.Fatalf("heading = %q; want %q", failure.Heading, test.heading)
			}
			caps := queueCapabilitiesFor(state, state.snap)
			if caps.Retry != test.retry || caps.StartAgain != test.startAgain || caps.OpenSource != test.openSource || caps.CopyLink != test.openSource || caps.ChangeFolder != test.changeFolder || !caps.Remove {
				t.Fatalf("capabilities = %#v", caps)
			}
		})
	}
}

func TestFailureReasonUsesWrappedFilesystemErrors(t *testing.T) {
	if got := errorReason(fmt.Errorf("write output: %w", syscall.ENOSPC)); got != "disk_full" {
		t.Fatalf("disk-full reason = %q", got)
	}
	if got := errorReason(fmt.Errorf("publish output: %w", os.ErrPermission)); got != "permission_denied" {
		t.Fatalf("permission reason = %q", got)
	}
	if got := humanError(fmt.Errorf("write /private/path: %w", syscall.ENOSPC)); got != "Not enough disk space" {
		t.Fatalf("disk-full copy = %q", got)
	}
}

func TestQueueStartAgainReleasesDurableRowBeforeReturningURL(t *testing.T) {
	store, _, _ := newV2TestStore(t, "disk-failed")
	store.state.Jobs[0].Lifecycle = jobmodel.LifecycleFailed
	store.state.Jobs[0].LastErrorCode = "disk_full"

	manager := New(nil, nil)
	defer manager.Close()
	if err := manager.SetStateStore(store); err != nil {
		t.Fatal(err)
	}
	if err := manager.RestoreStateV2(store.Snapshot()); err != nil {
		t.Fatal(err)
	}
	view := manager.QueueView()
	if len(view.Rows) != 1 || !view.Rows[0].Capabilities.StartAgain {
		t.Fatalf("queue view = %#v", view.Rows)
	}
	url, err := manager.QueueStartAgainURL("disk-failed", view.Rows[0].CommandToken)
	if err != nil {
		t.Fatal(err)
	}
	if url != "https://www.youtube.com/watch?v=abc123" {
		t.Fatalf("source URL = %q", url)
	}
	if len(store.Snapshot().Jobs) != 0 || len(manager.QueueView().Rows) != 0 {
		t.Fatal("start again returned before the old row and reservation were removed")
	}
	if _, err := manager.QueueStartAgainURL("disk-failed", view.Rows[0].CommandToken); err == nil {
		t.Fatal("consumed start-again authority was replayed")
	}
}
