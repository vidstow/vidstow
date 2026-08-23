package admission

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/tejasa97/vidstow/internal/jobmodel"
	"github.com/tejasa97/vidstow/internal/jobs"
	"github.com/tejasa97/vidstow/internal/outputplan"
	"github.com/tejasa97/vidstow/internal/reservation"
	"github.com/tejasa97/vidstow/internal/reservationfs"
	"github.com/tejasa97/ytdlp-go/engine"
	"github.com/tejasa97/ytdlp-go/engine/value"
)

const MaxCollectionChildren = 500

// Collection describes the durable parent identity. Policy is a reviewed,
// server-authored label such as "video:1080p"; raw selectors never enter it.
type Collection struct {
	Kind         jobmodel.CollectionKind
	PlaylistID   string
	SourceURL    string
	Title        string
	Channel      string
	Thumbnail    string
	Policy       string
	AuthIntent   jobmodel.AuthIntent
	Discovered   int
	Ready        int
	AuthRequired int
	Unavailable  int
	Invalid      int
	Approved     int
}

// CollectionRequest contains children that the application has independently
// resolved from a trusted playlist preview and analyzed on the backend.
type CollectionChildRequest struct {
	Request Request
	// ResolvedPlan is selected from private backend analysis metadata. It must
	// never be decoded from a renderer request.
	ResolvedPlan outputplan.Plan
	// Root optionally binds this child to a distinct caller-owned output root.
	// Playlist children normally share the collection root; pasted batches may
	// use per-video subfolders without weakening atomic reservation admission.
	Root *reservationfs.Root
}

type CollectionRequest struct {
	Collection Collection
	Children   []CollectionChildRequest
}

type CollectionChildResult struct {
	Job         jobmodel.DurableJob
	Plan        outputplan.Plan
	Reservation jobmodel.ReservationSet
	Artifacts   []engine.ArtifactDeclaration
}

type CollectionResult struct {
	Collection jobmodel.DurableCollection
	Children   []CollectionChildResult
	Submitted  int
}

type preparedCollectionChild struct {
	request      Request
	plan         outputplan.Plan
	artifacts    []engine.ArtifactDeclaration
	declarations []reservation.ArtifactDeclaration
	selector     *reservation.Selector
	volume       reservation.Volume
	rootRef      jobmodel.OutputRootRef
	jobID        string
	attemptID    string
	sessionID    string
}

type preparedReservationRoot struct {
	selector *reservation.Selector
	volume   reservation.Volume
	rootRef  jobmodel.OutputRootRef
}

