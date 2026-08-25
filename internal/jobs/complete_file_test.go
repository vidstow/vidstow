package jobs

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tejasa97/vidstow/internal/jobmodel"
	"github.com/tejasa97/ytdlp-go/engine"
)

func completeFileTestReservation(root string) jobmodel.ReservationSet {
	return jobmodel.ReservationSet{
		GroupID: "job-1", Directory: jobmodel.OutputRootRef{CanonicalPath: root},
		Artifacts: []jobmodel.ReservedArtifact{
			{Kind: string(engine.ArtifactKindPrimary), Identity: "primary", Basename: "Video.mp4"},
			{Kind: completeFallbackArtifactKind, Identity: completeMKVIdentity, Basename: "Video.mkv"},
			{Kind: completeSubtitleArtifactKind, Identity: "en", Basename: "Video.srt"},
		},
	}
}

func completeFileFixtureRunner(t *testing.T, failEmbeds int, calls *[]engine.Request) downloadRunner {
	t.Helper()
	return func(ctx context.Context, request engine.Request, handler engine.EventHandler) (engine.Result, error) {
		*calls = append(*calls, request)
		if handler != nil {
			if err := handler(ctx, engine.Event{Kind: engine.EventPostprocessStarting}); err != nil {
				return engine.Result{}, err
			}
		}
		if len(*calls) <= failEmbeds && request.Subtitles.Embed {
			return engine.Result{}, &engine.Error{Category: engine.ErrorInternal, Op: "postprocess", Err: errors.New("fixture mux failure")}
		}
		target := request.Postprocessors[len(request.Postprocessors)-1].Remux.Format
		media := filepath.Join(request.OutputDir, strings.ReplaceAll(request.OutputTemplate, "%(ext)s", target))
		if err := os.WriteFile(media, []byte("playable media"), 0o600); err != nil {
			return engine.Result{}, err
		}
		result := engine.Result{Filename: media, Bytes: 14, Artifacts: []engine.Artifact{{Path: media, Kind: "media"}}}
		if request.Subtitles.KeepFiles || !request.Subtitles.Embed {
			subtitle := filepath.Join(request.OutputDir, "Video.en.srt")
			if err := os.WriteFile(subtitle, []byte("1\n00:00:00,000 --> 00:00:01,000\nHello\n"), 0o600); err != nil {
				return engine.Result{}, err
			}
			result.Artifacts = append(result.Artifacts, engine.Artifact{Path: subtitle, Kind: completeSubtitleArtifactKind})
		}
		return result, nil
	}
}

func TestRunCompleteFilePublishesMP4AndOptionalSRTAtomically(t *testing.T) {
	root := t.TempDir()
	var calls []engine.Request
	request := engine.Request{
		OutputDir:  root,
		Subtitles:  engine.SubtitleOptions{WriteManual: true, Embed: true, KeepFiles: true, ConvertFormat: "srt", Languages: []string{"en"}},
		Thumbnails: engine.ThumbnailOptions{Write: true, Embed: true},
	}
	result, delivery, err := runCompleteFile(context.Background(), request, completeFileTestReservation(root), completeFileFixtureRunner(t, 0, &calls), nil, engine.NewPublicationArbiter())
	if err != nil {
		t.Fatal(err)
	}
	if len(calls) != 1 || delivery.Container != "MP4" || delivery.UsedMKVFallback || !delivery.SubtitleSidecar || delivery.Publication == nil {
		t.Fatalf("calls=%d delivery=%#v", len(calls), delivery)
	}
	delivery.Publication.FinishPublication()
	if result.Filename != filepath.Join(root, "Video.mp4") {
		t.Fatalf("filename = %q", result.Filename)
	}
	for _, name := range []string{"Video.mp4", "Video.srt"} {
		if info, statErr := os.Stat(filepath.Join(root, name)); statErr != nil || !info.Mode().IsRegular() {
			t.Fatalf("published %s: info=%#v err=%v", name, info, statErr)
		}
	}
	if matches, _ := filepath.Glob(filepath.Join(root, ".vidstow-complete-*")); len(matches) != 0 {
		t.Fatalf("workspace junk remains: %v", matches)
	}
	if calls[0].Filesystem.Resume.SessionID != "" || !calls[0].Subtitles.Embed || !calls[0].Thumbnails.Embed {
		t.Fatalf("MP4 attempt = %#v", calls[0])
	}
}

func TestRunCompleteFileFallsBackFromMP4ToMKV(t *testing.T) {
	root := t.TempDir()
	var calls []engine.Request
	request := engine.Request{
		OutputDir:  root,
		Subtitles:  engine.SubtitleOptions{WriteManual: true, Embed: true, ConvertFormat: "vtt"},
		Thumbnails: engine.ThumbnailOptions{Write: true, Embed: true},
	}
	result, delivery, err := runCompleteFile(context.Background(), request, completeFileTestReservation(root), completeFileFixtureRunner(t, 1, &calls), nil, engine.NewPublicationArbiter())
	if err != nil {
		t.Fatal(err)
	}
	if len(calls) != 2 || delivery.Container != "MKV" || !delivery.UsedMKVFallback || delivery.SubtitleSidecar || delivery.Publication == nil {
		t.Fatalf("calls=%d delivery=%#v", len(calls), delivery)
	}
	delivery.Publication.FinishPublication()
	if result.Filename != filepath.Join(root, "Video.mkv") {
		t.Fatalf("filename = %q", result.Filename)
	}
	if _, statErr := os.Stat(filepath.Join(root, "Video.mp4")); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("MP4 partial was published: %v", statErr)
	}
}

