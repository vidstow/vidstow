package main

import (
	"slices"
	"testing"

	"github.com/tejasa97/vidstow/internal/jobmodel"
	"github.com/tejasa97/vidstow/internal/jobs"
)

func TestResolveCollectionSubtitleLanguages(t *testing.T) {
	es := []jobs.SubtitleLanguage{{Code: "es", Name: "Spanish"}}
	es419 := []jobs.SubtitleLanguage{{Code: "es-419", Name: "Spanish"}}
	en := []jobs.SubtitleLanguage{{Code: "en", Name: "English"}}
	de := []jobs.SubtitleLanguage{{Code: "de", Name: "German"}}
	autoEs := []jobs.SubtitleLanguage{{Code: "es", Name: "Spanish", Auto: true}}
	enAndAutoEs := []jobs.SubtitleLanguage{
		{Code: "en", Name: "English"},
		{Code: "es", Name: "Spanish", Auto: true},
	}

	tests := []struct {
		name      string
		picked    []string
		fallback  []string
		offered   []jobs.SubtitleLanguage
		allowAuto bool
		want      []string
	}{
		{name: "spanish present", picked: []string{"es"}, offered: es, allowAuto: true, want: []string{"es"}},
		{name: "spanish region sends the offered code", picked: []string{"es"}, fallback: []string{"en"}, offered: es419, allowAuto: true, want: []string{"es-419"}},
		{name: "spanish missing uses settings english", picked: []string{"es"}, fallback: []string{"en"}, offered: en, allowAuto: true, want: []string{"en"}},
		{name: "spanish missing and settings is pick for me", picked: []string{"es"}, offered: en, allowAuto: true},
		{name: "card is english or first available", fallback: []string{"en"}, offered: es, allowAuto: true},
		{name: "same named language missing uses engine pick", picked: []string{"es"}, fallback: []string{"es"}, offered: en, allowAuto: true},
		{name: "both named languages missing uses engine pick", picked: []string{"es"}, fallback: []string{"en"}, offered: de, allowAuto: true},
		{name: "empty analysis uses engine pick", picked: []string{"es"}, fallback: []string{"en"}, allowAuto: true},
		{name: "auto spanish used when auto captions are on", picked: []string{"es"}, offered: autoEs, allowAuto: true, want: []string{"es"}},
		{name: "auto spanish skipped without auto captions", picked: []string{"es"}, fallback: []string{"en"}, offered: enAndAutoEs, want: []string{"en"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := resolveCollectionSubtitleLanguages(test.picked, test.fallback, test.offered, test.allowAuto)
			if !slices.Equal(got, test.want) {
				t.Fatalf("got %v, want %v", got, test.want)
			}
		})
	}
}

func TestCollectionChildOptionsPicksOneLanguageWithoutMutatingTheCard(t *testing.T) {
	picked := jobmodel.OutputOptions{
		SubtitleMode:         jobmodel.SubtitleModeSidecar,
		SubtitleLanguages:    []string{"es"},
		SubtitleAutoCaptions: true,
	}
	got := collectionChildOptions(picked, []string{"en"}, []jobs.SubtitleLanguage{{Code: "en"}})
	if !slices.Equal(got.SubtitleLanguages, []string{"en"}) {
		t.Fatalf("child languages = %v, want [en]", got.SubtitleLanguages)
	}
	if !slices.Equal(picked.SubtitleLanguages, []string{"es"}) {
		t.Fatalf("card languages mutated: %v", picked.SubtitleLanguages)
	}

	off := jobmodel.OutputOptions{SubtitleLanguages: []string{"es"}}
	if kept := collectionChildOptions(off, []string{"en"}, []jobs.SubtitleLanguage{{Code: "en"}}); !slices.Equal(kept.SubtitleLanguages, []string{"es"}) {
		t.Fatalf("off mode languages = %v, want [es]", kept.SubtitleLanguages)
	}
}
