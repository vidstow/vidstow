package follow

import (
	"fmt"
	"sort"

	"github.com/tejasa97/vidstow/internal/jobs"
)

// Listing is one AnalyzePlaylist snapshot used for follow baseline and checks.
type Listing struct {
	PlaylistID string
	SourceURL  string
	Title      string
	Channel    string
	Thumbnail  string
	VideoCount int
	Entries    []jobs.PlaylistEntrySummary
}

func ItemFromEntry(entry jobs.PlaylistEntrySummary) Item {
	title := entry.Title
	if title == "" {
		title = "Untitled video"
	}
	return Item{
		VideoID:   entry.VideoID,
		URL:       entry.URL,
		Title:     title,
		Duration:  entry.Duration,
		Thumbnail: entry.Thumbnail,
		Available: entry.Available,
		Index:     entry.Index,
	}
}

// BaselineKnownIDs records every current playlist video so those entries are
// not treated as new on the next check.
func BaselineKnownIDs(entries []jobs.PlaylistEntrySummary) []string {
	seen := map[string]struct{}{}
	ids := make([]string, 0, len(entries))
	for _, entry := range entries {
		if len(entry.VideoID) != 11 {
			continue
		}
		if _, ok := seen[entry.VideoID]; ok {
			continue
		}
		seen[entry.VideoID] = struct{}{}
		ids = append(ids, entry.VideoID)
	}
	return ids
}

func AvailableIndexes(entries []jobs.PlaylistEntrySummary) []int {
	indexes := make([]int, 0, len(entries))
	for _, entry := range entries {
		if entry.Available && entry.URL != "" && len(entry.VideoID) == 11 {
			indexes = append(indexes, entry.Index)
		}
	}
	return indexes
}

// IndexesForVideoIDs maps follow review video IDs onto the current trusted
// playlist preview. Order follows the playlist, not the request.
func IndexesForVideoIDs(entries []jobs.PlaylistEntrySummary, videoIDs []string) ([]int, error) {
	if len(videoIDs) == 0 || len(videoIDs) > jobs.MaxPlaylistEntries {
		return nil, fmt.Errorf("invalid follow review selection")
	}
	wanted := map[string]struct{}{}
	for _, id := range videoIDs {
		if len(id) != 11 {
			return nil, fmt.Errorf("invalid follow review video")
		}
		if _, dup := wanted[id]; dup {
			return nil, fmt.Errorf("duplicate follow review video")
		}
		wanted[id] = struct{}{}
	}
	indexes := make([]int, 0, len(wanted))
	for _, entry := range entries {
		if _, ok := wanted[entry.VideoID]; !ok {
			continue
		}
		if !entry.Available || entry.URL == "" || len(entry.VideoID) != 11 {
			return nil, fmt.Errorf("a selected video is no longer available")
		}
		indexes = append(indexes, entry.Index)
		delete(wanted, entry.VideoID)
	}
	if len(wanted) != 0 {
		return nil, fmt.Errorf("a selected video is no longer in this playlist")
	}
	return indexes, nil
}

// ApplyCheck diffs a fresh listing against the known set. New available videos
// join pending. Stop-safe: callers persist after each playlist.
func ApplyCheck(rec Record, listing Listing, checkedAt string) Record {
	out := CloneRecord(rec)
	out.Title = firstNonEmpty(listing.Title, out.Title)
	out.Channel = firstNonEmpty(listing.Channel, out.Channel)
	out.Thumbnail = firstNonEmpty(listing.Thumbnail, out.Thumbnail)
	if listing.VideoCount > 0 {
		out.VideoCount = listing.VideoCount
	}
	alreadyKnown := map[string]struct{}{}
	for _, id := range out.KnownVideoIDs {
		alreadyKnown[id] = struct{}{}
	}
	known := map[string]struct{}{}
	for id := range alreadyKnown {
		known[id] = struct{}{}
	}
	pendingByID := map[string]int{}
	for i, item := range out.Pending {
		pendingByID[item.VideoID] = i
	}
	skipped := map[string]struct{}{}
	for _, item := range out.Skipped {
		skipped[item.VideoID] = struct{}{}
	}
	for _, entry := range listing.Entries {
		if len(entry.VideoID) != 11 {
			continue
		}
		item := ItemFromEntry(entry)
		known[entry.VideoID] = struct{}{}
		if _, isSkipped := skipped[entry.VideoID]; isSkipped {
			continue
		}
		if index, isPending := pendingByID[entry.VideoID]; isPending {
			out.Pending[index] = item
			continue
		}
		if _, seen := alreadyKnown[entry.VideoID]; seen {
			continue
		}
		if entry.Available && entry.URL != "" {
			pendingByID[entry.VideoID] = len(out.Pending)
			out.Pending = append(out.Pending, item)
		}
	}
	out.KnownVideoIDs = sortedIDs(known)
	out.LastCheckedAt = checkedAt
	out.LastCheckError = ""
	out.UpdatedAt = checkedAt
	return out
}

