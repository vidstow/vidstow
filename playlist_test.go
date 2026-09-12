package main

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/tejasa97/vidstow/internal/jobs"
	"github.com/tejasa97/vidstow/internal/outputplan"
	"github.com/tejasa97/vidstow/internal/reservation"
	"github.com/tejasa97/ytdlp-go/engine"
)

type boundedPlaylistAnalyzer struct {
	active atomic.Int32
	max    atomic.Int32
}

func (a *boundedPlaylistAnalyzer) AnalyzeForAdmission(ctx context.Context, rawURL string) (jobs.InfoSummary, []outputplan.Plan, error) {
	active := a.active.Add(1)
	defer a.active.Add(-1)
	for {
		maximum := a.max.Load()
		if active <= maximum || a.max.CompareAndSwap(maximum, active) {
			break
		}
	}
	select {
	case <-time.After(10 * time.Millisecond):
	case <-ctx.Done():
		return jobs.InfoSummary{}, nil, ctx.Err()
	}
	videoID := rawURL[len(rawURL)-11:]
	return jobs.InfoSummary{URL: rawURL, VideoID: videoID, Title: "Title " + videoID}, []outputplan.Plan{{
		ID: "video", Kind: outputplan.KindVideo, Height: 1080, Container: "MP4", Label: "1080p", Available: true, Selector: "v+a",
	}}, nil
}

func TestAnalyzePlaylistChildrenIsBoundedOrderedAndCarriesPrivatePlans(t *testing.T) {
	analyzer := &boundedPlaylistAnalyzer{}
	entries := make([]jobs.PlaylistEntrySummary, 8)
	for index := range entries {
		videoID := "fixture" + string(rune('0'+index)) + "001"
		entries[index] = jobs.PlaylistEntrySummary{Index: index + 1, VideoID: videoID, URL: "https://www.youtube.com/watch?v=" + videoID, Available: true}
	}
	children, err := analyzePlaylistChildren(context.Background(), analyzer, entries, jobs.Quality1080p, 0)
	if err != nil {
		t.Fatal(err)
	}
	if analyzer.max.Load() > playlistAnalysisConcurrency || analyzer.max.Load() < 2 || len(children) != len(entries) {
		t.Fatalf("max concurrency=%d children=%d", analyzer.max.Load(), len(children))
	}
	for index, child := range children {
		if child.entry.Index != index+1 || child.summary.VideoID != entries[index].VideoID || child.plan.Selector != "v+a" {
			t.Fatalf("child[%d] = %#v", index, child)
		}
	}
}

type selectivePlaylistAnalyzer struct {
	failAt map[string]error
}

func (a selectivePlaylistAnalyzer) AnalyzeForAdmission(_ context.Context, rawURL string) (jobs.InfoSummary, []outputplan.Plan, error) {
	videoID := rawURL[len(rawURL)-11:]
	if err, ok := a.failAt[videoID]; ok {
		return jobs.InfoSummary{}, nil, err
	}
	return jobs.InfoSummary{URL: rawURL, VideoID: videoID, Title: "Title " + videoID}, []outputplan.Plan{{
		ID: "video", Kind: outputplan.KindVideo, Height: 1080, Container: "MP4", Label: "1080p", Available: true, Selector: "v+a",
	}}, nil
}

