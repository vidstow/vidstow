package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/tejasa97/vidstow/internal/follow"
	"github.com/tejasa97/vidstow/internal/jobs"
)

func TestFollowPlaylistPersistsFutureOnlyWithoutQueueing(t *testing.T) {
	app := newFollowApp(t)
	restoreListing := stubFollowListing(t, courseListing(2))
	defer restoreListing()

	view, err := app.FollowPlaylist(FollowPlaylistRequest{
		URL: "https://www.youtube.com/playlist?list=PLfixture", PlaylistID: "PLfixture",
		Quality: jobs.Quality1080p, Scope: follow.ScopeFuture,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(view.Follows) != 1 || view.Follows[0].PlaylistID != "PLfixture" || len(view.Follows[0].Pending) != 0 {
		t.Fatalf("view = %#v", view)
	}
	if len(view.Follows[0].KnownVideoIDs) != 2 {
		t.Fatalf("known = %v", view.Follows[0].KnownVideoIDs)
	}
	if len(app.store.Snapshot().Collections) != 0 || len(app.store.Snapshot().Jobs) != 0 {
		t.Fatal("future-only follow admitted a queue collection")
	}
}

func TestListFollowsDoesNotCheckPlaylists(t *testing.T) {
	app := newFollowApp(t)
	calls := 0
	restoreListing := stubFollowListingFn(t, func(_ *App, _ context.Context, _, _ string, refresh bool) (jobs.PlaylistSummary, error) {
		calls++
		listing := courseListing(2)
		if refresh {
			listing = courseListing(3)
		}
		return listing, nil
	})
	defer restoreListing()
	if _, err := app.FollowPlaylist(FollowPlaylistRequest{
		URL: "https://www.youtube.com/playlist?list=PLfixture", PlaylistID: "PLfixture", Quality: jobs.Quality1080p,
	}); err != nil {
		t.Fatal(err)
	}
	followCalls := calls
	_ = app.ListFollows()
	if calls != followCalls {
		t.Fatalf("ListFollows triggered a listing refresh: calls %d -> %d", followCalls, calls)
	}
}

func TestCheckFollowAddsNewVideosAndKeepsFindsOnFailure(t *testing.T) {
	app := newFollowApp(t)
	listings := []jobs.PlaylistSummary{courseListing(2)}
	restoreListing := stubFollowListingFn(t, func(_ *App, _ context.Context, _, _ string, refresh bool) (jobs.PlaylistSummary, error) {
		if !refresh {
			return listings[0], nil
		}
		if len(listings) == 1 {
			return courseListing(3), nil
		}
		return jobs.PlaylistSummary{}, context.DeadlineExceeded
	})
	defer restoreListing()
	if _, err := app.FollowPlaylist(FollowPlaylistRequest{
		URL: "https://www.youtube.com/playlist?list=PLfixture", PlaylistID: "PLfixture", Quality: jobs.Quality1080p,
	}); err != nil {
		t.Fatal(err)
	}
	view, err := app.CheckFollow(app.ListFollows().Follows[0].ID)
	if err != nil || len(view.Follows[0].Pending) != 1 || view.Follows[0].Pending[0].VideoID != "ccccccccccc" {
		t.Fatalf("check = %#v err=%v", view, err)
	}
	listings = append(listings, jobs.PlaylistSummary{})
	failed, err := app.CheckFollow(view.Follows[0].ID)
	if err == nil || failed.Follows[0].LastCheckError != followCheckFailedMessage || len(failed.Follows[0].Pending) != 1 {
		t.Fatalf("failed check = %#v err=%v", failed, err)
	}
	if failed.Follows[0].LastCheckedAt == "" {
		t.Fatal("failed check cleared the last successful check time")
	}
}

func TestStopFollowChecksCancelsInFlight(t *testing.T) {
	app := newFollowApp(t)
	started := make(chan struct{})
	restoreListing := stubFollowListingFn(t, func(_ *App, ctx context.Context, _, _ string, refresh bool) (jobs.PlaylistSummary, error) {
		if !refresh {
			return courseListing(1), nil
		}
		close(started)
		<-ctx.Done()
		return jobs.PlaylistSummary{}, ctx.Err()
	})
	defer restoreListing()
	if _, err := app.FollowPlaylist(FollowPlaylistRequest{
		URL: "https://www.youtube.com/playlist?list=PLfixture", PlaylistID: "PLfixture", Quality: jobs.Quality1080p,
	}); err != nil {
		t.Fatal(err)
	}
	errCh := make(chan error, 1)
	go func() {
		_, err := app.CheckAllFollows()
		errCh <- err
	}()
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("check never started")
	}
	_ = app.StopFollowChecks()
	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("stopped check error = %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("check did not stop")
	}
	if got := app.ListFollows(); got.Checking || len(got.Follows) != 1 || got.Follows[0].LastCheckError != "" {
		t.Fatalf("stopped view = %#v", got)
	}
}

func TestUnfollowLeavesQueueJobsAndFiles(t *testing.T) {
	app := newFollowApp(t)
	restoreListing := stubFollowListing(t, courseListing(1))
	defer restoreListing()
	view, err := app.FollowPlaylist(FollowPlaylistRequest{
		URL: "https://www.youtube.com/playlist?list=PLfixture", PlaylistID: "PLfixture", Quality: jobs.Quality1080p,
	})
	if err != nil {
		t.Fatal(err)
	}
	snapshot := app.store.Snapshot()
	if _, err := app.UnfollowPlaylist(view.Follows[0].ID); err != nil {
		t.Fatal(err)
	}
	after := app.store.Snapshot()
	if len(after.Jobs) != len(snapshot.Jobs) || len(after.Collections) != len(snapshot.Collections) || len(after.History) != len(snapshot.History) {
		t.Fatal("unfollow mutated queue state")
	}
	if len(app.ListFollows().Follows) != 0 {
		t.Fatal("unfollow left the follow record")
	}
}

func TestAdmitFollowReviewRejectsUnknownVideos(t *testing.T) {
	app := newFollowApp(t)
	restoreListing := stubFollowListing(t, courseListing(3))
	defer restoreListing()
	view, err := app.FollowPlaylist(FollowPlaylistRequest{
		URL: "https://www.youtube.com/playlist?list=PLfixture", PlaylistID: "PLfixture", Quality: jobs.Quality1080p,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := app.CheckFollow(view.Follows[0].ID); err != nil {
		t.Fatal(err)
	}
	_, err = app.AdmitFollowReview(FollowReviewRequest{FollowID: view.Follows[0].ID, VideoIDs: []string{"zzzzzzzzzzz"}})
	if err == nil || !strings.Contains(err.Error(), "no longer in this playlist") {
		t.Fatalf("admit error = %v", err)
	}
}

func newFollowApp(t *testing.T) *App {
	t.Helper()
	restore := installAppTestSeams(t)
	t.Cleanup(restore)
	root := secureAppTempDir(t)
	downloads := filepath.Join(root, "downloads")
	if err := os.Mkdir(downloads, 0o700); err != nil {
		t.Fatal(err)
	}
	app := NewApp()
	app.startupAt(context.Background(), filepath.Join(root, "state.json"))
	if app.jobs == nil || app.store == nil || app.follows == nil {
		t.Fatal("startup did not initialize follow persistence")
	}
	t.Cleanup(func() {
		app.stopCleanup(context.Background())
		_ = app.jobs.Close(context.Background())
		_ = app.store.Close()
	})
	settings := app.store.Settings()
	settings.DownloadFolder = downloads
	if err := app.store.SetSettings(settings); err != nil {
		t.Fatal(err)
	}
	return app
}

func stubFollowListing(t *testing.T, listing jobs.PlaylistSummary) func() {
	t.Helper()
	return stubFollowListingFn(t, func(_ *App, _ context.Context, _, _ string, _ bool) (jobs.PlaylistSummary, error) {
		return listing, nil
	})
}

func stubFollowListingFn(t *testing.T, fn func(*App, context.Context, string, string, bool) (jobs.PlaylistSummary, error)) func() {
	t.Helper()
	previous := loadFollowListing
	loadFollowListing = fn
	return func() { loadFollowListing = previous }
}

func courseListing(count int) jobs.PlaylistSummary {
	ids := []string{"aaaaaaaaaaa", "bbbbbbbbbbb", "ccccccccccc"}
	entries := make([]jobs.PlaylistEntrySummary, 0, count)
	for i := 0; i < count; i++ {
		entries = append(entries, jobs.PlaylistEntrySummary{
			Index: i + 1, VideoID: ids[i], URL: "https://www.youtube.com/watch?v=" + ids[i],
			Title: "Video " + ids[i], Available: true, Duration: "1:00",
		})
	}
	return jobs.PlaylistSummary{
		ID: "PLfixture", URL: "https://www.youtube.com/playlist?list=PLfixture",
		Title: "Course", Channel: "Teacher", EntryCount: count, Available: count, Entries: entries,
	}
}