func TestRunCompleteFileFallsBackToMKVWithSidecar(t *testing.T) {
	root := t.TempDir()
	var calls []engine.Request
	request := engine.Request{
		OutputDir: root, EmbedMetadata: true,
		Subtitles:  engine.SubtitleOptions{WriteManual: true, Embed: true, ConvertFormat: "vtt", Languages: []string{"en"}},
		Thumbnails: engine.ThumbnailOptions{Write: true, Embed: true},
	}
	_, delivery, err := runCompleteFile(context.Background(), request, completeFileTestReservation(root), completeFileFixtureRunner(t, 2, &calls), nil, engine.NewPublicationArbiter())
	if err != nil {
		t.Fatal(err)
	}
	if len(calls) != 3 || delivery.Container != "MKV" || !delivery.UsedMKVFallback || !delivery.SubtitleSidecar || !delivery.DegradedEmbedding || delivery.Publication == nil {
		t.Fatalf("calls=%d delivery=%#v", len(calls), delivery)
	}
	delivery.Publication.FinishPublication()
	degraded := calls[2]
	if degraded.Subtitles.Embed || degraded.Subtitles.ConvertFormat != "srt" || degraded.EmbedMetadata || degraded.EmbedChapters != nil || degraded.Thumbnails.Embed {
		t.Fatalf("degraded attempt still embeds: %#v", degraded)
	}
	for _, name := range []string{"Video.mkv", "Video.srt"} {
		if _, statErr := os.Stat(filepath.Join(root, name)); statErr != nil {
			t.Fatalf("missing %s: %v", name, statErr)
		}
	}
}

func TestRunCompleteFileNeverOverwritesAnOccupiedReservation(t *testing.T) {
	root := t.TempDir()
	destination := filepath.Join(root, "Video.mp4")
	if err := os.WriteFile(destination, []byte("existing"), 0o600); err != nil {
		t.Fatal(err)
	}
	var calls []engine.Request
	_, _, err := runCompleteFile(
		context.Background(),
		engine.Request{OutputDir: root, Subtitles: engine.SubtitleOptions{Embed: true}},
		completeFileTestReservation(root), completeFileFixtureRunner(t, 0, &calls), nil, engine.NewPublicationArbiter(),
	)
	if err == nil {
		t.Fatal("occupied reservation unexpectedly published")
	}
	contents, readErr := os.ReadFile(destination)
	if readErr != nil || string(contents) != "existing" {
		t.Fatalf("existing destination changed: %q, %v", contents, readErr)
	}
	if matches, _ := filepath.Glob(filepath.Join(root, ".vidstow-complete-*")); len(matches) != 0 {
		t.Fatalf("failed publication left workspace junk: %v", matches)
	}
}

func TestRunCompleteFileHonorsCancelWinnerBeforePublication(t *testing.T) {
	root := t.TempDir()
	arbiter := engine.NewPublicationArbiter()
	cancelReservation, err := arbiter.BeginCancel(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	cancelReservation.WinCancel()
	var calls []engine.Request
	_, delivery, err := runCompleteFile(
		context.Background(),
		engine.Request{OutputDir: root, Subtitles: engine.SubtitleOptions{Embed: true}},
		completeFileTestReservation(root), completeFileFixtureRunner(t, 0, &calls), nil, arbiter,
	)
	if err == nil || delivery.Publication != nil {
		t.Fatalf("cancel winner published: delivery=%#v err=%v", delivery, err)
	}
	for _, name := range []string{"Video.mp4", "Video.mkv", "Video.srt"} {
		if _, statErr := os.Stat(filepath.Join(root, name)); !errors.Is(statErr, os.ErrNotExist) {
			t.Fatalf("cancel winner left public artifact %s: %v", name, statErr)
		}
	}
}

func TestRunCompleteFileDoesNotFallbackForTransferFailure(t *testing.T) {
	root := t.TempDir()
	calls := 0
	runner := func(ctx context.Context, request engine.Request, handler engine.EventHandler) (engine.Result, error) {
		calls++
		if handler != nil {
			_ = handler(ctx, engine.Event{Kind: engine.EventDownloadStarting})
		}
		return engine.Result{}, &engine.Error{Category: engine.ErrorNetwork, Op: "download", Err: errors.New("fixture network failure")}
	}
	_, _, err := runCompleteFile(context.Background(), engine.Request{OutputDir: root, Subtitles: engine.SubtitleOptions{Embed: true}}, completeFileTestReservation(root), runner, nil, engine.NewPublicationArbiter())
	if err == nil || calls != 1 {
		t.Fatalf("err=%v calls=%d; want one terminal transfer failure", err, calls)
	}
	for _, name := range []string{"Video.mp4", "Video.mkv", "Video.srt"} {
		if _, statErr := os.Stat(filepath.Join(root, name)); !errors.Is(statErr, os.ErrNotExist) {
			t.Fatalf("unexpected public artifact %s: %v", name, statErr)
		}
	}

	// A fresh retry starts from an empty private workspace and can publish the
	// same reservation without inheriting junk from the failed transfer.
	var retryCalls []engine.Request
	result, retryDelivery, retryErr := runCompleteFile(
		context.Background(),
		engine.Request{OutputDir: root, Subtitles: engine.SubtitleOptions{Embed: true}},
		completeFileTestReservation(root), completeFileFixtureRunner(t, 0, &retryCalls), nil, engine.NewPublicationArbiter(),
	)
	if retryErr != nil || result.Filename != filepath.Join(root, "Video.mp4") || len(retryCalls) != 1 || retryDelivery.Publication == nil {
		t.Fatalf("clean retry result=%#v calls=%d delivery=%#v err=%v", result, len(retryCalls), retryDelivery, retryErr)
	}
	retryDelivery.Publication.FinishPublication()
}
