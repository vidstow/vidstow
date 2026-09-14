package follow

import (
	"strings"
	"testing"

	"github.com/tejasa97/vidstow/internal/jobs"
)

func TestApplyCheckTreatsBaselineAsNotNew(t *testing.T) {
	entries := []jobs.PlaylistEntrySummary{
		{Index: 1, VideoID: "aaaaaaaaaaa", URL: "https://www.youtube.com/watch?v=aaaaaaaaaaa", Title: "One", Available: true},
		{Index: 2, VideoID: "bbbbbbbbbbb", URL: "https://www.youtube.com/watch?v=bbbbbbbbbbb", Title: "Two", Available: true},
	}
	rec := Record{
		ID: "fol-1", PlaylistID: "PLfixture", SourceURL: "https://www.youtube.com/playlist?list=PLfixture",
		Title: "Course", VideoCount: 2, KnownVideoIDs: BaselineKnownIDs(entries),
		Output:    Output{Quality: jobs.Quality1080p, Folder: "/tmp/Course"},
		CreatedAt: "t0", UpdatedAt: "t0",
	}
	next := ApplyCheck(rec, Listing{Title: "Course", VideoCount: 2, Entries: entries}, "t1")
	if len(next.Pending) != 0 {
		t.Fatalf("pending = %#v, want none for the follow baseline", next.Pending)
	}
	if next.LastCheckedAt != "t1" || next.LastCheckError != "" {
		t.Fatalf("check metadata = %#v", next)
	}
}

func TestApplyCheckAddsOnlyNewAvailableVideos(t *testing.T) {
	rec := Record{
		ID: "fol-1", PlaylistID: "PLfixture", SourceURL: "https://www.youtube.com/playlist?list=PLfixture",
		Title: "Course", VideoCount: 1, KnownVideoIDs: []string{"aaaaaaaaaaa"},
		Output:    Output{Quality: jobs.Quality1080p, Folder: "/tmp/Course"},
		CreatedAt: "t0", UpdatedAt: "t0",
	}
	listing := Listing{Title: "Course", VideoCount: 3, Entries: []jobs.PlaylistEntrySummary{
		{Index: 1, VideoID: "aaaaaaaaaaa", URL: "https://www.youtube.com/watch?v=aaaaaaaaaaa", Title: "One", Available: true},
		{Index: 2, VideoID: "bbbbbbbbbbb", URL: "https://www.youtube.com/watch?v=bbbbbbbbbbb", Title: "Two", Available: true, Duration: "1:00"},
		{Index: 3, VideoID: "ccccccccccc", Title: "Members only", Available: false},
	}}
	next := ApplyCheck(rec, listing, "t1")
	if len(next.Pending) != 1 || next.Pending[0].VideoID != "bbbbbbbbbbb" || !next.Pending[0].Available {
		t.Fatalf("pending = %#v", next.Pending)
	}
	if !containsID(next.KnownVideoIDs, "bbbbbbbbbbb") || !containsID(next.KnownVideoIDs, "ccccccccccc") {
		t.Fatalf("known = %v, want new ids recorded even when unavailable", next.KnownVideoIDs)
	}
	again := ApplyCheck(next, listing, "t2")
	if len(again.Pending) != 1 {
		t.Fatalf("second check pending = %#v, want no duplicates", again.Pending)
	}
}

func TestSkipRestoreAndAdmitKeepKnownIDs(t *testing.T) {
	rec := Record{
		ID: "fol-1", PlaylistID: "PLfixture", SourceURL: "https://www.youtube.com/playlist?list=PLfixture",
		Title: "Course", VideoCount: 2, KnownVideoIDs: []string{"aaaaaaaaaaa", "bbbbbbbbbbb"},
		Pending: []Item{
			{VideoID: "aaaaaaaaaaa", URL: "https://www.youtube.com/watch?v=aaaaaaaaaaa", Title: "One", Available: true, Index: 1},
			{VideoID: "bbbbbbbbbbb", URL: "https://www.youtube.com/watch?v=bbbbbbbbbbb", Title: "Two", Available: true, Index: 2},
		},
		Output:    Output{Quality: jobs.Quality1080p, Folder: "/tmp/Course"},
		CreatedAt: "t0", UpdatedAt: "t0",
	}
	skipped, err := SkipItems(rec, []string{"aaaaaaaaaaa"}, "t1")
	if err != nil || len(skipped.Pending) != 1 || len(skipped.Skipped) != 1 {
		t.Fatalf("skip = %#v err=%v", skipped, err)
	}
	listing := Listing{Entries: []jobs.PlaylistEntrySummary{
		{Index: 1, VideoID: "aaaaaaaaaaa", URL: "https://www.youtube.com/watch?v=aaaaaaaaaaa", Title: "One", Available: true},
		{Index: 2, VideoID: "bbbbbbbbbbb", URL: "https://www.youtube.com/watch?v=bbbbbbbbbbb", Title: "Two", Available: true},
	}}
	checked := ApplyCheck(skipped, listing, "t2")
	if len(checked.Pending) != 1 || checked.Pending[0].VideoID != "bbbbbbbbbbb" || len(checked.Skipped) != 1 {
		t.Fatalf("check after skip = pending %#v skipped %#v", checked.Pending, checked.Skipped)
	}
	restored, err := RestoreItems(checked, []string{"aaaaaaaaaaa"}, "t3")
	if err != nil || len(restored.Pending) != 2 || len(restored.Skipped) != 0 {
		t.Fatalf("restore = %#v err=%v", restored, err)
	}
	admitted, err := AdmitPending(restored, []string{"bbbbbbbbbbb"}, "t4")
	if err != nil || len(admitted.Pending) != 1 || admitted.Pending[0].VideoID != "aaaaaaaaaaa" {
		t.Fatalf("admit = %#v err=%v", admitted, err)
	}
	if !containsID(admitted.KnownVideoIDs, "bbbbbbbbbbb") {
		t.Fatal("admit dropped a known video id")
	}
}

func TestIndexesForVideoIDsPreservePlaylistOrder(t *testing.T) {
	entries := []jobs.PlaylistEntrySummary{
		{Index: 1, VideoID: "aaaaaaaaaaa", URL: "https://www.youtube.com/watch?v=aaaaaaaaaaa", Available: true},
		{Index: 2, VideoID: "bbbbbbbbbbb", Title: "Gone", Available: false},
		{Index: 3, VideoID: "ccccccccccc", URL: "https://www.youtube.com/watch?v=ccccccccccc", Available: true},
	}
	indexes, err := IndexesForVideoIDs(entries, []string{"ccccccccccc", "aaaaaaaaaaa"})
	if err != nil || len(indexes) != 2 || indexes[0] != 1 || indexes[1] != 3 {
		t.Fatalf("indexes = %v err=%v", indexes, err)
	}
	if _, err := IndexesForVideoIDs(entries, []string{"bbbbbbbbbbb"}); err == nil || !strings.Contains(err.Error(), "available") {
		t.Fatalf("unavailable error = %v", err)
	}
}

func containsID(ids []string, want string) bool {
	for _, id := range ids {
		if id == want {
			return true
		}
	}
	return false
}
