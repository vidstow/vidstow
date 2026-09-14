// Package follow is VidStow's durable playlist-watch store. It is not a Queue
// collection: follows remember a public playlist and its download settings so
// the user can check later, review new videos, and admit chosen ones through
// the existing playlist collection path.
package follow

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/tejasa97/vidstow/internal/jobmodel"
	"github.com/tejasa97/vidstow/internal/jobs"
)

const (
	DocumentVersion = 1
	MaxFollows      = 200
	MaxKnownIDs     = 2_000
	MaxReviewItems  = 500
	maxIDBytes      = 128
	maxTextBytes    = 4096
	maxPathBytes    = 4096
)

const (
	ScopeFuture = "future"
	ScopeAll    = "all"
)

const (
	CheckIdle     = "idle"
	CheckChecking = "checking"
	CheckWaiting  = "waiting"
)

// Item is one playlist video remembered for review or skip.
type Item struct {
	VideoID   string `json:"videoId"`
	URL       string `json:"url"`
	Title     string `json:"title"`
	Duration  string `json:"duration,omitempty"`
	Thumbnail string `json:"thumbnail,omitempty"`
	Available bool   `json:"available"`
	Index     int    `json:"index"`
}

// Output is the download policy saved with a follow. Folder is the playlist
// destination computed at follow time, not a Queue collection root.
type Output struct {
	Quality      jobs.Quality           `json:"quality"`
	AudioBitrate int                    `json:"audioBitrate,omitempty"`
	Options      jobmodel.OutputOptions `json:"options,omitempty"`
	Folder       string                 `json:"folder"`
}

// Record is one followed playlist. KnownVideoIDs are the baseline plus every
// video seen later. Pending and skipped are review lists, not queue jobs.
type Record struct {
	ID             string   `json:"id"`
	PlaylistID     string   `json:"playlistId"`
	SourceURL      string   `json:"sourceUrl"`
	Title          string   `json:"title"`
	Channel        string   `json:"channel,omitempty"`
	Thumbnail      string   `json:"thumbnail,omitempty"`
	VideoCount     int      `json:"videoCount"`
	Output         Output   `json:"output"`
	KnownVideoIDs  []string `json:"knownVideoIds"`
	Pending        []Item   `json:"pending,omitempty"`
	Skipped        []Item   `json:"skipped,omitempty"`
	LastCheckedAt  string   `json:"lastCheckedAt,omitempty"`
	LastCheckError string   `json:"lastCheckError,omitempty"`
	CreatedAt      string   `json:"createdAt"`
	UpdatedAt      string   `json:"updatedAt"`
}

// Document is the on-disk follows.json image. It is a sibling of State v2,
// not a field inside the queue document.
type Document struct {
	Version int      `json:"version"`
	Follows []Record `json:"follows"`
}

func CloneRecord(in Record) Record {
	out := in
	out.KnownVideoIDs = append([]string(nil), in.KnownVideoIDs...)
	out.Pending = append([]Item(nil), in.Pending...)
	out.Skipped = append([]Item(nil), in.Skipped...)
	out.Output.Options = in.Output.Options.Clone()
	return out
}

func CloneDocument(in Document) Document {
	out := in
	out.Follows = make([]Record, len(in.Follows))
	for i := range in.Follows {
		out.Follows[i] = CloneRecord(in.Follows[i])
	}
	return out
}

func ValidateOutput(output Output) error {
	if err := output.Options.Validate(); err != nil {
		return err
	}
	switch output.Quality {
	case jobs.QualityBest, jobs.Quality4K, jobs.Quality1440p, jobs.Quality1080p, jobs.Quality720p:
		if output.AudioBitrate != 0 {
			return fmt.Errorf("audio bitrate is only valid for audio playlists")
		}
	case jobs.QualityAudioOnly:
		switch output.AudioBitrate {
		case 0, 128, 192, 256:
		default:
			return fmt.Errorf("audio bitrate must be original, 128, 192, or 256 kbps")
		}
	default:
		return fmt.Errorf("unsupported playlist quality")
	}
	if !validText(output.Folder, maxPathBytes, true) {
		return fmt.Errorf("follow destination is missing")
	}
	return nil
}

