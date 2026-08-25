package jobs

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/tejasa97/vidstow/internal/jobmodel"
	"github.com/tejasa97/ytdlp-go/engine"
)

const (
	completeFallbackArtifactKind = "complete-fallback"
	completeSubtitleArtifactKind = "subtitle"
	completeMKVIdentity          = "mkv"
)

type completeFileDelivery struct {
	Container         string
	UsedMKVFallback   bool
	SubtitleSidecar   bool
	DegradedToSidecar bool
}

func usesCompleteFileStaging(state *jobState) bool {
	return state != nil && state.fromStateV2 && state.plan != nil &&
		state.plan.Kind == "video" && state.options.SubtitleMode == jobmodel.SubtitleModeEmbed
}

// runCompleteFile keeps every processing attempt in a private same-volume
// workspace. Only a fully processed artifact set is hard-linked into the
// reserved public names, so FFmpeg failure, cancellation, and retries cannot
// expose a partial media file.
func runCompleteFile(
	ctx context.Context,
	base engine.Request,
	reservation jobmodel.ReservationSet,
	runner downloadRunner,
	handler engine.EventHandler,
) (engine.Result, completeFileDelivery, error) {
	workspace, err := os.MkdirTemp(base.OutputDir, ".vidstow-complete-")
	if err != nil {
		return engine.Result{}, completeFileDelivery{}, fmt.Errorf("create complete-file workspace: %w", err)
	}
	defer os.RemoveAll(workspace)

	attempt := func(container string, embed bool) (engine.Result, string, bool, error) {
		attemptRoot, mkdirErr := os.MkdirTemp(workspace, container+"-")
		if mkdirErr != nil {
			return engine.Result{}, "", false, fmt.Errorf("create %s attempt workspace: %w", container, mkdirErr)
		}
		request, requestErr := completeFileAttemptRequest(base, reservation, attemptRoot, container, embed)
		if requestErr != nil {
			return engine.Result{}, attemptRoot, false, requestErr
		}
		sawPostprocess := false
		attemptHandler := func(eventCtx context.Context, event engine.Event) error {
			if event.Kind == engine.EventPostprocessStarting {
				sawPostprocess = true
			}
			if handler == nil {
				return nil
			}
			return handler(eventCtx, event)
		}
		result, runErr := runner(ctx, request, attemptHandler)
		if runErr == nil {
			extension := strings.TrimPrefix(strings.ToLower(filepath.Ext(result.Filename)), ".")
			if extension != container {
				runErr = fmt.Errorf("complete-file %s attempt produced %q", container, extension)
			}
		}
		return result, attemptRoot, sawPostprocess, runErr
	}

	result, attemptRoot, sawPostprocess, err := attempt("mp4", true)
	if err == nil {
		published, sidecar, publishErr := publishCompleteFile(result, attemptRoot, reservation)
		if publishErr != nil {
			return engine.Result{}, completeFileDelivery{}, publishErr
		}
		return published, completeFileDelivery{Container: "MP4", SubtitleSidecar: sidecar}, nil
	}
	_ = os.RemoveAll(attemptRoot)
	if !completeFileFallbackEligible(ctx, err, sawPostprocess) {
		return engine.Result{}, completeFileDelivery{}, err
	}

	result, attemptRoot, sawPostprocess, err = attempt("mkv", true)
	if err == nil {
		published, sidecar, publishErr := publishCompleteFile(result, attemptRoot, reservation)
		if publishErr != nil {
			return engine.Result{}, completeFileDelivery{}, publishErr
		}
		return published, completeFileDelivery{Container: "MKV", UsedMKVFallback: true, SubtitleSidecar: sidecar}, nil
	}
	_ = os.RemoveAll(attemptRoot)
	if !completeFileFallbackEligible(ctx, err, sawPostprocess) {
		return engine.Result{}, completeFileDelivery{}, err
	}

	// Final degradation keeps a playable MKV and an SRT when captions exist.
	// All embedding is disabled because both complete-container attempts have
	// failed; publication remains an all-or-nothing reserved set.
	result, attemptRoot, _, err = attempt("mkv", false)
	if err != nil {
		_ = os.RemoveAll(attemptRoot)
		return engine.Result{}, completeFileDelivery{}, err
	}
	published, sidecar, publishErr := publishCompleteFile(result, attemptRoot, reservation)
	if publishErr != nil {
		return engine.Result{}, completeFileDelivery{}, publishErr
	}
	return published, completeFileDelivery{
		Container: "MKV", UsedMKVFallback: true, SubtitleSidecar: sidecar, DegradedToSidecar: sidecar,
	}, nil
}

