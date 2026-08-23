# Contributing to VidStow

Keep changes focused on the Desktop product and preserve the deliberate
boundary between the UI workflow and the engine/provider composition.
VidStow must continue to use the public engine + providers/youtube packages
from the ytdlp-go module github.com/tejasa97/ytdlp-go; do not reach into its
internal packages or the broad pkg/ytdlp facade.

## Platform scope

VidStow is built for macOS, Linux, and Windows. Product code and features should
support all three; the platforms currently receiving generated release
artifacts do not define product scope. See [Platform support](docs/PLATFORM_SUPPORT.md).

Before submitting a change:

~~~sh
gofmt -w .
go mod tidy -diff
go vet ./...
go test -count=1 ./...
go build ./...

cd frontend
npm ci
npm run check
npm run test:ui
npm run build
~~~

Run Wails builds when a native build is available on the host. Keep UI
validation and request mapping narrow; do not add hidden source-specific or
release-distribution feature gates to the engine composition.

By contributing, you agree that your contribution is available under the
repository's Apache-2.0 license.