func MarkCheckFailed(rec Record, message, updatedAt string) Record {
	out := CloneRecord(rec)
	out.LastCheckError = message
	out.UpdatedAt = updatedAt
	return out
}

func SkipItems(rec Record, videoIDs []string, updatedAt string) (Record, error) {
	out := CloneRecord(rec)
	wanted := uniqueIDs(videoIDs)
	if len(wanted) == 0 {
		return Record{}, fmt.Errorf("select videos to skip")
	}
	kept := out.Pending[:0]
	moved := 0
	skippedByID := map[string]Item{}
	for _, item := range out.Skipped {
		skippedByID[item.VideoID] = item
	}
	for _, item := range out.Pending {
		if _, ok := wanted[item.VideoID]; ok {
			skippedByID[item.VideoID] = item
			moved++
			continue
		}
		kept = append(kept, item)
	}
	if moved != len(wanted) {
		return Record{}, fmt.Errorf("a selected video is not waiting in review")
	}
	out.Pending = kept
	out.Skipped = itemsFromMap(skippedByID)
	out.UpdatedAt = updatedAt
	return out, nil
}

func RestoreItems(rec Record, videoIDs []string, updatedAt string) (Record, error) {
	out := CloneRecord(rec)
	wanted := uniqueIDs(videoIDs)
	if len(wanted) == 0 {
		return Record{}, fmt.Errorf("select videos to restore")
	}
	kept := out.Skipped[:0]
	moved := 0
	pendingByID := map[string]Item{}
	for _, item := range out.Pending {
		pendingByID[item.VideoID] = item
	}
	for _, item := range out.Skipped {
		if _, ok := wanted[item.VideoID]; ok {
			pendingByID[item.VideoID] = item
			moved++
			continue
		}
		kept = append(kept, item)
	}
	if moved != len(wanted) {
		return Record{}, fmt.Errorf("a selected video is not in the skipped list")
	}
	out.Skipped = kept
	out.Pending = itemsFromMap(pendingByID)
	out.UpdatedAt = updatedAt
	return out, nil
}

func AdmitPending(rec Record, videoIDs []string, updatedAt string) (Record, error) {
	out := CloneRecord(rec)
	wanted := uniqueIDs(videoIDs)
	if len(wanted) == 0 {
		return Record{}, fmt.Errorf("select videos to download")
	}
	kept := out.Pending[:0]
	moved := 0
	for _, item := range out.Pending {
		if _, ok := wanted[item.VideoID]; ok {
			moved++
			continue
		}
		kept = append(kept, item)
	}
	if moved != len(wanted) {
		return Record{}, fmt.Errorf("a selected video is not waiting in review")
	}
	out.Pending = kept
	out.UpdatedAt = updatedAt
	return out, nil
}

func uniqueIDs(ids []string) map[string]struct{} {
	wanted := map[string]struct{}{}
	for _, id := range ids {
		if id == "" {
			continue
		}
		wanted[id] = struct{}{}
	}
	return wanted
}

func sortedIDs(ids map[string]struct{}) []string {
	out := make([]string, 0, len(ids))
	for id := range ids {
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}

func itemsFromMap(items map[string]Item) []Item {
	out := make([]Item, 0, len(items))
	for _, item := range items {
		out = append(out, item)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Index == out[j].Index {
			return out[i].VideoID < out[j].VideoID
		}
		return out[i].Index < out[j].Index
	})
	return out
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