func completeFileAttemptRequest(base engine.Request, reservation jobmodel.ReservationSet, root, container string, embed bool) (engine.Request, error) {
	primary, ok := completeReservedArtifact(reservation, string(engine.ArtifactKindPrimary), "primary")
	if !ok {
		return engine.Request{}, errors.New("complete-file reservation has no MP4 primary")
	}
	stem := strings.TrimSuffix(primary.Basename, filepath.Ext(primary.Basename))
	if stem == "" {
		return engine.Request{}, errors.New("complete-file reservation has an invalid primary name")
	}

	request := base
	request.OutputDir = root
	request.OutputTemplate = stem + ".%(ext)s"
	request.OutputTemplates = nil
	request.OutputPaths = engine.OutputPaths{}
	request.Overwrite = true
	request.MergeOutputFormat = container
	request.Postprocessors = append([]engine.Postprocessor(nil), base.Postprocessors...)
	request.Postprocessors = append(request.Postprocessors, engine.Postprocessor{
		Remux: &engine.RemuxPostprocessor{Format: container},
	})
	request.Filesystem.Resume = engine.ResumeOptions{}
	request.Filesystem.PreservePartialOnCancel = false
	if embed {
		return request, nil
	}

	request.Subtitles.Embed = false
	request.Subtitles.KeepFiles = false
	request.Subtitles.ConvertFormat = "srt"
	request.EmbedMetadata = false
	request.EmbedChapters = nil
	request.Thumbnails = engine.ThumbnailOptions{}
	return request, nil
}

func completeFileFallbackEligible(ctx context.Context, err error, sawPostprocess bool) bool {
	if err == nil || ctx.Err() != nil || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) || !sawPostprocess {
		return false
	}
	var engineErr *engine.Error
	if !errors.As(err, &engineErr) {
		return true
	}
	switch engineErr.Category {
	case engine.ErrorUnsupported, engine.ErrorInvalidInput, engine.ErrorInternal:
		return true
	default:
		return false
	}
}

