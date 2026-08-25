package admission

import (
	"fmt"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/tejasa97/vidstow/internal/jobmodel"
	"github.com/tejasa97/vidstow/internal/jobs"
	"github.com/tejasa97/vidstow/internal/outputplan"
	"github.com/tejasa97/vidstow/internal/reservation"
	"github.com/tejasa97/ytdlp-go/engine"
	"github.com/tejasa97/ytdlp-go/engine/value"
)

const (
	completeFallbackArtifactKind    engine.ArtifactKind = "complete-fallback"
	completeSubtitleArtifactKind    engine.ArtifactKind = "subtitle"
	completeMKVIdentity                                 = "mkv"
	completeDefaultSubtitleIdentity                     = "default"
)

// preparePlanArtifacts makes the complete-file publication set explicit
// before admission. MP4 is the primary claim, MKV is the container fallback,
// and bounded SRT claims cover either the optional retained copy or the final
// subtitle-sidecar fallback. The selector reserves the whole set under one
// suffix without changing reservationfs.
func preparePlanArtifacts(plan outputplan.Plan, options jobmodel.OutputOptions, metadata value.Info) (outputplan.Plan, []engine.ArtifactDeclaration, error) {
	if plan.Kind != outputplan.KindVideo || (!options.EmbedThumbnail && !options.EmbedChapters && options.SubtitleMode != jobmodel.SubtitleModeEmbed) {
		artifacts, err := engine.RenderOutputArtifacts(engine.OutputPreviewRequest{
			Template: jobs.OutputTemplateForPlan(plan), Metadata: metadata,
			Extension: strings.ToLower(strings.TrimPrefix(plan.Container, ".")),
		})
		return plan, artifacts, err
	}

	effective := plan
	effective.Container = "MP4"
	effective.RequiresFFmpeg = true
	primary, err := engine.RenderOutputArtifacts(engine.OutputPreviewRequest{
		Template: jobs.OutputTemplateForPlan(effective), Metadata: metadata, Extension: "mp4",
	})
	if err != nil || len(primary) != 1 {
		if err == nil {
			err = fmt.Errorf("engine returned %d primary artifacts", len(primary))
		}
		return outputplan.Plan{}, nil, err
	}
	fallback, err := engine.RenderOutputArtifacts(engine.OutputPreviewRequest{
		Template: jobs.OutputTemplateForPlan(effective), Metadata: metadata, Extension: "mkv",
	})
	if err != nil || len(fallback) != 1 {
		if err == nil {
			err = fmt.Errorf("engine returned %d MKV fallback artifacts", len(fallback))
		}
		return outputplan.Plan{}, nil, err
	}
	fallback[0].Kind = completeFallbackArtifactKind
	fallback[0].Identity = completeMKVIdentity

	artifacts := make([]engine.ArtifactDeclaration, 0, 2+len(options.SubtitleLanguages))
	artifacts = append(artifacts, primary[0], fallback[0])
	if options.SubtitleMode == jobmodel.SubtitleModeEmbed {
		identities := completeSubtitleIdentities(options.SubtitleLanguages)
		for _, identity := range identities {
			suffix := ""
			if len(identities) > 1 {
				suffix = strings.ToLower(identity)
			}
			basename, nameErr := completeSidecarBasename(primary[0].ProposedBasename, suffix)
			if nameErr != nil {
				return outputplan.Plan{}, nil, nameErr
			}
			artifacts = append(artifacts, engine.ArtifactDeclaration{
				Kind: completeSubtitleArtifactKind, Identity: identity, ProposedBasename: basename,
			})
		}
	}
	return effective, artifacts, nil
}

func completeSubtitleIdentities(languages []string) []string {
	if len(languages) == 0 {
		return []string{completeDefaultSubtitleIdentity}
	}
	result := make([]string, 0, len(languages))
	seen := make(map[string]bool, len(languages))
	for _, language := range languages {
		key := strings.ToLower(language)
		if seen[key] {
			continue
		}
		seen[key] = true
		result = append(result, language)
	}
	if len(result) == 0 {
		return []string{completeDefaultSubtitleIdentity}
	}
	return result
}

func completeSidecarBasename(primary, language string) (string, error) {
	extension := filepath.Ext(primary)
	stem := strings.TrimSuffix(primary, extension)
	tail := ".srt"
	if language != "" {
		tail = "." + language + tail
	}
	room := reservation.MaxBasenameBytes - len(tail)
	if room <= 0 {
		return "", fmt.Errorf("admission: subtitle basename suffix is too long")
	}
	stem = truncateCompleteBasename(stem, room)
	basename := stem + tail
	if err := reservation.ValidateBasename(basename); err != nil {
		return "", fmt.Errorf("admission: subtitle basename: %w", err)
	}
	return basename, nil
}

func truncateCompleteBasename(value string, maxBytes int) string {
	if len(value) <= maxBytes {
		return value
	}
	for maxBytes > 0 && !utf8.ValidString(value[:maxBytes]) {
		maxBytes--
	}
	return value[:maxBytes]
}
