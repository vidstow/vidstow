package jobmodel

import "testing"

func TestOutputOptionsValidateAcceptsContractShapes(t *testing.T) {
	valid := []OutputOptions{
		{},
		{SubtitleMode: SubtitleModeSidecar, SubtitleFormat: "srt", SubtitleLanguages: []string{"en"}},
		{SubtitleMode: SubtitleModeEmbed, SubtitleSidecar: true, SubtitleFormat: "srt", SubtitleAutoCaptions: true, SubtitleLanguages: []string{"en", "pt-BR"}},
		{SubtitleMode: SubtitleModeSidecar, SubtitleFormat: "vtt"},
		{EmbedMetadata: true, EmbedChapters: true, EmbedThumbnail: true},
	}
	for _, options := range valid {
		if err := options.Validate(); err != nil {
			t.Fatalf("Validate(%#v) = %v; want nil", options, err)
		}
	}
}

func TestOutputOptionsValidateRejectsOutOfContractShapes(t *testing.T) {
	invalid := []OutputOptions{
		{SubtitleMode: "banana"},
		{SubtitleMode: SubtitleModeSidecar, SubtitleFormat: "ass"},
		{SubtitleMode: SubtitleModeEmbed, SubtitleLanguages: []string{"en", ""}},
		{SubtitleMode: SubtitleModeEmbed, SubtitleLanguages: []string{"en.*"}},
		{SubtitleMode: SubtitleModeEmbed, SubtitleLanguages: []string{"all,-en"}},
		{SubtitleMode: SubtitleModeEmbed, SubtitleSidecar: true, SubtitleFormat: "vtt"},
		{SubtitleLanguages: make([]string, maxSubtitleLanguages+1)},
	}
	for _, options := range invalid {
		if err := options.Validate(); err == nil {
			t.Fatalf("Validate(%#v) = nil; want error", options)
		}
	}
}

func TestOutputOptionsRequiresFFmpeg(t *testing.T) {
	cases := []struct {
		options OutputOptions
		want    bool
	}{
		{OutputOptions{}, false},
		{OutputOptions{SubtitleMode: SubtitleModeSidecar}, false},
		{OutputOptions{SubtitleMode: SubtitleModeSidecar, SubtitleFormat: "srt"}, true},
		{OutputOptions{SubtitleSidecar: true, SubtitleFormat: "srt"}, false},
		{OutputOptions{SubtitleMode: SubtitleModeEmbed}, true},
		{OutputOptions{SubtitleMode: SubtitleModeEmbed, SubtitleSidecar: true, SubtitleFormat: "srt"}, true},
		{OutputOptions{EmbedMetadata: true}, true},
		{OutputOptions{EmbedThumbnail: true}, true},
		{OutputOptions{EmbedChapters: true}, true},
	}
	for _, tc := range cases {
		if got := tc.options.RequiresFFmpeg(); got != tc.want {
			t.Fatalf("RequiresFFmpeg(%#v) = %v; want %v", tc.options, got, tc.want)
		}
	}
}

func TestOutputOptionsEqualAndClone(t *testing.T) {
	base := OutputOptions{SubtitleMode: SubtitleModeEmbed, SubtitleSidecar: true, SubtitleLanguages: []string{"en", "de"}}
	if !base.Equal(base.Clone()) {
		t.Fatal("clone must compare equal")
	}
	if base.Clone().IsZero() {
		t.Fatal("non-zero options must not report zero")
	}
	drifted := base
	drifted.SubtitleLanguages = []string{"de", "en"}
	if base.Equal(drifted) {
		t.Fatal("language order must participate in equality")
	}
	drifted = base
	drifted.SubtitleSidecar = false
	if base.Equal(drifted) {
		t.Fatal("sidecar choice must participate in equality")
	}
	if !(OutputOptions{}).IsZero() {
		t.Fatal("zero value must report zero")
	}
}

