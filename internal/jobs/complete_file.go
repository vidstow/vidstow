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
	DegradedEmbedding bool
	DegradedToSidecar bool
	Publication       *engine.PublicationReservation
}

func usesCompleteFileStaging(state *jobState) bool {
	return state != nil && state.fromStateV2 && state.plan != nil &&
		state.plan.Kind == "video" && (state.options.SubtitleMode == jobmodel.SubtitleModeEmbed ||
		state.options.EmbedThumbnail || state.options.EmbedChapters)
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
	arbiter *engine.PublicationArbiter,
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
		published, sidecar, publication, publishErr := publishCompleteFile(ctx, arbiter, result, attemptRoot, reservation)
		if publishErr != nil {
			return engine.Result{}, completeFileDelivery{}, publishErr
		}
		return published, completeFileDelivery{Container: "MP4", SubtitleSidecar: sidecar, Publication: publication}, nil
	}
	_ = os.RemoveAll(attemptRoot)
	if !completeFileFallbackEligible(ctx, err, sawPostprocess, false) {
		return engine.Result{}, completeFileDelivery{}, err
	}

	result, attemptRoot, sawPostprocess, err = attempt("mkv", true)
	if err == nil {
		published, sidecar, publication, publishErr := publishCompleteFile(ctx, arbiter, result, attemptRoot, reservation)
		if publishErr != nil {
			return engine.Result{}, completeFileDelivery{}, publishErr
		}
		return published, completeFileDelivery{Container: "MKV", UsedMKVFallback: true, SubtitleSidecar: sidecar, Publication: publication}, nil
	}
	_ = os.RemoveAll(attemptRoot)
	if !base.Subtitles.Embed || !completeFileFallbackEligible(ctx, err, sawPostprocess, true) {
		return engine.Result{}, completeFileDelivery{}, err
	}

	// Final subtitle degradation keeps the requested non-subtitle embedding
	// and emits an SRT when captions exist. Publication remains an
	// all-or-nothing reserved set.
	result, attemptRoot, _, err = attempt("mkv", false)
	if err != nil {
		_ = os.RemoveAll(attemptRoot)
		return engine.Result{}, completeFileDelivery{}, err
	}
	published, sidecar, publication, publishErr := publishCompleteFile(ctx, arbiter, result, attemptRoot, reservation)
	if publishErr != nil {
		return engine.Result{}, completeFileDelivery{}, publishErr
	}
	return published, completeFileDelivery{
		Container: "MKV", UsedMKVFallback: true, SubtitleSidecar: sidecar,
		DegradedEmbedding: true, DegradedToSidecar: sidecar, Publication: publication,
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
	return request, nil
}

func completeFileFallbackEligible(ctx context.Context, err error, sawPostprocess, subtitleDegradation bool) bool {
	if err == nil || ctx.Err() != nil || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) || !sawPostprocess {
		return false
	}
	// A filesystem/resource failure cannot become compatible by downloading the
	// same media again in another container. Fail once and preserve the original
	// error instead of multiplying disk or permission failures.
	var pathErr *os.PathError
	if errors.As(err, &pathErr) || errors.Is(err, os.ErrPermission) {
		return false
	}
	foundTyped, compatible := completeFileTypedFailureCompatible(err, subtitleDegradation, 0)
	return foundTyped && compatible
}

func completeFileTypedFailureCompatible(err error, subtitleDegradation bool, depth int) (bool, bool) {
	if err == nil {
		return false, true
	}
	if depth > 16 {
		return false, false
	}
	foundTyped := false
	if engineErr, ok := err.(*engine.Error); ok {
		foundTyped = true
		switch engineErr.Category {
		case engine.ErrorUnsupported, engine.ErrorInvalidInput, engine.ErrorInternal:
		default:
			return true, false
		}
		operation := strings.ToLower(strings.TrimSpace(engineErr.Op))
		compatible := false
		switch operation {
		case "embed subtitles":
			compatible = true
		case "run postprocessors":
			// The pinned engine reports FFmpeg/tool absence as Unsupported at
			// this operation. Only an actual media-processing failure (Internal)
			// is evidence that another container may help.
			compatible = !subtitleDegradation && engineErr.Category == engine.ErrorInternal
		case "embed metadata", "embed thumbnail":
			compatible = !subtitleDegradation && engineErr.Category != engine.ErrorInvalidInput
		}
		if !compatible {
			return true, false
		}
	}
	if joined, ok := err.(interface{ Unwrap() []error }); ok {
		for _, cause := range joined.Unwrap() {
			childFound, childCompatible := completeFileTypedFailureCompatible(cause, subtitleDegradation, depth+1)
			foundTyped = foundTyped || childFound
			if !childCompatible {
				return foundTyped, false
			}
		}
		return foundTyped, true
	}
	if cause := errors.Unwrap(err); cause != nil {
		childFound, childCompatible := completeFileTypedFailureCompatible(cause, subtitleDegradation, depth+1)
		return foundTyped || childFound, childCompatible
	}
	return foundTyped, true
}

