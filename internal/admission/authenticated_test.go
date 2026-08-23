package admission

import (
	"context"
	"testing"

	"github.com/tejasa97/vidstow/internal/authsource"
	"github.com/tejasa97/vidstow/internal/jobmodel"
	"github.com/tejasa97/vidstow/internal/outputplan"
)

func TestAuthenticatedAdmissionPersistsAuthorityBindingNotSettingsDefault(t *testing.T) {
	plan := fixturePlanForAuthTest()
	coordinator, root, state, request, _ := admissionFixture(t, plan)
	if err := state.Transaction(nil, func(document *jobmodel.State) error {
		document.AuthSourceBindings = []authsource.Binding{
			{ID: "chrome-binding", Enabled: true, Descriptor: authsource.Descriptor{SchemaVersion: 1, Platform: "darwin", Browser: authsource.BrowserChrome}},
			{ID: "safari-binding", Enabled: true, Descriptor: authsource.Descriptor{SchemaVersion: 1, Platform: "darwin", Browser: authsource.BrowserSafari}},
		}
		document.Settings.BrowserAccessEnabled = true
		document.Settings.BrowserAccessConsentVersion = authsource.CurrentConsentVersion
		document.Settings.DefaultAuthSourceBindingRef = "safari-binding"
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	request.ResolvedPlan = &plan
	request.AuthIntent = jobmodel.AuthIntent{RequiresAuthenticatedExecution: true, AuthSourceBindingRef: "chrome-binding"}
	result, err := coordinator.Admit(context.Background(), root, request)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Job.AuthIntent.RequiresAuthenticatedExecution || result.Job.AuthIntent.AuthSourceBindingRef != "chrome-binding" {
		t.Fatalf("durable intent = %#v", result.Job.AuthIntent)
	}
	if got := state.Settings().DefaultAuthSourceBindingRef; got != "safari-binding" {
		t.Fatalf("Settings default changed to %q", got)
	}
}

func TestAuthenticatedAdmissionRejectsDisabledBoundSource(t *testing.T) {
	plan := fixturePlanForAuthTest()
	coordinator, root, state, request, _ := admissionFixture(t, plan)
	if err := state.Transaction(nil, func(document *jobmodel.State) error {
		document.AuthSourceBindings = []authsource.Binding{{ID: "chrome-binding", Enabled: false, Descriptor: authsource.Descriptor{SchemaVersion: 1, Platform: "darwin", Browser: authsource.BrowserChrome}}}
		document.Settings.BrowserAccessEnabled = true
		document.Settings.BrowserAccessConsentVersion = authsource.CurrentConsentVersion
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	before := state.Snapshot()
	request.ResolvedPlan = &plan
	request.AuthIntent = jobmodel.AuthIntent{RequiresAuthenticatedExecution: true, AuthSourceBindingRef: "chrome-binding"}
	if _, err := coordinator.Admit(context.Background(), root, request); err == nil {
		t.Fatal("disabled source admission unexpectedly succeeded")
	}
	after := state.Snapshot()
	if len(after.Jobs) != len(before.Jobs) || after.NextQueueOrdinal != before.NextQueueOrdinal {
		t.Fatalf("rejected admission mutated State v2: before=%#v after=%#v", before, after)
	}
}

func fixturePlanForAuthTest() outputplan.Plan {
	return outputplan.Plan{ID: "video-1080-mp4", Kind: outputplan.KindVideo, Label: "1080p", Container: "MP4", Selector: "137+140"}
}
