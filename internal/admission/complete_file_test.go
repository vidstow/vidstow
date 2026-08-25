package admission

import (
	"strings"
	"testing"

	"github.com/tejasa97/vidstow/internal/jobmodel"
	"github.com/tejasa97/vidstow/internal/outputplan"
	"github.com/tejasa97/ytdlp-go/engine"
	"github.com/tejasa97/ytdlp-go/engine/value"
)

func TestPreparePlanArtifactsReservesCompleteFileFallbackSet(t *testing.T) {
	plan := outputplan.Plan{
		ID: "video-1080-webm", Kind: outputplan.KindVideo, Label: "1080p",
		Container: "WEBM", Selector: "303+251",
	}
	metadata := value.NewInfo(value.NewObject(
		value.Field{Key: "title", Value: value.String("Demo")},
		value.Field{Key: "id", Value: value.String("abc123")},
	))
	options := jobmodel.DefaultOutputOptions()
	options.SubtitleMode = jobmodel.SubtitleModeEmbed
	options.SubtitleLanguages = []string{"pt-BR", "en", "en"}

	effective, artifacts, err := preparePlanArtifacts(plan, options, metadata)
	if err != nil {
		t.Fatal(err)
	}
	if effective.Container != "MP4" || !effective.RequiresFFmpeg {
		t.Fatalf("effective plan = %#v; want MP4 requiring FFmpeg", effective)
	}
	if len(artifacts) != 4 {
		t.Fatalf("artifacts = %#v; want MP4, MKV, and two SRT claims", artifacts)
	}
	if artifacts[0].Kind != engine.ArtifactKindPrimary || artifacts[0].Identity != "primary" || !strings.HasSuffix(artifacts[0].ProposedBasename, ".mp4") {
		t.Fatalf("primary artifact = %#v", artifacts[0])
	}
	if artifacts[1].Kind != completeFallbackArtifactKind || artifacts[1].Identity != completeMKVIdentity || !strings.HasSuffix(artifacts[1].ProposedBasename, ".mkv") {
		t.Fatalf("fallback artifact = %#v", artifacts[1])
	}
	for _, artifact := range artifacts[2:] {
		if artifact.Kind != completeSubtitleArtifactKind || !strings.HasSuffix(artifact.ProposedBasename, ".srt") {
			t.Fatalf("subtitle artifact = %#v", artifact)
		}
	}
}

func TestPreparePlanArtifactsLeavesAudioPlanAlone(t *testing.T) {
	plan := outputplan.Plan{ID: "audio-m4a-original", Kind: outputplan.KindAudio, Label: "M4A", Container: "M4A", Selector: "140"}
	metadata := value.NewInfo(value.NewObject(
		value.Field{Key: "title", Value: value.String("Demo")},
		value.Field{Key: "id", Value: value.String("abc123")},
	))
	effective, artifacts, err := preparePlanArtifacts(plan, jobmodel.OutputOptions{}, metadata)
	if err != nil {
		t.Fatal(err)
	}
	if effective.Container != "M4A" || len(artifacts) != 1 || artifacts[0].Kind != engine.ArtifactKindPrimary {
		t.Fatalf("effective=%#v artifacts=%#v", effective, artifacts)
	}
}