func TestAnalyzePlaylistChildrenSkipsUnsupportedItemsAndKeepsPlaylistOrder(t *testing.T) {
	membersOnly := &engine.Error{
		Category: engine.ErrorUnsupported,
		Op:       "youtube extraction",
		Err:      errors.New("media unavailable: Join this channel to get access to members-only content like this video and other exclusive perks."),
	}
	entries := []jobs.PlaylistEntrySummary{
		{Index: 1, VideoID: "fixture0001", URL: "https://www.youtube.com/watch?v=fixture0001", Available: true},
		{Index: 2, VideoID: "fixture0002", URL: "https://www.youtube.com/watch?v=fixture0002", Available: true},
		{Index: 3, VideoID: "fixture0003", URL: "https://www.youtube.com/watch?v=fixture0003", Available: true},
		{Index: 4, VideoID: "fixture0004", URL: "https://www.youtube.com/watch?v=fixture0004", Available: true},
		{Index: 5, VideoID: "fixture0005", URL: "https://www.youtube.com/watch?v=fixture0005", Available: true},
	}
	children, err := analyzePlaylistChildren(context.Background(), selectivePlaylistAnalyzer{failAt: map[string]error{"fixture0003": membersOnly}}, entries, jobs.Quality1080p, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(children) != 4 {
		t.Fatalf("children=%d, want 4 public videos after skipping members-only item 3", len(children))
	}
	if children[0].entry.Index != 1 || children[1].entry.Index != 2 || children[2].entry.Index != 4 || children[3].entry.Index != 5 {
		t.Fatalf("playlist order after skip = %#v", children)
	}
}

func TestAnalyzePlaylistChildrenAllUnsupportedUsesRawChildError(t *testing.T) {
	unsupported := &engine.Error{
		Category: engine.ErrorUnsupported,
		Op:       "youtube extraction",
		Err:      errors.New("media unavailable: Join this channel to get access to members-only content like this video and other exclusive perks."),
	}
	entries := []jobs.PlaylistEntrySummary{
		{Index: 3, VideoID: "fixture0003", URL: "https://www.youtube.com/watch?v=fixture0003", Available: true},
	}
	_, err := analyzePlaylistChildren(context.Background(), selectivePlaylistAnalyzer{failAt: map[string]error{"fixture0003": unsupported}}, entries, jobs.Quality1080p, 0)
	if err == nil {
		t.Fatal("error = nil")
	}
	mapped := friendlyPlaylistStartError(err)
	if mapped == "That link is not a supported single YouTube video." {
		t.Fatalf("playlist start mapped a child failure through the single-video analyzer copy: %q", mapped)
	}
	if !strings.Contains(mapped, "channel membership") {
		t.Fatalf("playlist start error = %q, want members-only copy", mapped)
	}
}

func TestFriendlyPlaylistStartErrorDoesNotUseSingleVideoCopy(t *testing.T) {
	err := fmt.Errorf("analyze playlist item 3: %w", &engine.Error{
		Category: engine.ErrorUnsupported,
		Op:       "youtube extraction",
		Err:      errors.New("video unavailable"),
	})
	got := friendlyPlaylistStartError(err)
	if got == "That link is not a supported single YouTube video." {
		t.Fatal("playlist start used the single-video unsupported message")
	}
	if !strings.Contains(got, "selected video") {
		t.Fatalf("playlist start error = %q", got)
	}
}

func TestValidatePlaylistPolicy(t *testing.T) {
	for _, test := range []struct {
		quality jobs.Quality
		bitrate int
		want    string
	}{
		{jobs.QualityBest, 0, "video:best"},
		{jobs.Quality1080p, 0, "video:1080p"},
		{jobs.QualityAudioOnly, 0, "audio:original"},
		{jobs.QualityAudioOnly, 192, "audio:mp3-192"},
	} {
		got, err := validatePlaylistPolicy(test.quality, test.bitrate)
		if err != nil || got != test.want {
			t.Fatalf("policy(%q, %d) = %q, %v", test.quality, test.bitrate, got, err)
		}
	}
	for _, test := range []struct {
		quality jobs.Quality
		bitrate int
	}{{jobs.Quality1080p, 128}, {jobs.QualityAudioOnly, 320}, {jobs.Quality("invented"), 0}} {
		if _, err := validatePlaylistPolicy(test.quality, test.bitrate); err == nil {
			t.Fatalf("policy(%q, %d) error = nil", test.quality, test.bitrate)
		}
	}
}

func TestChoosePlaylistPlanUsesPerChildAvailabilityAndCaps(t *testing.T) {
	plans := []outputplan.Plan{
		{ID: "4k", Kind: outputplan.KindVideo, Height: 2160, Available: true},
		{ID: "1080", Kind: outputplan.KindVideo, Height: 1080, Available: true},
		{ID: "720", Kind: outputplan.KindVideo, Height: 720, Available: true},
		{ID: "original", Kind: outputplan.KindAudio, Available: true},
		{ID: "mp3", Kind: outputplan.KindAudio, Available: true, RequiresFFmpeg: true, AudioBitrateKbps: 192},
	}
	for _, test := range []struct {
		quality jobs.Quality
		bitrate int
		want    string
	}{
		{jobs.QualityBest, 0, "4k"}, {jobs.Quality1080p, 0, "1080"},
		{jobs.Quality720p, 0, "720"}, {jobs.QualityAudioOnly, 0, "original"},
		{jobs.QualityAudioOnly, 192, "mp3"},
	} {
		plan, err := choosePlaylistPlan(plans, test.quality, test.bitrate)
		if err != nil || plan.ID != test.want {
			t.Fatalf("choose(%q, %d) = %#v, %v", test.quality, test.bitrate, plan, err)
		}
	}
	if _, err := choosePlaylistPlan(plans, jobs.Quality1440p, 0); err != nil {
		t.Fatalf("1440p policy should fall back to the best child plan under the cap: %v", err)
	}
	if _, err := choosePlaylistPlan(plans, jobs.QualityAudioOnly, 256); err == nil {
		t.Fatal("unavailable MP3 policy error = nil")
	}
}

func TestPlaylistSubfolderBoundsUntrustedIdentity(t *testing.T) {
	for _, title := range []string{strings.Repeat("title", 50), strings.Repeat("界", 200), `bad:/\\*title`} {
		folder := playlistSubfolder(title, strings.Repeat("x", 500))
		if len(folder) > 200 || strings.ContainsAny(folder, `/\\`) || reservation.ValidateBasename(folder) != nil {
			t.Fatalf("unsafe playlist folder = %q (%d bytes)", folder, len(folder))
		}
	}
}