func ValidateRecord(rec Record) error {
	if !validID(rec.ID) || !validText(rec.PlaylistID, maxIDBytes, true) || !validText(rec.SourceURL, maxTextBytes, true) ||
		!validText(rec.Title, maxTextBytes, true) || !validText(rec.Channel, maxTextBytes, false) ||
		!validText(rec.Thumbnail, maxTextBytes, false) || !validText(rec.LastCheckedAt, maxIDBytes, false) ||
		!validText(rec.LastCheckError, maxTextBytes, false) || !validText(rec.CreatedAt, maxIDBytes, true) ||
		!validText(rec.UpdatedAt, maxIDBytes, true) {
		return fmt.Errorf("follow record is incomplete")
	}
	if rec.VideoCount < 0 || rec.VideoCount > jobs.MaxPlaylistEntries {
		return fmt.Errorf("follow video count is out of range")
	}
	if err := ValidateOutput(rec.Output); err != nil {
		return err
	}
	if len(rec.KnownVideoIDs) > MaxKnownIDs || len(rec.Pending) > MaxReviewItems || len(rec.Skipped) > MaxReviewItems {
		return fmt.Errorf("follow review lists exceed limit")
	}
	seenKnown := map[string]struct{}{}
	for _, id := range rec.KnownVideoIDs {
		if !validVideoID(id) {
			return fmt.Errorf("follow known video id is invalid")
		}
		if _, dup := seenKnown[id]; dup {
			return fmt.Errorf("follow known video id is duplicated")
		}
		seenKnown[id] = struct{}{}
	}
	seenPending := map[string]struct{}{}
	for _, item := range rec.Pending {
		if err := validateItem(item); err != nil {
			return err
		}
		if _, dup := seenPending[item.VideoID]; dup {
			return fmt.Errorf("follow pending video is duplicated")
		}
		if _, known := seenKnown[item.VideoID]; !known {
			return fmt.Errorf("follow pending video is not in the known set")
		}
		seenPending[item.VideoID] = struct{}{}
	}
	seenSkipped := map[string]struct{}{}
	for _, item := range rec.Skipped {
		if err := validateItem(item); err != nil {
			return err
		}
		if _, dup := seenSkipped[item.VideoID]; dup {
			return fmt.Errorf("follow skipped video is duplicated")
		}
		if _, pending := seenPending[item.VideoID]; pending {
			return fmt.Errorf("follow video cannot be pending and skipped")
		}
		if _, known := seenKnown[item.VideoID]; !known {
			return fmt.Errorf("follow skipped video is not in the known set")
		}
		seenSkipped[item.VideoID] = struct{}{}
	}
	return nil
}

func ValidateDocument(doc Document) error {
	if doc.Version != DocumentVersion {
		return fmt.Errorf("unsupported follows version")
	}
	if len(doc.Follows) > MaxFollows {
		return fmt.Errorf("too many followed playlists")
	}
	seenID := map[string]struct{}{}
	seenPlaylist := map[string]struct{}{}
	for _, rec := range doc.Follows {
		if err := ValidateRecord(rec); err != nil {
			return err
		}
		if _, dup := seenID[rec.ID]; dup {
			return fmt.Errorf("duplicate follow id")
		}
		if _, dup := seenPlaylist[rec.PlaylistID]; dup {
			return fmt.Errorf("playlist is already followed")
		}
		seenID[rec.ID] = struct{}{}
		seenPlaylist[rec.PlaylistID] = struct{}{}
	}
	return nil
}

func validateItem(item Item) error {
	if !validVideoID(item.VideoID) || !validText(item.URL, maxTextBytes, true) || !validText(item.Title, maxTextBytes, true) ||
		!validText(item.Duration, maxIDBytes, false) || !validText(item.Thumbnail, maxTextBytes, false) || item.Index <= 0 {
		return fmt.Errorf("follow review item is incomplete")
	}
	return nil
}

func validID(value string) bool {
	return validText(value, maxIDBytes, true) && !strings.ContainsAny(value, `\/`)
}

func validVideoID(value string) bool {
	return len(value) == 11 && validText(value, 11, true)
}

func validText(value string, max int, required bool) bool {
	if required && strings.TrimSpace(value) == "" {
		return false
	}
	if len(value) > max || !utf8.ValidString(value) || strings.IndexByte(value, 0) >= 0 {
		return false
	}
	return true
}

func nowUTC() string {
	return time.Now().UTC().Format(time.RFC3339Nano)
}
