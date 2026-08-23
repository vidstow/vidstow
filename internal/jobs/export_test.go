package jobs

import (
	"context"

	"github.com/tejasa97/vidstow/internal/authsource"
	"github.com/tejasa97/ytdlp-go/engine"
)

// InstallDownloadHooksForTest replaces the manager's inspect and download
// seams. It exists so external tests can drive Retry against a real V2Store.
func InstallDownloadHooksForTest(
	m *Manager,
	inspect func(context.Context, engine.OutputRootRef, string) (engine.ResumeSummary, error),
	run func(context.Context, engine.Request, engine.EventHandler) (engine.Result, error),
) {
	if inspect != nil {
		m.inspectResume = inspect
	}
	if run != nil {
		m.runDownload = run
	}
}

// InstallBrowserSpecResolverForTest makes authenticated behavioral tests
// independent of the host while the production default remains runtime-gated.
func InstallBrowserSpecResolverForTest(m *Manager, resolver func(authsource.Descriptor) (string, error)) {
	m.resolveBrowserSpec = resolver
}

const RetryCodeMediaLinkExpiredForTest = retryCodeMediaLinkExpired
