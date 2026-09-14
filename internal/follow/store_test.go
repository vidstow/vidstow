package follow

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/tejasa97/vidstow/internal/jobs"
)

func TestStorePersistsFollowsWithoutTouchingQueueFiles(t *testing.T) {
	dir := t.TempDir()
	statePath := filepath.Join(dir, "state.json")
	if err := os.WriteFile(statePath, []byte(`{"untouched":true}`), 0o600); err != nil {
		t.Fatal(err)
	}
	store, err := Open(PathForState(statePath))
	if err != nil {
		t.Fatal(err)
	}
	rec := validRecord()
	if err := store.Put(rec); err != nil {
		t.Fatal(err)
	}
	reopened, err := Open(PathForState(statePath))
	if err != nil {
		t.Fatal(err)
	}
	got, ok := reopened.ByPlaylist("PLfixture")
	if !ok || got.Title != "Course" || len(got.KnownVideoIDs) != 1 {
		t.Fatalf("reloaded = %#v ok=%v", got, ok)
	}
	if _, err := reopened.Delete(got.ID); err != nil {
		t.Fatal(err)
	}
	if _, ok := reopened.Get(got.ID); ok {
		t.Fatal("deleted follow is still listed")
	}
	raw, err := os.ReadFile(statePath)
	if err != nil || string(raw) != `{"untouched":true}` {
		t.Fatalf("queue file changed: %q err=%v", raw, err)
	}
}

func TestStoreRejectsASecondFollowForTheSamePlaylist(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "follows.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Put(validRecord()); err != nil {
		t.Fatal(err)
	}
	dup := validRecord()
	dup.ID = "fol-2"
	if err := store.Put(dup); err == nil {
		t.Fatal("duplicate playlist follow succeeded")
	}
}

func TestOpenQuarantinesCorruptFollowsWithoutFailing(t *testing.T) {
	path := filepath.Join(t.TempDir(), "follows.json")
	if err := os.WriteFile(path, []byte(`{"nope":true}`), 0o600); err != nil {
		t.Fatal(err)
	}
	store, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(store.List()) != 0 {
		t.Fatal("corrupt follows were loaded")
	}
	if _, err := os.Stat(path + ".bad"); err != nil {
		t.Fatalf("corrupt file was not quarantined: %v", err)
	}
}

func validRecord() Record {
	return Record{
		ID: "fol-1", PlaylistID: "PLfixture", SourceURL: "https://www.youtube.com/playlist?list=PLfixture",
		Title: "Course", VideoCount: 1, KnownVideoIDs: []string{"aaaaaaaaaaa"},
		Output:    Output{Quality: jobs.Quality1080p, Folder: "/tmp/Course"},
		CreatedAt: "t0", UpdatedAt: "t0",
	}
}