func publishCompleteFile(ctx context.Context, arbiter *engine.PublicationArbiter, result engine.Result, attemptRoot string, reservation jobmodel.ReservationSet) (engine.Result, bool, *engine.PublicationReservation, error) {
	if arbiter == nil {
		return engine.Result{}, false, nil, errors.New("complete-file publication arbiter is unavailable")
	}
	mediaSource, err := completeStagedRegularFile(attemptRoot, result.Filename)
	if err != nil {
		return engine.Result{}, false, nil, err
	}
	mediaInfo, err := os.Stat(mediaSource)
	if err != nil {
		return engine.Result{}, false, nil, fmt.Errorf("inspect staged complete file: %w", err)
	}
	container := strings.TrimPrefix(strings.ToLower(filepath.Ext(mediaSource)), ".")
	var mediaReservation jobmodel.ReservedArtifact
	switch container {
	case "mp4":
		var ok bool
		mediaReservation, ok = completeReservedArtifact(reservation, string(engine.ArtifactKindPrimary), "primary")
		if !ok {
			return engine.Result{}, false, nil, errors.New("complete-file MP4 reservation is unavailable")
		}
	case "mkv":
		var ok bool
		mediaReservation, ok = completeReservedArtifact(reservation, completeFallbackArtifactKind, completeMKVIdentity)
		if !ok {
			return engine.Result{}, false, nil, errors.New("complete-file MKV reservation is unavailable")
		}
	default:
		return engine.Result{}, false, nil, fmt.Errorf("complete-file produced unsupported container %q", container)
	}

	subtitleSources := make([]string, 0)
	seenSources := make(map[string]bool)
	for _, artifact := range result.Artifacts {
		if artifact.Kind != completeSubtitleArtifactKind || !strings.EqualFold(filepath.Ext(artifact.Path), ".srt") {
			continue
		}
		path, pathErr := completeStagedRegularFile(attemptRoot, artifact.Path)
		if pathErr != nil {
			return engine.Result{}, false, nil, pathErr
		}
		if !seenSources[path] {
			seenSources[path] = true
			subtitleSources = append(subtitleSources, path)
		}
	}
	sort.Strings(subtitleSources)
	subtitleReservations := completeReservedArtifacts(reservation, completeSubtitleArtifactKind)
	if len(subtitleSources) > len(subtitleReservations) {
		return engine.Result{}, false, nil, errors.New("complete-file produced more subtitle files than were reserved")
	}

	type publicationItem struct{ source, destination string }
	items := make([]publicationItem, 0, len(subtitleSources)+1)
	available := append([]jobmodel.ReservedArtifact(nil), subtitleReservations...)
	for _, source := range subtitleSources {
		index := matchingSubtitleReservation(source, available)
		if index < 0 {
			return engine.Result{}, false, nil, errors.New("complete-file has no matching subtitle reservation")
		}
		items = append(items, publicationItem{
			source: source, destination: filepath.Join(reservation.Directory.CanonicalPath, available[index].Basename),
		})
		available = append(available[:index], available[index+1:]...)
	}
	mediaDestination := filepath.Join(reservation.Directory.CanonicalPath, mediaReservation.Basename)
	items = append(items, publicationItem{source: mediaSource, destination: mediaDestination})

	publication, err := arbiter.BeginPublication(ctx)
	if err != nil {
		return engine.Result{}, false, nil, fmt.Errorf("begin complete-file publication: %w", err)
	}
	terminalized := false
	defer func() {
		if !terminalized {
			publication.MarkIndeterminate()
		}
	}()
	abort := func(cause error) (engine.Result, bool, *engine.PublicationReservation, error) {
		publication.AbortBeforeReplace()
		terminalized = true
		return engine.Result{}, false, nil, cause
	}
	if err := validateCompletePublicationRoot(reservation.Directory); err != nil {
		return abort(err)
	}
	for _, item := range items {
		if _, statErr := os.Lstat(item.destination); statErr == nil {
			return abort(fmt.Errorf("reserved output appeared before publication: %s", filepath.Base(item.destination)))
		} else if !errors.Is(statErr, os.ErrNotExist) {
			return abort(fmt.Errorf("check reserved output: %w", statErr))
		}
	}

	linked := make([]string, 0, len(items))
	rollback := func(cause error) (error, bool) {
		var cleanupErr error
		for index := len(linked) - 1; index >= 0; index-- {
			if removeErr := os.Remove(linked[index]); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
				cleanupErr = errors.Join(cleanupErr, removeErr)
			}
		}
		return errors.Join(cause, cleanupErr), cleanupErr == nil
	}
	for _, item := range items {
		// The workspace is below the output root, so a hard link is a
		// same-volume, no-overwrite publication primitive. Media is linked last.
		if linkErr := os.Link(item.source, item.destination); linkErr != nil {
			publicationErr, clean := rollback(fmt.Errorf("publish complete file: %w", linkErr))
			if clean {
				publication.AbortBeforeReplace()
			} else {
				publication.MarkIndeterminate()
			}
			terminalized = true
			return engine.Result{}, false, nil, publicationErr
		}
		linked = append(linked, item.destination)
	}

	publishedArtifacts := make([]engine.Artifact, 0, len(items))
	for index, item := range items {
		kind := completeSubtitleArtifactKind
		if index == len(items)-1 {
			kind = "media"
		}
		publishedArtifacts = append(publishedArtifacts, engine.Artifact{Path: item.destination, Kind: kind})
	}
	result.Filename = mediaDestination
	result.Artifacts = publishedArtifacts
	result.Bytes = mediaInfo.Size()
	publication.MarkDestinationReplaced()
	terminalized = true
	return result, len(subtitleSources) > 0, publication, nil
}

func validateCompletePublicationRoot(root jobmodel.OutputRootRef) error {
	validated, err := engine.ValidateOutputRoot(root.CanonicalPath)
	if err != nil {
		return fmt.Errorf("validate complete-file output root: %w", err)
	}
	if validated.CanonicalPath != root.CanonicalPath || root.EngineIdentity != "" && validated.Identity != root.EngineIdentity {
		return errors.New("complete-file output root identity changed before publication")
	}
	return nil
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