// AdmitCollection chooses every child reservation and commits the parent and
// all pending children in one State v2 transaction. Only after that atomic
// commit succeeds are the children handed to the live FIFO manager.
func (c *Coordinator) AdmitCollection(ctx context.Context, root *reservationfs.Root, request CollectionRequest) (CollectionResult, error) {
	if c == nil {
		return CollectionResult{}, errors.New("admission: nil coordinator")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if root == nil {
		return CollectionResult{}, errors.New("admission: nil reservation root")
	}
	if err := ctx.Err(); err != nil {
		return CollectionResult{}, err
	}
	if request.Collection.Kind == "" {
		request.Collection.Kind = jobmodel.CollectionKindPlaylist
	}
	if strings.TrimSpace(request.Collection.Title) == "" || strings.TrimSpace(request.Collection.Policy) == "" {
		return CollectionResult{}, errors.New("admission: incomplete collection identity")
	}
	switch request.Collection.Kind {
	case jobmodel.CollectionKindPlaylist:
		if strings.TrimSpace(request.Collection.PlaylistID) == "" || strings.TrimSpace(request.Collection.SourceURL) == "" {
			return CollectionResult{}, errors.New("admission: incomplete playlist collection identity")
		}
	case jobmodel.CollectionKindBatch:
		if request.Collection.PlaylistID != "" || request.Collection.SourceURL != "" {
			return CollectionResult{}, errors.New("admission: batch collection cannot retain playlist source metadata")
		}
	default:
		return CollectionResult{}, errors.New("admission: unsupported collection kind")
	}
	if len(request.Children) == 0 || len(request.Children) > MaxCollectionChildren {
		return CollectionResult{}, fmt.Errorf("admission: collection must contain between 1 and %d children", MaxCollectionChildren)
	}

	rootCache := make(map[*reservationfs.Root]preparedReservationRoot)
	prepareRoot := func(candidate *reservationfs.Root) (preparedReservationRoot, error) {
		if prepared, ok := rootCache[candidate]; ok {
			return prepared, nil
		}
		facts := candidate.Facts()
		if facts.Volume.CanonicalPath == "" || facts.Volume.Identity == "" || facts.Policies().Names == nil || facts.Policies().Volumes == nil || facts.Probe == nil {
			return preparedReservationRoot{}, errors.New("admission: incomplete reservation root facts")
		}
		engineRoot, err := engine.ValidateOutputRoot(facts.Volume.CanonicalPath)
		if err != nil {
			return preparedReservationRoot{}, fmt.Errorf("admission: engine output root validation: %w", err)
		}
		if engineRoot.CanonicalPath != facts.Volume.CanonicalPath {
			return preparedReservationRoot{}, errors.New("admission: engine and reservation output roots differ")
		}
		selector, err := reservation.NewSelector(reservation.Options{Policies: facts.Policies(), Probe: facts.Probe, MaxSuffix: c.maxSuffix})
		if err != nil {
			return preparedReservationRoot{}, fmt.Errorf("admission: configure reservation selector: %w", err)
		}
		prepared := preparedReservationRoot{
			selector: selector,
			volume:   facts.Volume,
			rootRef:  jobmodel.OutputRootRef{CanonicalPath: facts.Volume.CanonicalPath, Identity: facts.Volume.Identity, EngineIdentity: engineRoot.Identity},
		}
		rootCache[candidate] = prepared
		return prepared, nil
	}
	if _, err := prepareRoot(root); err != nil {
		return CollectionResult{}, err
	}

	collectionID := c.deps.NewCollectionID()
	if collectionID == "" || strings.ContainsAny(collectionID, `\/`) {
		return CollectionResult{}, errors.New("admission: generated collection identity is invalid")
	}
	if request.Collection.AuthIntent.RequiresAuthenticatedExecution != (request.Collection.AuthIntent.AuthSourceBindingRef != "") {
		return CollectionResult{}, errors.New("admission: invalid shared collection browser authority")
	}
	if request.Collection.AuthIntent.RequiresAuthenticatedExecution {
		if request.Collection.Discovered <= 0 || request.Collection.Ready <= 0 || request.Collection.Approved != len(request.Children) ||
			request.Collection.Approved > request.Collection.Ready || request.Collection.Discovered != request.Collection.Ready+request.Collection.AuthRequired+request.Collection.Unavailable+request.Collection.Invalid {
			return CollectionResult{}, errors.New("admission: invalid authenticated playlist outcome counts")
		}
	} else if request.Collection.Discovered != 0 || request.Collection.Ready != 0 || request.Collection.AuthRequired != 0 || request.Collection.Unavailable != 0 || request.Collection.Invalid != 0 || request.Collection.Approved != 0 {
		return CollectionResult{}, errors.New("admission: public collection cannot claim authenticated review counts")
	}
	prepared := make([]preparedCollectionChild, len(request.Children))
	seenIDs := map[string]struct{}{collectionID: {}}
	seenOccurrences := make(map[string]struct{}, len(request.Children))
	lastSourceIndex := 0
	for index, collectionChild := range request.Children {
		child := collectionChild.Request
		if child.AuthIntent != request.Collection.AuthIntent {
			return CollectionResult{}, fmt.Errorf("admission: collection child %d browser authority differs from parent", index+1)
		}
		if request.Collection.AuthIntent.RequiresAuthenticatedExecution {
			if child.SourceIndex <= lastSourceIndex || uuid.Validate(child.SourceOccurrenceID) != nil {
				return CollectionResult{}, fmt.Errorf("admission: invalid authenticated collection occurrence %d", index+1)
			}
			if _, duplicate := seenOccurrences[child.SourceOccurrenceID]; duplicate {
				return CollectionResult{}, errors.New("admission: duplicate authenticated collection occurrence")
			}
			seenOccurrences[child.SourceOccurrenceID] = struct{}{}
			lastSourceIndex = child.SourceIndex
		} else if child.SourceIndex != 0 || child.SourceOccurrenceID != "" {
			return CollectionResult{}, errors.New("admission: public collection cannot claim authenticated occurrence authority")
		}
		if err := ctx.Err(); err != nil {
			return CollectionResult{}, err
		}
		if child.Queue.URL == "" || child.Queue.VideoID == "" || child.Queue.Title == "" || child.Queue.OutputDir == "" || child.Queue.PlanID == "" {
			return CollectionResult{}, fmt.Errorf("admission: incomplete collection child %d", index+1)
		}
		childRoot := collectionChild.Root
		if childRoot == nil {
			childRoot = root
		}
		preparedRoot, err := prepareRoot(childRoot)
		if err != nil {
			return CollectionResult{}, fmt.Errorf("admission: prepare collection child %d root: %w", index+1, err)
		}
		if err := verifyOutputRoot(child.Queue.OutputDir, preparedRoot.volume.CanonicalPath); err != nil {
			return CollectionResult{}, err
		}
		plan := collectionChild.ResolvedPlan
		if err := validateResolvedPlan(plan, child.Queue.PlanID); err != nil {
			return CollectionResult{}, err
		}
		metadata := value.NewInfo(child.Metadata.Fields().Clone())
		metadata.Set("title", value.String(child.Queue.Title))
		metadata.Set("id", value.String(child.Queue.VideoID))
		if child.Queue.Channel != "" {
			metadata.Set("channel", value.String(child.Queue.Channel))
		}
		artifacts, renderErr := engine.RenderOutputArtifacts(engine.OutputPreviewRequest{
			Template: jobs.OutputTemplateForPlan(plan), Metadata: metadata,
			Extension: strings.ToLower(strings.TrimPrefix(plan.Container, ".")),
		})
		if renderErr != nil || len(artifacts) == 0 {
			if renderErr == nil {
				renderErr = errors.New("engine returned no output artifacts")
			}
			return CollectionResult{}, fmt.Errorf("admission: render child %d output artifacts: %w", index+1, renderErr)
		}
		declarations := make([]reservation.ArtifactDeclaration, len(artifacts))
		for artifactIndex, artifact := range artifacts {
			declarations[artifactIndex] = reservation.ArtifactDeclaration{Kind: string(artifact.Kind), Identity: artifact.Identity, ProposedBasename: artifact.ProposedBasename}
		}
		jobID, attemptID, sessionID := c.deps.NewIDs()
		if err := validateGeneratedIDs(jobID, attemptID, sessionID); err != nil {
			return CollectionResult{}, err
		}
		for _, id := range []string{jobID, attemptID, sessionID} {
			if _, duplicate := seenIDs[id]; duplicate {
				return CollectionResult{}, errors.New("admission: duplicate generated collection identity")
			}
			seenIDs[id] = struct{}{}
		}
		prepared[index] = preparedCollectionChild{
			request: child, plan: plan, artifacts: artifacts, declarations: declarations,
			selector: preparedRoot.selector, volume: preparedRoot.volume, rootRef: preparedRoot.rootRef,
			jobID: jobID, attemptID: attemptID, sessionID: sessionID,
		}
	}

	now := c.deps.Now().UTC()
	if now.IsZero() {
		return CollectionResult{}, errors.New("admission: clock returned zero time")
	}
	result := CollectionResult{Children: make([]CollectionChildResult, len(prepared))}
	err := c.deps.Store.Transaction(nil, func(state *jobmodel.State) error {
		if err := validateAuthIntent(*state, request.Collection.AuthIntent); err != nil {
			return err
		}
		if state.NextQueueOrdinal > ^uint64(0)-uint64(len(prepared)) {
			return errors.New("admission: queue ordinal exhausted")
		}
		active := activeReservations(*state)
		childIDs := make([]string, len(prepared))
		for index, child := range prepared {
			selected, selectErr := child.selector.Select(ctx, reservation.SelectionRequest{
				GroupID:   child.jobID,
				Directory: child.volume,
				Artifacts: child.declarations,
			}, active)
			if selectErr != nil {
				return selectErr
			}
			jobReservation := toJobReservation(selected)
			if _, outputErr := primaryOutput(jobReservation); outputErr != nil {
				return outputErr
			}
			quality := child.request.Queue.Quality
			if quality == "" {
				quality = jobs.QualityBest
			}
			durable := jobmodel.DurableJob{
				ID: child.jobID, CollectionID: collectionID, CollectionIndex: index + 1,
				SourceIndex: child.request.SourceIndex, SourceOccurrenceID: child.request.SourceOccurrenceID,
				Revision: 1, AttemptID: child.attemptID, SessionID: child.sessionID,
				QueueOrdinal: state.NextQueueOrdinal, Lifecycle: jobmodel.LifecyclePending,
				Phase: jobmodel.PhasePreparing, Desired: jobmodel.DesiredRunning,
				Request:    jobmodel.PersistedRequest{SourceURL: child.request.Queue.URL, VideoID: child.request.Queue.VideoID, Title: child.request.Queue.Title, Channel: child.request.Queue.Channel, Quality: string(quality), PlanID: child.request.Queue.PlanID, Duration: child.request.Queue.Duration},
				Plan:       jobmodel.PersistedPlan{ID: child.plan.ID, Kind: string(child.plan.Kind), Label: child.plan.Label, Container: child.plan.Container, VideoCodec: child.plan.VideoCodec, AudioCodec: child.plan.AudioCodec, RequiresFFmpeg: child.plan.RequiresFFmpeg, PrivateSelector: child.plan.Selector},
				OutputRoot: child.rootRef, Reservation: jobReservation, RetryMode: jobmodel.RetryModeNone,
				AuthIntent: child.request.AuthIntent, CreatedAt: now, UpdatedAt: now,
			}
			state.Jobs = append(state.Jobs, durable)
			state.NextQueueOrdinal++
			active = append(active, toReservation(jobReservation))
			childIDs[index] = child.jobID
			result.Children[index] = CollectionChildResult{Job: durable, Plan: child.plan, Reservation: jobReservation, Artifacts: child.artifacts}
		}
		result.Collection = jobmodel.DurableCollection{
			ID: collectionID, Revision: 1, Kind: request.Collection.Kind, PlaylistID: request.Collection.PlaylistID,
			SourceURL: request.Collection.SourceURL, Title: request.Collection.Title,
			Channel: request.Collection.Channel, Thumbnail: request.Collection.Thumbnail,
			Policy: request.Collection.Policy, AuthIntent: request.Collection.AuthIntent,
			Discovered: request.Collection.Discovered, Ready: request.Collection.Ready, AuthRequired: request.Collection.AuthRequired,
			Unavailable: request.Collection.Unavailable, Invalid: request.Collection.Invalid, Approved: request.Collection.Approved,
			ChildJobIDs: childIDs, CreatedAt: now, UpdatedAt: now,
		}
		state.Collections = append(state.Collections, result.Collection)
		return nil
	})
	if err != nil {
		return CollectionResult{}, fmt.Errorf("admission: commit collection reservations and jobs: %w", err)
	}

	for index, child := range prepared {
		output, outputErr := primaryOutput(result.Children[index].Reservation)
		if outputErr != nil {
			return result, outputErr
		}
		queueRequest := child.request.Queue
		if queueRequest.Quality == "" {
			queueRequest.Quality = jobs.QualityBest
		}
		admittedID, submitErr := c.deps.Queue.SubmitAdmitted(child.jobID, queueRequest, &child.plan, output)
		if submitErr != nil {
			return result, fmt.Errorf("admission: FIFO manager child %d: %w", index+1, submitErr)
		}
		if admittedID != child.jobID {
			return result, fmt.Errorf("admission: FIFO manager returned job %q for admitted child %q", admittedID, child.jobID)
		}
		result.Submitted++
	}
	return result, nil
}
