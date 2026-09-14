package main

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/tejasa97/vidstow/internal/follow"
	"github.com/tejasa97/vidstow/internal/jobs"
	"github.com/tejasa97/vidstow/internal/urlcheck"
)

const followCheckFailedMessage = "Couldn't check for updates."

var loadFollowListing = loadFollowListingDefault

type FollowPlaylistRequest struct {
	URL          string             `json:"url"`
	PlaylistID   string             `json:"playlistId"`
	Quality      jobs.Quality       `json:"quality"`
	AudioBitrate int                `json:"audioBitrate,omitempty"`
	Options      jobs.OutputOptions `json:"options,omitempty"`
	Scope        string             `json:"scope"`
}

type FollowReviewRequest struct {
	FollowID string   `json:"followId"`
	VideoIDs []string `json:"videoIds"`
}

type FollowOutputRequest struct {
	FollowID     string             `json:"followId"`
	Quality      jobs.Quality       `json:"quality"`
	AudioBitrate int                `json:"audioBitrate,omitempty"`
	Options      jobs.OutputOptions `json:"options,omitempty"`
	Folder       string             `json:"folder,omitempty"`
}

type FollowSnapshot struct {
	follow.Record
	CheckState string `json:"checkState"`
}

type FollowsView struct {
	Follows         []FollowSnapshot `json:"follows"`
	Checking        bool             `json:"checking"`
	CheckDone       int              `json:"checkDone"`
	CheckTotal      int              `json:"checkTotal"`
	CurrentFollowID string           `json:"currentFollowId,omitempty"`
}

func loadFollowListingDefault(a *App, ctx context.Context, playlistID, sourceURL string, refresh bool) (jobs.PlaylistSummary, error) {
	if a == nil || a.jobs == nil {
		return jobs.PlaylistSummary{}, errors.New("playlist listing is unavailable")
	}
	if !refresh {
		if preview, err := a.jobs.PlaylistPreview(playlistID); err == nil {
			return preview, nil
		}
	}
	if strings.TrimSpace(sourceURL) == "" {
		return jobs.PlaylistSummary{}, errors.New("playlist preview expired; analyze the playlist again")
	}
	return a.jobs.AnalyzePlaylist(ctx, sourceURL)
}

func (a *App) ListFollows() FollowsView {
	return a.followsView()
}