func TestOutputOptionsNote(t *testing.T) {
	cases := []struct {
		options OutputOptions
		want    string
	}{
		{OutputOptions{}, ""},
		{OutputOptions{SubtitleMode: SubtitleModeSidecar, SubtitleLanguages: []string{"en"}}, "subtitles (en)"},
		{OutputOptions{SubtitleMode: SubtitleModeEmbed}, "embedded subtitles"},
		{OutputOptions{SubtitleMode: SubtitleModeEmbed, SubtitleSidecar: true, SubtitleLanguages: []string{"en"}}, "embedded subtitles (en) · SRT sidecar (en)"},
		{OutputOptions{SubtitleSidecar: true, SubtitleFormat: "srt", SubtitleLanguages: []string{"en"}}, ""},
		{OutputOptions{EmbedMetadata: true, EmbedThumbnail: true, EmbedChapters: true}, "embedded metadata, thumbnail, chapters"},
		{OutputOptions{SubtitleMode: SubtitleModeSidecar, SubtitleLanguages: []string{"en"}, EmbedMetadata: true}, "subtitles (en) · embedded metadata"},
	}
	for _, tc := range cases {
		if got := tc.options.Note(); got != tc.want {
			t.Fatalf("Note(%#v) = %q; want %q", tc.options, got, tc.want)
		}
	}
}

func TestDefaultOutputOptionsAndCompleteVideoNormalization(t *testing.T) {
	defaults := DefaultOutputOptions()
	wantDefaults := OutputOptions{
		SubtitleAutoCaptions: true, EmbedThumbnail: true, EmbedChapters: true,
	}
	if !defaults.Equal(wantDefaults) || defaults.SubtitleSidecar {
		t.Fatalf("DefaultOutputOptions() = %#v; want %#v", defaults, wantDefaults)
	}

	legacy := OutputOptions{
		SubtitleMode: SubtitleModeSidecar, SubtitleFormat: "vtt",
		SubtitleLanguages: []string{"de"}, SubtitleAutoCaptions: false,
	}
	got := legacy.ForCompleteVideo()
	if got.SubtitleMode != SubtitleModeEmbed || !got.SubtitleSidecar || got.SubtitleFormat != "srt" ||
		!got.EmbedThumbnail || !got.EmbedChapters || !got.SubtitleAutoCaptions || len(got.SubtitleLanguages) != 1 || got.SubtitleLanguages[0] != "de" {
		t.Fatalf("legacy.ForCompleteVideo() = %#v", got)
	}
	if legacy.SubtitleMode != SubtitleModeSidecar || legacy.SubtitleFormat != "vtt" {
		t.Fatalf("normalization mutated legacy options: %#v", legacy)
	}

	explicit := OutputOptions{SubtitleSidecar: true, SubtitleFormat: "vtt", SubtitleLanguages: []string{"fr", "de"}}
	got = explicit.ForCompleteVideo()
	if got.SubtitleMode != "" || !got.SubtitleSidecar || got.SubtitleFormat != "" || got.SubtitleAutoCaptions || len(got.SubtitleLanguages) != 1 || got.SubtitleLanguages[0] != "fr" {
		t.Fatalf("explicit off.ForCompleteVideo() = %#v", got)
	}
	if err := explicit.Validate(); err != nil {
		t.Fatalf("dormant sidecar Validate() = %v", err)
	}

	embedded := OutputOptions{SubtitleMode: SubtitleModeEmbed, SubtitleLanguages: []string{"es", "en"}}
	got = embedded.ForCompleteVideo()
	if !got.SubtitleAutoCaptions || len(got.SubtitleLanguages) != 1 || got.SubtitleLanguages[0] != "es" {
		t.Fatalf("embedded.ForCompleteVideo() = %#v", got)
	}

	got = (OutputOptions{}).ForCompleteVideo()
	if !got.Equal(defaults) {
		t.Fatalf("zero.ForCompleteVideo() = %#v; want defaults %#v", got, defaults)
	}

	withoutSidecar := OutputOptions{SubtitleMode: SubtitleModeEmbed, SubtitleFormat: "srt", SubtitleLanguages: []string{"en"}}
	got = withoutSidecar.ForCompleteVideo()
	if got.SubtitleFormat != "" || got.SubtitleSidecar {
		t.Fatalf("sidecar-off normalization retained a subtitle format: %#v", got)
	}
}
