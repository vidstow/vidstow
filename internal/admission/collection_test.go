package admission

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/tejasa97/vidstow/internal/authsource"
	"github.com/tejasa97/vidstow/internal/jobmodel"
	"github.com/tejasa97/vidstow/internal/jobs"
	"github.com/tejasa97/vidstow/internal/outputplan"
	"github.com/tejasa97/vidstow/internal/reservationfs"
	"github.com/tejasa97/vidstow/internal/store"
	"github.com/tejasa97/ytdlp-go/engine/value"
)

func collectionFixture(t *testing.T) (*Coordinator, *reservationfs.Root, *store.V2Store, *recordingQueue, CollectionRequest) {
	t.Helper()
	outputDir := t.TempDir()
	root, err := reservationfs.OpenRoot(outputDir)
	if err != nil {
		t.Fatal(err)
	}
	state, status, err := store.OpenV2(filepath.Join(t.TempDir(), "state.json"))
	if err != nil || !status.Healthy() {
		t.Fatalf("OpenV2: %v, %#v", err, status)
	}
	t.Cleanup(func() { _ = root.Close(); _ = state.Close() })
	plan := outputplan.Plan{ID: "video-1080-mp4", Kind: outputplan.KindVideo, Label: "1080p", Container: "MP4", Selector: "v1+a1"}
	queue := &recordingQueue{store: state}
	ids := [][3]string{
		{"job-1", "attempt-1", "0123456789abcdef0123456789abcdef"},
		{"job-2", "attempt-2", "abcdef0123456789abcdef0123456789"},
	}
	next := 0
	coordinator, err := NewCoordinator(Dependencies{
		Store: state, Resolver: staticPlanResolver{plan: plan}, Queue: queue,
		Now:             func() time.Time { return time.Date(2026, 8, 14, 7, 0, 0, 0, time.UTC) },
		NewCollectionID: func() string { return "collection-1" },
		NewIDs: func() (string, string, string) {
			current := ids[next]
			next++
			return current[0], current[1], current[2]
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	children := make([]CollectionChildRequest, 2)
	for index := range children {
		children[index] = CollectionChildRequest{
			Request: Request{Queue: jobs.Request{
				URL: "https://www.youtube.com/watch?v=fixture0001", VideoID: "fixture0001",
				Title: "Repeated title", Channel: "Channel", PlanID: plan.ID,
				Quality: jobs.Quality1080p, OutputDir: outputDir,
			}, Metadata: value.NewInfo(value.NewObject())},
			ResolvedPlan: plan,
		}
	}
	request := CollectionRequest{
		Collection: Collection{PlaylistID: "PLfixture", SourceURL: "https://www.youtube.com/playlist?list=PLfixture", Title: "Fixture playlist", Channel: "Channel", Policy: "video:1080p"},
		Children:   children,
	}
	return coordinator, root, state, queue, request
}

func TestCoordinatorAdmitsCollectionAtomicallyBeforeFIFO(t *testing.T) {
	coordinator, root, state, queue, request := collectionFixture(t)
	result, err := coordinator.AdmitCollection(context.Background(), root, request)
	if err != nil {
		t.Fatal(err)
	}
	if result.Collection.ID != "collection-1" || result.Submitted != 2 || len(result.Children) != 2 || len(queue.ids) != 2 {
		t.Fatalf("result = %#v, queue ids = %#v", result, queue.ids)
	}
	snapshot := state.Snapshot()
	if len(snapshot.Collections) != 1 || len(snapshot.Jobs) != 2 || snapshot.NextQueueOrdinal != 3 {
		t.Fatalf("state = %#v", snapshot)
	}
	for index, childID := range snapshot.Collections[0].ChildJobIDs {
		job := snapshot.Jobs[index]
		if childID != job.ID || job.CollectionID != result.Collection.ID || job.CollectionIndex != index+1 || job.QueueOrdinal != uint64(index+1) {
			t.Fatalf("child[%d] = %#v", index, job)
		}
	}
	first := result.Children[0].Reservation.Artifacts[0].Basename
	second := result.Children[1].Reservation.Artifacts[0].Basename
	if first == second || second != "Repeated title [fixture0001] [1080p] (2).mp4" {
		t.Fatalf("reservations = %q, %q", first, second)
	}
}

func TestCoordinatorAdmitsBatchChildrenAcrossDistinctReservationRoots(t *testing.T) {
	coordinator, root, state, _, request := collectionFixture(t)
	request.Collection = Collection{Kind: jobmodel.CollectionKindBatch, Title: "Batch download · 2 videos", Policy: "video:1080p"}
	secondDir := t.TempDir()
	secondRoot, err := reservationfs.OpenRoot(secondDir)
	if err != nil {
		t.Fatal(err)
	}
	defer secondRoot.Close()
	request.Children[1].Root = secondRoot
	request.Children[1].Request.Queue.OutputDir = secondDir

	result, err := coordinator.AdmitCollection(context.Background(), root, request)
	if err != nil {
		t.Fatal(err)
	}
	if result.Collection.Kind != jobmodel.CollectionKindBatch || result.Collection.SourceURL != "" || result.Collection.PlaylistID != "" {
		t.Fatalf("batch collection = %#v", result.Collection)
	}
	snapshot := state.Snapshot()
	if len(snapshot.Jobs) != 2 || snapshot.Jobs[0].OutputRoot.CanonicalPath == snapshot.Jobs[1].OutputRoot.CanonicalPath || snapshot.Jobs[1].OutputRoot.CanonicalPath != secondDir {
		t.Fatalf("batch child roots = %#v, %#v", snapshot.Jobs[0].OutputRoot, snapshot.Jobs[1].OutputRoot)
	}
}

func TestCoordinatorCollectionManagerFailureLeavesWholeDurableCollection(t *testing.T) {
	coordinator, root, state, queue, request := collectionFixture(t)
	sentinel := errors.New("queue unavailable")
	queue.err = sentinel
	result, err := coordinator.AdmitCollection(context.Background(), root, request)
	if !errors.Is(err, sentinel) || result.Submitted != 0 {
		t.Fatalf("result=%#v error=%v", result, err)
	}
	snapshot := state.Snapshot()
	if len(snapshot.Collections) != 1 || len(snapshot.Jobs) != 2 {
		t.Fatalf("durable state = %#v", snapshot)
	}
	for _, job := range snapshot.Jobs {
		if job.Lifecycle != jobmodel.LifecyclePending || job.CollectionID != snapshot.Collections[0].ID {
			t.Fatalf("pending child = %#v", job)
		}
	}
}

func TestCoordinatorAdmitsAuthenticatedCollectionWithOneSharedBinding(t *testing.T) {
	coordinator, root, state, queue, request := collectionFixture(t)
	const bindingRef = "browser-binding"
	if err := state.Transaction(nil, func(document *jobmodel.State) error {
		document.AuthSourceBindings = []authsource.Binding{{ID: bindingRef, Enabled: true, Descriptor: authsource.Descriptor{SchemaVersion: 1, Platform: "darwin", Browser: authsource.BrowserChrome}}}
		document.Settings.BrowserAccessEnabled = true
		document.Settings.BrowserAccessConsentVersion = authsource.CurrentConsentVersion
		document.Settings.DefaultAuthSourceBindingRef = bindingRef
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	intent := jobmodel.AuthIntent{RequiresAuthenticatedExecution: true, AuthSourceBindingRef: bindingRef}
	request.Collection.AuthIntent = intent
	request.Collection.Discovered = 5
	request.Collection.Ready = 2
	request.Collection.AuthRequired = 1
	request.Collection.Unavailable = 1
	request.Collection.Invalid = 1
	request.Collection.Approved = 2
	for index := range request.Children {
		request.Children[index].Request.AuthIntent = intent
		request.Children[index].Request.SourceIndex = index*4 + 1
	}
	request.Children[0].Request.SourceOccurrenceID = "11111111-1111-4111-8111-111111111111"
	request.Children[1].Request.SourceOccurrenceID = "22222222-2222-4222-8222-222222222222"
	result, err := coordinator.AdmitCollection(context.Background(), root, request)
	if err != nil {
		t.Fatal(err)
	}
	if result.Submitted != 2 || len(queue.ids) != 2 {
		t.Fatalf("authenticated result = %#v, queue=%#v", result, queue.ids)
	}
	snapshot := state.Snapshot()
	if len(snapshot.Collections) != 1 || snapshot.Collections[0].AuthIntent != intent || snapshot.Collections[0].Discovered != 5 || snapshot.Collections[0].Approved != 2 {
		t.Fatalf("durable authenticated collection = %#v", snapshot.Collections)
	}
	if len(snapshot.Jobs) != 2 || snapshot.Jobs[0].AuthIntent != intent || snapshot.Jobs[1].AuthIntent != intent || snapshot.Jobs[0].SourceIndex != 1 || snapshot.Jobs[1].SourceIndex != 5 {
		t.Fatalf("durable authenticated children = %#v", snapshot.Jobs)
	}
}

func TestCoordinatorRejectsMixedAuthenticatedCollectionAtomically(t *testing.T) {
	coordinator, root, state, queue, request := collectionFixture(t)
	request.Children[0].Request.AuthIntent = jobmodel.AuthIntent{RequiresAuthenticatedExecution: true, AuthSourceBindingRef: "browser-binding"}
	if _, err := coordinator.AdmitCollection(context.Background(), root, request); err == nil {
		t.Fatal("mixed authenticated collection admission unexpectedly succeeded")
	}
	if snapshot := state.Snapshot(); len(snapshot.Collections) != 0 || len(snapshot.Jobs) != 0 || len(queue.ids) != 0 {
		t.Fatalf("mixed collection mutated state: %#v", snapshot)
	}
}

func TestCoordinatorRejectsCollectionBeforeAnyDurableMutation(t *testing.T) {
	coordinator, root, state, queue, request := collectionFixture(t)
	request.Children[1].Request.Queue.PlanID = "invented-plan"
	if _, err := coordinator.AdmitCollection(context.Background(), root, request); err == nil {
		t.Fatal("AdmitCollection error = nil")
	}
	if snapshot := state.Snapshot(); len(snapshot.Collections) != 0 || len(snapshot.Jobs) != 0 || len(queue.ids) != 0 {
		t.Fatalf("state mutated after rejection: %#v", snapshot)
	}
}