func (a *App) FollowPlaylist(req FollowPlaylistRequest) (FollowsView, error) {
	if err := a.requireReady(); err != nil {
		return FollowsView{}, err
	}
	if a.follows == nil {
		return FollowsView{}, errors.New("followed playlists could not be saved")
	}
	validated, err := urlcheck.Validate(req.URL)
	if err != nil {
		return FollowsView{}, err
	}
	if validated.Kind != urlcheck.KindPlaylist || validated.PlaylistID == "" || validated.PlaylistID != req.PlaylistID {
		return FollowsView{}, errors.New("playlist identity does not match the analyzed preview")
	}
	if _, err := validatePlaylistPolicy(req.Quality, req.AudioBitrate); err != nil {
		return FollowsView{}, err
	}
	if err := req.Options.Validate(); err != nil {
		return FollowsView{}, fmt.Errorf("invalid output options: %w", err)
	}
	scope := strings.TrimSpace(req.Scope)
	if scope == "" {
		scope = follow.ScopeFuture
	}
	if scope != follow.ScopeFuture && scope != follow.ScopeAll {
		return FollowsView{}, errors.New("choose future videos only or download all now")
	}
	listing, err := loadFollowListing(a, a.followContext(), validated.PlaylistID, validated.PlaylistURL, false)
	if err != nil {
		return FollowsView{}, err
	}
	if listing.ID != validated.PlaylistID {
		return FollowsView{}, errors.New("playlist identity does not match the analyzed preview")
	}
	settings := a.store.Settings()
	if strings.TrimSpace(settings.DownloadFolder) == "" {
		return FollowsView{}, errors.New("Choose a folder to follow this playlist.")
	}
	folder, err := canonicalOutputRequestPath(joinFollowFolder(settings.DownloadFolder, listing.Title, listing.ID))
	if err != nil {
		return FollowsView{}, err
	}
	now := followNow()
	rec := follow.Record{
		ID:            uuid.NewString(),
		PlaylistID:    listing.ID,
		SourceURL:     listing.URL,
		Title:         listing.Title,
		Channel:       listing.Channel,
		Thumbnail:     listing.Thumbnail,
		VideoCount:    listing.EntryCount,
		Output:        follow.Output{Quality: req.Quality, AudioBitrate: req.AudioBitrate, Options: req.Options.Clone(), Folder: folder},
		KnownVideoIDs: follow.BaselineKnownIDs(listing.Entries),
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if rec.Title == "" {
		rec.Title = "Playlist"
	}
	if err := a.follows.Put(rec); err != nil {
		return FollowsView{}, err
	}
	a.emitFollows()
	if scope != follow.ScopeAll {
		return a.followsView(), nil
	}
	indexes := follow.AvailableIndexes(listing.Entries)
	if len(indexes) == 0 {
		return a.followsView(), errors.New("This playlist has no available videos to download.")
	}
	_, err = a.admitPlaylistDownload(StartPlaylistRequest{
		URL: listing.URL, PlaylistID: listing.ID, Quality: req.Quality, AudioBitrate: req.AudioBitrate,
		SelectedItems: indexes, Options: req.Options,
	}, rec.Output.Folder)
	if err != nil {
		return a.followsView(), fmt.Errorf("Followed the playlist, but could not start downloads: %w", err)
	}
	return a.followsView(), nil
}

func (a *App) UnfollowPlaylist(id string) (FollowsView, error) {
	if err := a.requireReady(); err != nil {
		return FollowsView{}, err
	}
	if a.follows == nil {
		return FollowsView{}, errors.New("followed playlists could not be saved")
	}
	if _, err := a.follows.Delete(id); err != nil {
		return FollowsView{}, err
	}
	a.followMu.Lock()
	delete(a.followCheckStates, id)
	a.followMu.Unlock()
	a.emitFollows()
	return a.followsView(), nil
}

func (a *App) CheckFollow(id string) (FollowsView, error) {
	return a.runFollowChecks([]string{id})
}

func (a *App) CheckAllFollows() (FollowsView, error) {
	if err := a.requireReady(); err != nil {
		return FollowsView{}, err
	}
	if a.follows == nil {
		return FollowsView{}, errors.New("followed playlists could not be saved")
	}
	ids := make([]string, 0)
	for _, rec := range a.follows.List() {
		ids = append(ids, rec.ID)
	}
	if len(ids) == 0 {
		return a.followsView(), nil
	}
	return a.runFollowChecks(ids)
}

func (a *App) StopFollowChecks() FollowsView {
	a.followMu.Lock()
	cancel := a.followCheckCancel
	a.followMu.Unlock()
	if cancel != nil {
		cancel()
	}
	return a.followsView()
}

func (a *App) AdmitFollowReview(req FollowReviewRequest) (FollowsView, error) {
	if err := a.requireReady(); err != nil {
		return FollowsView{}, err
	}
	rec, err := a.requireFollow(req.FollowID)
	if err != nil {
		return FollowsView{}, err
	}
	listing, err := loadFollowListing(a, a.followContext(), rec.PlaylistID, rec.SourceURL, true)
	if err != nil {
		return FollowsView{}, err
	}
	indexes, err := follow.IndexesForVideoIDs(listing.Entries, req.VideoIDs)
	if err != nil {
		return FollowsView{}, err
	}
	_, err = a.admitPlaylistDownload(StartPlaylistRequest{
		URL: rec.SourceURL, PlaylistID: rec.PlaylistID, Quality: rec.Output.Quality,
		AudioBitrate: rec.Output.AudioBitrate, SelectedItems: indexes, Options: rec.Output.Options,
	}, rec.Output.Folder)
	if err != nil {
		return FollowsView{}, err
	}
	next, err := follow.AdmitPending(rec, req.VideoIDs, followNow())
	if err != nil {
		return FollowsView{}, err
	}
	if err := a.follows.Put(next); err != nil {
		return FollowsView{}, err
	}
	a.emitFollows()
	return a.followsView(), nil
}

func (a *App) SkipFollowItems(req FollowReviewRequest) (FollowsView, error) {
	return a.patchFollow(req.FollowID, func(rec follow.Record) (follow.Record, error) {
		return follow.SkipItems(rec, req.VideoIDs, followNow())
	})
}

func (a *App) RestoreFollowItems(req FollowReviewRequest) (FollowsView, error) {
	return a.patchFollow(req.FollowID, func(rec follow.Record) (follow.Record, error) {
		return follow.RestoreItems(rec, req.VideoIDs, followNow())
	})
}

func (a *App) UpdateFollowOutput(req FollowOutputRequest) (FollowsView, error) {
	return a.patchFollow(req.FollowID, func(rec follow.Record) (follow.Record, error) {
		next := follow.CloneRecord(rec)
		next.Output.Quality = req.Quality
		next.Output.AudioBitrate = req.AudioBitrate
		next.Output.Options = req.Options.Clone()
		if strings.TrimSpace(req.Folder) != "" {
			folder, err := canonicalOutputRequestPath(req.Folder)
			if err != nil {
				return follow.Record{}, err
			}
			next.Output.Folder = folder
		}
		if err := follow.ValidateOutput(next.Output); err != nil {
			return follow.Record{}, err
		}
		next.UpdatedAt = followNow()
		return next, nil
	})
}

func (a *App) runFollowChecks(ids []string) (FollowsView, error) {
	if err := a.requireReady(); err != nil {
		return FollowsView{}, err
	}
	if a.follows == nil {
		return FollowsView{}, errors.New("followed playlists could not be saved")
	}
	if len(ids) == 0 {
		return a.followsView(), nil
	}
	a.followMu.Lock()
	if a.followCheckCancel != nil {
		a.followMu.Unlock()
		return FollowsView{}, errors.New("A check is already running.")
	}
	ctx, cancel := context.WithCancel(a.followContext())
	a.followCheckCancel = cancel
	a.followCheckDone = 0
	a.followCheckTotal = len(ids)
	a.followCheckCurrent = ""
	a.followCheckStates = map[string]string{}
	for _, id := range ids {
		a.followCheckStates[id] = follow.CheckWaiting
	}
	a.followMu.Unlock()
	a.emitFollows()
	defer func() {
		a.followMu.Lock()
		if a.followCheckCancel != nil {
			a.followCheckCancel()
		}
		a.followCheckCancel = nil
		a.followCheckStates = nil
		a.followCheckDone = 0
		a.followCheckTotal = 0
		a.followCheckCurrent = ""
		a.followMu.Unlock()
		a.emitFollows()
	}()

	var firstErr error
	for _, id := range ids {
		if err := ctx.Err(); err != nil {
			break
		}
		a.setFollowCheckState(id, follow.CheckChecking)
		a.emitFollows()
		rec, ok := a.follows.Get(id)
		if !ok {
			a.advanceFollowCheck(id)
			continue
		}
		listing, err := loadFollowListing(a, ctx, rec.PlaylistID, rec.SourceURL, true)
		if err != nil {
			if ctx.Err() != nil {
				break
			}
			message := followCheckFailedMessage
			_ = a.follows.Put(follow.MarkCheckFailed(rec, message, followNow()))
			if firstErr == nil {
				firstErr = errors.New(message)
			}
			a.advanceFollowCheck(id)
			a.emitFollows()
			continue
		}
		next := follow.ApplyCheck(rec, follow.Listing{
			PlaylistID: listing.ID, SourceURL: listing.URL, Title: listing.Title, Channel: listing.Channel,
			Thumbnail: listing.Thumbnail, VideoCount: listing.EntryCount, Entries: listing.Entries,
		}, followNow())
		if err := a.follows.Put(next); err != nil && firstErr == nil {
			firstErr = err
		}
		a.advanceFollowCheck(id)
		a.emitFollows()
	}
	return a.followsView(), firstErr
}

func (a *App) patchFollow(id string, mutate func(follow.Record) (follow.Record, error)) (FollowsView, error) {
	if err := a.requireReady(); err != nil {
		return FollowsView{}, err
	}
	rec, err := a.requireFollow(id)
	if err != nil {
		return FollowsView{}, err
	}
	next, err := mutate(rec)
	if err != nil {
		return FollowsView{}, err
	}
	if err := a.follows.Put(next); err != nil {
		return FollowsView{}, err
	}
	a.emitFollows()
	return a.followsView(), nil
}

func (a *App) requireFollow(id string) (follow.Record, error) {
	if a.follows == nil {
		return follow.Record{}, errors.New("followed playlists could not be saved")
	}
	rec, ok := a.follows.Get(id)
	if !ok {
		return follow.Record{}, errors.New("that playlist is not followed")
	}
	return rec, nil
}

func (a *App) followsView() FollowsView {
	view := FollowsView{Follows: []FollowSnapshot{}}
	if a == nil || a.follows == nil {
		return view
	}
	a.followMu.Lock()
	states := a.followCheckStates
	view.Checking = a.followCheckCancel != nil
	view.CheckDone = a.followCheckDone
	view.CheckTotal = a.followCheckTotal
	view.CurrentFollowID = a.followCheckCurrent
	a.followMu.Unlock()
	for _, rec := range a.follows.List() {
		state := follow.CheckIdle
		if states != nil {
			if next, ok := states[rec.ID]; ok && next != "" {
				state = next
			}
		}
		view.Follows = append(view.Follows, FollowSnapshot{Record: rec, CheckState: state})
	}
	return view
}

func (a *App) emitFollows() {
	if a == nil || a.ctx == nil {
		return
	}
	emitAppEvent(a.ctx, "follows:update", a.followsView())
}

func (a *App) followContext() context.Context {
	if a != nil && a.ctx != nil {
		return a.ctx
	}
	return context.Background()
}

func (a *App) setFollowCheckState(id, state string) {
	a.followMu.Lock()
	defer a.followMu.Unlock()
	if a.followCheckStates == nil {
		a.followCheckStates = map[string]string{}
	}
	a.followCheckStates[id] = state
	a.followCheckCurrent = id
}

func (a *App) advanceFollowCheck(id string) {
	a.followMu.Lock()
	defer a.followMu.Unlock()
	if a.followCheckStates != nil {
		a.followCheckStates[id] = follow.CheckIdle
	}
	a.followCheckDone++
}

func joinFollowFolder(root, title, playlistID string) string {
	return filepath.Join(root, playlistSubfolder(title, playlistID))
}

func followNow() string {
	return time.Now().UTC().Format(time.RFC3339Nano)
}
