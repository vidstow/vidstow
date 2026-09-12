# Lifecycle UI foundation

These components define presentation-safe lifecycle contracts and executable
interaction behavior. The live Queue and Settings routes consume the
backend-authored `QueueView` contract; ordered `job:update` and `queue:update`
events both carry a monotonically revised view so progress cannot freeze or
regress when bridge delivery is delayed.

`CannotSaveDialog` is the live stop when startup cannot write the data folder
or the folder is unsafe. `RecoveryRequiredShell` remains in this package for
the contract and tests; `App.svelte` does not mount it. Unreadable queue state
is skipped instead, with a dismissible workstation notice.

Row and queue actions require an explicit positive backend capability plus an
opaque command token. Persistence failure revokes every action in both the
backend projection and the route adapter. Lifecycle, phase, desired state, and
slot occupancy remain distinct fields throughout the contract.

Destination Review and Download again intentionally remain unavailable until
the backend exposes the public session/re-admission facades needed to authorize
them after restart. The UI therefore renders no enabled action for those paths.

Destination-conflict actions echo only a backend-issued opaque token. Runtime
payloads fail closed unless that token is a non-empty Unicode scalar string of
at most 256 UTF-8 bytes with no C0/C1 control characters. The UI never treats
the displayed proposed filename as authority.