func publishCompleteFile(result engine.Result, attemptRoot string, reservation jobmodel.ReservationSet) (engine.Result, bool, error) {
	mediaSource, err := completeStagedRegularFile(attemptRoot, result.Filename)
	if err != nil {
		return engine.Result{}, false, err
	}
	container := strings.TrimPrefix(strings.ToLower(filepath.Ext(mediaSource)), ".")
	var mediaReservation jobmodel.ReservedArtifact
	switch container {
	case "mp4":
		var ok bool
		mediaReservation, ok = completeReservedArtifact(reservation, string(engine.ArtifactKindPrimary), "primary")
		if !ok {
			return engine.Result{}, false, errors.New("complete-file MP4 reservation is unavailable")
		}
	case "mkv":
		var ok bool
		mediaReservation, ok = completeReservedArtifact(reservation, completeFallbackArtifactKind, completeMKVIdentity)
		if !ok {
			return engine.Result{}, false, errors.New("complete-file MKV reservation is unavailable")
		}
	default:
		return engine.Result{}, false, fmt.Errorf("complete-file produced unsupported container %q", container)
	}

	subtitleSources := make([]string, 0)
	seenSources := make(map[string]bool)
	for _, artifact := range result.Artifacts {
		if artifact.Kind != completeSubtitleArtifactKind || !strings.EqualFold(filepath.Ext(artifact.Path), ".srt") {
			continue
		}
		path, pathErr := completeStagedRegularFile(attemptRoot, artifact.Path)
		if pathErr != nil {
			return engine.Result{}, false, pathErr
		}
		if !seenSources[path] {
			seenSources[path] = true
			subtitleSources = append(subtitleSources, path)
		}
	}
	sort.Strings(subtitleSources)
	subtitleReservations := completeReservedArtifacts(reservation, completeSubtitleArtifactKind)
	if len(subtitleSources) > len(subtitleReservations) {
		return engine.Result{}, false, errors.New("complete-file produced more subtitle files than were reserved")
	}

	type publication struct{ source, destination string }
	publications := make([]publication, 0, len(subtitleSources)+1)
	available := append([]jobmodel.ReservedArtifact(nil), subtitleReservations...)
	for _, source := range subtitleSources {
		index := matchingSubtitleReservation(source, available)
		if index < 0 {
			return engine.Result{}, false, errors.New("complete-file has no matching subtitle reservation")
		}
		publications = append(publications, publication{
			source: source, destination: filepath.Join(reservation.Directory.CanonicalPath, available[index].Basename),
		})
		available = append(available[:index], available[index+1:]...)
	}
	mediaDestination := filepath.Join(reservation.Directory.CanonicalPath, mediaReservation.Basename)
	publications = append(publications, publication{source: mediaSource, destination: mediaDestination})

	for _, item := range publications {
		if _, statErr := os.Lstat(item.destination); statErr == nil {
			return engine.Result{}, false, fmt.Errorf("reserved output appeared before publication: %s", filepath.Base(item.destination))
		} else if !errors.Is(statErr, os.ErrNotExist) {
			return engine.Result{}, false, fmt.Errorf("check reserved output: %w", statErr)
		}
	}

	linked := make([]string, 0, len(publications))
	rollback := func(cause error) error {
		var cleanupErr error
		for index := len(linked) - 1; index >= 0; index-- {
			if removeErr := os.Remove(linked[index]); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
				cleanupErr = errors.Join(cleanupErr, removeErr)
			}
		}
		return errors.Join(cause, cleanupErr)
	}
	for _, item := range publications {
		// The workspace is below the output root, so a hard link is a
		// same-volume, no-overwrite publication primitive. Media is linked last.
		if linkErr := os.Link(item.source, item.destination); linkErr != nil {
			return engine.Result{}, false, rollback(fmt.Errorf("publish complete file: %w", linkErr))
		}
		linked = append(linked, item.destination)
	}

	publishedArtifacts := make([]engine.Artifact, 0, len(publications))
	for index, item := range publications {
		kind := completeSubtitleArtifactKind
		if index == len(publications)-1 {
			kind = "media"
		}
		publishedArtifacts = append(publishedArtifacts, engine.Artifact{Path: item.destination, Kind: kind})
	}
	result.Filename = mediaDestination
	result.Artifacts = publishedArtifacts
	if info, statErr := os.Stat(mediaDestination); statErr == nil {
		result.Bytes = info.Size()
	}
	return result, len(subtitleSources) > 0, nil
}

func completeStagedRegularFile(root, candidate string) (string, error) {
	if candidate == "" {
		return "", errors.New("complete-file attempt returned no media filename")
	}
	absoluteRoot, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	absoluteCandidate, err := filepath.Abs(candidate)
	if err != nil {
		return "", err
	}
	relative, err := filepath.Rel(absoluteRoot, absoluteCandidate)
	if err != nil || relative == "." || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", errors.New("complete-file artifact escaped its private workspace")
	}
	info, err := os.Lstat(absoluteCandidate)
	if err != nil {
		return "", fmt.Errorf("inspect complete-file artifact: %w", err)
	}
	if !info.Mode().IsRegular() {
		return "", errors.New("complete-file artifact is not a regular file")
	}
	return absoluteCandidate, nil
}

func completeReservedArtifact(set jobmodel.ReservationSet, kind, identity string) (jobmodel.ReservedArtifact, bool) {
	for _, artifact := range set.Artifacts {
		if artifact.Kind == kind && artifact.Identity == identity {
			return artifact, true
		}
	}
	return jobmodel.ReservedArtifact{}, false
}

func completeReservedArtifacts(set jobmodel.ReservationSet, kind string) []jobmodel.ReservedArtifact {
	result := make([]jobmodel.ReservedArtifact, 0)
	for _, artifact := range set.Artifacts {
		if artifact.Kind == kind {
			result = append(result, artifact)
		}
	}
	return result
}

func matchingSubtitleReservation(source string, candidates []jobmodel.ReservedArtifact) int {
	basename := strings.TrimSuffix(filepath.Base(source), filepath.Ext(source))
	language := strings.TrimPrefix(filepath.Ext(basename), ".")
	for index, candidate := range candidates {
		if strings.EqualFold(candidate.Identity, language) {
			return index
		}
	}
	for index, candidate := range candidates {
		if candidate.Identity == "default" {
			return index
		}
	}
	if len(candidates) > 0 {
		return 0
	}
	return -1
}
