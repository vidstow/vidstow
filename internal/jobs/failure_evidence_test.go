package jobs

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/tejasa97/vidstow/internal/jobmodel"
	"github.com/tejasa97/ytdlp-go/engine"
)

func TestFailureEvidencePersistsAcrossQueueRestore(t *testing.T) {
	store, _, _ := newV2TestStore(t, "failed", "canceled", "completed")
	document := store.Snapshot()
	for i := range document.Jobs {
		document.Jobs[i].Lifecycle = jobmodel.LifecyclePaused
		document.Jobs[i].Plan.Label = "480p"
	}
	document.Jobs[1].Lifecycle = jobmodel.LifecycleCanceled
	document.Jobs[1].Desired = jobmodel.DesiredCanceled
	document.Jobs[2].Lifecycle = jobmodel.LifecycleCompleted
	store.state = document
	m := New(nil, nil)
	defer m.Close()
	if err := m.SetStateStore(store); err != nil {
		t.Fatal(err)
	}
	if err := m.RestoreStateV2(store.Snapshot()); err != nil {
		t.Fatal(err)
	}
	evidence := failureEvidence(&engine.Error{Category: engine.ErrorAuthentication, Op: "extract", Err: errors.New("private https://example.test/?token=secret /private/path")}, false, false)
	state := m.all["failed"]
	terminal := state.snap
	terminal.Status = StatusFailed
	terminal.ErrorReason = "authentication"
	terminal.FailureEvidence = evidence
	err := m.settleDurable(state, jobmodel.LifecycleFailed, jobmodel.DesiredRunning, jobmodel.PhasePreparing, terminal, "authentication", false, 0)
	if err != nil {
		t.Fatal(err)
	}
	saved := store.Snapshot()
	serialized, err := json.Marshal(saved)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(serialized), "secret") || strings.Contains(string(serialized), "/private/path") {
		t.Fatal("raw error leaked into durable state")
	}
	restored := New(nil, nil)
	defer restored.Close()
	if err := restored.SetStateStore(store); err != nil {
		t.Fatal(err)
	}
	if err := restored.RestoreStateV2(saved); err != nil {
		t.Fatal(err)
	}
	view := restored.QueueView()
	if view.Summary.OccupiedSlots != 0 || view.Summary.WaitingJobs != 0 {
		t.Fatalf("historical jobs became scheduled: %+v", view.Summary)
	}
	for _, row := range view.Rows {
		if row.ID == "failed" {
			if row.QualityLabel != "480p" || row.Failure == nil || row.Failure.Evidence == nil || *row.Failure.Evidence != *evidence || !row.Capabilities.Retry {
				t.Fatalf("restored failure = %+v", row)
			}
		}
		if row.ID == "canceled" && (row.Capabilities.Resume || !row.Capabilities.Remove) {
			t.Fatalf("canceled capabilities = %+v", row.Capabilities)
		}
	}
}

func TestFailureEvidenceClassifiesStageWithoutRawText(t *testing.T) {
	for _, tc := range []struct {
		download, processing bool
		stage                string
	}{{false, false, "extraction"}, {true, false, "download"}, {true, true, "processing"}} {
		got := failureEvidence(&engine.Error{Category: engine.ErrorAuthentication, Op: "extract", Err: errors.New("secret")}, tc.download, tc.processing)
		if got.Stage != tc.stage || got.Code != "authentication" || got.At.IsZero() {
			t.Fatalf("evidence = %+v", got)
		}
	}
}
