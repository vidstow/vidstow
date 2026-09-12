package main

import (
	"strings"

	"github.com/tejasa97/vidstow/internal/jobmodel"
	"github.com/tejasa97/vidstow/internal/jobs"
)

// collectionChildOptions copies the card's extras onto one playlist or batch
// child, then picks a single subtitle language for that video. Named languages
// are a preference, not a union: Spanish on the card does not also write
// English just because Settings named English as the default.
func collectionChildOptions(picked jobmodel.OutputOptions, settingsDefault []string, offered []jobs.SubtitleLanguage) jobmodel.OutputOptions {
	out := picked.Clone()
	if out.SubtitleMode == "" {
		return out
	}
	out.SubtitleLanguages = resolveCollectionSubtitleLanguages(picked.SubtitleLanguages, settingsDefault, offered, out.SubtitleAutoCaptions)
	return out
}

// resolveCollectionSubtitleLanguages prefers the language on the card when
// that video has it. Missing Spanish (or any named pick) uses the Settings
// default when that is also a named language this video has. Empty on the card
// means English or first available for every title. Empty after both named
// picks miss lets the engine choose one track rather than leaving captions off.
func resolveCollectionSubtitleLanguages(picked, fallback []string, offered []jobs.SubtitleLanguage, allowAuto bool) []string {
	preferred := firstLanguageCode(picked)
	if preferred == "" {
		return nil
	}
	if code := matchingOfferedLanguage(offered, preferred, allowAuto); code != "" {
		return []string{code}
	}
	fallbackCode := firstLanguageCode(fallback)
	if fallbackCode != "" && !languageCodeEqual(fallbackCode, preferred) {
		if code := matchingOfferedLanguage(offered, fallbackCode, allowAuto); code != "" {
			return []string{code}
		}
	}
	return nil
}

func firstLanguageCode(codes []string) string {
	for _, code := range codes {
		if trimmed := strings.TrimSpace(code); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func matchingOfferedLanguage(offered []jobs.SubtitleLanguage, code string, allowAuto bool) string {
	want := strings.ToLower(strings.TrimSpace(code))
	if want == "" {
		return ""
	}
	var prefix string
	for _, language := range offered {
		if language.Auto && !allowAuto {
			continue
		}
		have := strings.ToLower(strings.TrimSpace(language.Code))
		if have == want {
			return language.Code
		}
		if prefix == "" && languageCodeMatches(have, want) {
			prefix = language.Code
		}
	}
	return prefix
}

func languageCodeMatches(have, want string) bool {
	return have == want || strings.HasPrefix(have, want+"-") || strings.HasPrefix(have, want+"_")
}

func languageCodeEqual(left, right string) bool {
	return strings.EqualFold(strings.TrimSpace(left), strings.TrimSpace(right))
}
