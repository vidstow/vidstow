package store

import (
	"errors"

	"github.com/google/uuid"
	"github.com/tejasa97/vidstow/internal/authsource"
	"github.com/tejasa97/vidstow/internal/jobmodel"
)

// BrowserSourceStatus is the bounded owner-facing projection of one durable
// non-secret source binding.
type BrowserSourceStatus struct {
	BindingRef string             `json:"bindingRef"`
	Browser    authsource.Browser `json:"browser"`
	Label      string             `json:"label"`
	Enabled    bool               `json:"enabled"`
	Default    bool               `json:"default"`
}

func (s *V2Store) BrowserSourceOptions() []authsource.Option {
	return authsource.SupportedOptions()
}

func (s *V2Store) BrowserSources() []BrowserSourceStatus {
	if s == nil || !s.Status().Healthy() {
		return []BrowserSourceStatus{}
	}
	state := s.Snapshot()
	out := make([]BrowserSourceStatus, 0, len(state.AuthSourceBindings))
	for _, binding := range state.AuthSourceBindings {
		out = append(out, BrowserSourceStatus{
			BindingRef: binding.ID,
			Browser:    binding.Descriptor.Browser,
			Label:      authsource.Label(binding.Descriptor),
			Enabled:    binding.Enabled,
			Default:    binding.ID == state.Settings.DefaultAuthSourceBindingRef,
		})
	}
	return out
}

// ConfigureBrowserSource persists only a descriptor selected from the
// backend-owned allowlist. It performs no browser access; the first explicit
// authenticated operation imports fresh browser data through ytdlp-go.
func (s *V2Store) ConfigureBrowserSource(optionID string, consentVersion int) (BrowserSourceStatus, error) {
	if s == nil {
		return BrowserSourceStatus{}, errors.New("browser source unavailable")
	}
	if consentVersion != authsource.CurrentConsentVersion {
		return BrowserSourceStatus{}, errors.New("current browser access consent is required")
	}
	descriptor, err := authsource.DescriptorForOption(optionID)
	if err != nil {
		return BrowserSourceStatus{}, errors.New("browser source is not supported")
	}
	return s.configureBrowserSource(descriptor, consentVersion)
}

func (s *V2Store) configureBrowserSource(descriptor authsource.Descriptor, consentVersion int) (BrowserSourceStatus, error) {
	if consentVersion != authsource.CurrentConsentVersion || authsource.ValidateDescriptor(descriptor) != nil {
		return BrowserSourceStatus{}, errors.New("current browser access consent is required")
	}
	var binding authsource.Binding
	err := s.Transaction(nil, func(state *jobmodel.State) error {
		for index := range state.AuthSourceBindings {
			candidate := state.AuthSourceBindings[index]
			if candidate.Enabled && candidate.Descriptor == descriptor {
				binding = candidate
				break
			}
		}
		// Disabled bindings are immutable tombstones. Reconfiguring the same
		// browser creates fresh authority so old analysis tokens and admitted
		// jobs can never be revived by a settings change.
		if binding.ID == "" {
			binding = authsource.Binding{ID: uuid.NewString(), Descriptor: descriptor, Enabled: true}
			state.AuthSourceBindings = append(state.AuthSourceBindings, binding)
		}
		state.Settings.BrowserAccessEnabled = true
		state.Settings.DefaultAuthSourceBindingRef = binding.ID
		state.Settings.BrowserAccessConsentVersion = authsource.CurrentConsentVersion
		return nil
	})
	if err != nil {
		return BrowserSourceStatus{}, err
	}
	return BrowserSourceStatus{BindingRef: binding.ID, Browser: binding.Descriptor.Browser, Label: authsource.Label(binding.Descriptor), Enabled: true, Default: true}, nil
}
