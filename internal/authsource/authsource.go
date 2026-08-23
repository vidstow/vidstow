// Package authsource owns VidStow's non-secret browser-source descriptors.
// Renderer values are limited to backend-advertised option IDs; engine cookie
// specifications are derived only after a durable binding has been resolved.
package authsource

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	DescriptorSchemaVersion = 1
	CurrentConsentVersion   = 1
	maxProfiles             = 128
	maxContainers           = 256
	maxContainersFile       = 1 << 20
)

type Browser string

const (
	BrowserChrome  Browser = "chrome"
	BrowserFirefox Browser = "firefox"
	BrowserSafari  Browser = "safari"
)

// Descriptor is non-secret durable authority. ProfileRef and ContainerRef are
// opaque backend-authored references, never filesystem paths or browser data.
type Descriptor struct {
	SchemaVersion int     `json:"schemaVersion"`
	Platform      string  `json:"platform"`
	Browser       Browser `json:"browser"`
	ProfileRef    string  `json:"profileRef,omitempty"`
	ContainerRef  string  `json:"containerRef,omitempty"`
}

// Binding has immutable identity. Forgetting a source tombstones it by setting
// Enabled=false so admitted jobs retain their exact non-secret intent.
type Binding struct {
	ID         string     `json:"id"`
	Descriptor Descriptor `json:"descriptor"`
	Enabled    bool       `json:"enabled"`
}

// Option is safe to expose to the renderer. ID is an allowlisted selection,
// not a path or an engine cookie specification.
type Option struct {
	ID             string  `json:"id"`
	Browser        Browser `json:"browser"`
	Label          string  `json:"label"`
	ProfileLabel   string  `json:"profileLabel,omitempty"`
	ContainerLabel string  `json:"containerLabel,omitempty"`
}

type discoveredSource struct {
	option      Option
	descriptor  Descriptor
	profileName string
	containerID int
}

var fixedDarwinSources = []discoveredSource{
	{option: Option{ID: "darwin.chrome.default", Browser: BrowserChrome, Label: "Chrome — Default", ProfileLabel: "Default"}, descriptor: Descriptor{SchemaVersion: DescriptorSchemaVersion, Platform: "darwin", Browser: BrowserChrome}},
	{option: Option{ID: "darwin.safari.default", Browser: BrowserSafari, Label: "Safari — Default", ProfileLabel: "Default"}, descriptor: Descriptor{SchemaVersion: DescriptorSchemaVersion, Platform: "darwin", Browser: BrowserSafari}},
}

func SupportedOptions() []Option {
	home, _ := os.UserHomeDir()
	return supportedOptionsAt(runtime.GOOS, home)
}

func supportedOptions(platform string) []Option {
	return supportedOptionsAt(platform, "")
}

func supportedOptionsAt(platform, home string) []Option {
	if platform != "darwin" {
		return []Option{}
	}
	sources := discoverSources(home)
	out := make([]Option, 0, len(sources))
	for _, source := range sources {
		out = append(out, source.option)
	}
	return out
}

// DescriptorForOption resolves only a backend-advertised option. Free-form
// browser names, profiles, containers, paths, and cookie material are rejected.
func DescriptorForOption(optionID string) (Descriptor, error) {
	home, _ := os.UserHomeDir()
	return descriptorForOptionAt(optionID, runtime.GOOS, home)
}

func descriptorForOption(optionID, platform string) (Descriptor, error) {
	return descriptorForOptionAt(optionID, platform, "")
}

func descriptorForOptionAt(optionID, platform, home string) (Descriptor, error) {
	if platform != "darwin" {
		return Descriptor{}, NewError("unsupported-platform")
	}
	for _, source := range discoverSources(home) {
		if optionID == source.option.ID {
			return source.descriptor, nil
		}
	}
	return Descriptor{}, NewError("invalid-source")
}

func Label(descriptor Descriptor) string {
	home, _ := os.UserHomeDir()
	for _, source := range discoverSources(home) {
		if descriptor == source.descriptor {
			return source.option.Label
		}
	}
	switch descriptor.Browser {
	case BrowserChrome:
		return "Chrome profile"
	case BrowserFirefox:
		return "Firefox profile"
	case BrowserSafari:
		return "Safari — Default"
	default:
		return "Browser source"
	}
}

// ValidateDescriptor validates only the durable descriptor shape. Source
// existence is checked at operation time so a removed profile becomes Action
// required instead of making State v2 corrupt during startup.
func ValidateDescriptor(descriptor Descriptor) error {
	if descriptor.SchemaVersion != DescriptorSchemaVersion || descriptor.Platform != "darwin" {
		return NewError("invalid-source")
	}
	switch descriptor.Browser {
	case BrowserChrome:
		if descriptor.ContainerRef != "" || (descriptor.ProfileRef != "" && !validOpaqueRef(descriptor.ProfileRef)) {
			return NewError("invalid-source")
		}
	case BrowserFirefox:
		if !validOpaqueRef(descriptor.ProfileRef) || (descriptor.ContainerRef != "" && !validOpaqueRef(descriptor.ContainerRef)) {
			return NewError("invalid-source")
		}
	case BrowserSafari:
		if descriptor.ProfileRef != "" || descriptor.ContainerRef != "" {
			return NewError("invalid-source")
		}
	default:
		return NewError("invalid-source")
	}
	return nil
}

// CookiesFromBrowser derives the sole engine credential input. CookieFile is
// never represented by this package or accepted from callers.
func CookiesFromBrowser(descriptor Descriptor) (string, error) {
	home, _ := os.UserHomeDir()
	return cookiesFromBrowserAt(descriptor, runtime.GOOS, home)
}

func cookiesFromBrowser(descriptor Descriptor, platform string) (string, error) {
	return cookiesFromBrowserAt(descriptor, platform, "")
}

func cookiesFromBrowserAt(descriptor Descriptor, platform, home string) (string, error) {
	if err := ValidateDescriptor(descriptor); err != nil {
		return "", err
	}
	if platform != "darwin" || descriptor.Platform != platform {
		return "", NewError("unsupported-platform")
	}
	if descriptor.Browser == BrowserChrome && descriptor.ProfileRef == "" {
		return "chrome", nil
	}
	if descriptor.Browser == BrowserSafari {
		return "safari", nil
	}
	for _, source := range discoverSources(home) {
		if source.descriptor != descriptor {
			continue
		}
		switch descriptor.Browser {
		case BrowserChrome:
			return "chrome:" + source.profileName, nil
		case BrowserFirefox:
			container := "none"
			if descriptor.ContainerRef != "" {
				container = "@" + strconv.Itoa(source.containerID)
			}
			return "firefox:" + source.profileName + "::" + container, nil
		}
	}
	return "", NewError("source-unavailable")
}

func discoverSources(home string) []discoveredSource {
	out := append([]discoveredSource(nil), fixedDarwinSources...)
	if strings.TrimSpace(home) == "" {
		return out
	}
	out = append(out, discoverChrome(home)...)
	out = append(out, discoverFirefox(home)...)
	sort.SliceStable(out[2:], func(i, j int) bool {
		a, b := out[i+2].option, out[j+2].option
		if a.Browser != b.Browser {
			return a.Browser < b.Browser
		}
		return a.Label < b.Label
	})
	return out
}

func discoverChrome(home string) []discoveredSource {
	root := filepath.Join(home, "Library", "Application Support", "Google", "Chrome")
	entries := safeChildDirectories(root)
	out := make([]discoveredSource, 0, len(entries))
	for _, entry := range entries {
		name := entry.Name()
		if name == "Default" || !hasRegularCookieDatabase(filepath.Join(root, name), []string{filepath.Join("Network", "Cookies"), "Cookies"}) {
			continue
		}
		ref := opaqueRef("darwin", string(BrowserChrome), name)
		profileLabel := safeLabel(name, "Profile")
		out = append(out, discoveredSource{
			option:      Option{ID: "darwin.chrome." + ref, Browser: BrowserChrome, Label: "Chrome — " + profileLabel, ProfileLabel: profileLabel},
			descriptor:  Descriptor{SchemaVersion: DescriptorSchemaVersion, Platform: "darwin", Browser: BrowserChrome, ProfileRef: ref},
			profileName: name,
		})
	}
	return out
}

func discoverFirefox(home string) []discoveredSource {
	root := filepath.Join(home, "Library", "Application Support", "Firefox", "Profiles")
	entries := safeChildDirectories(root)
	out := make([]discoveredSource, 0, len(entries))
	for _, entry := range entries {
		name := entry.Name()
		profileDir := filepath.Join(root, name)
		if !regularNoSymlink(filepath.Join(profileDir, "cookies.sqlite")) {
			continue
		}
		profileRef := opaqueRef("darwin", string(BrowserFirefox), name)
		profileLabel := firefoxProfileLabel(name)
		baseDescriptor := Descriptor{SchemaVersion: DescriptorSchemaVersion, Platform: "darwin", Browser: BrowserFirefox, ProfileRef: profileRef}
		out = append(out, discoveredSource{
			option:      Option{ID: "darwin.firefox." + profileRef, Browser: BrowserFirefox, Label: "Firefox — " + profileLabel, ProfileLabel: profileLabel},
			descriptor:  baseDescriptor,
			profileName: name,
		})
		for _, container := range discoverFirefoxContainers(profileDir) {
			containerRef := opaqueRef("darwin", string(BrowserFirefox), name, "container", strconv.Itoa(container.id))
			containerLabel := safeLabel(container.label, "Container")
			out = append(out, discoveredSource{
				option:      Option{ID: "darwin.firefox." + profileRef + "." + containerRef, Browser: BrowserFirefox, Label: "Firefox — " + profileLabel + " · " + containerLabel, ProfileLabel: profileLabel, ContainerLabel: containerLabel},
				descriptor:  Descriptor{SchemaVersion: DescriptorSchemaVersion, Platform: "darwin", Browser: BrowserFirefox, ProfileRef: profileRef, ContainerRef: containerRef},
				profileName: name,
				containerID: container.id,
			})
		}
	}
	return out
}

type firefoxContainer struct {
	id    int
	label string
}

func discoverFirefoxContainers(profileDir string) []firefoxContainer {
	path := filepath.Join(profileDir, "containers.json")
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Size() < 0 || info.Size() > maxContainersFile {
		return nil
	}
	raw, err := os.ReadFile(path)
	if err != nil || len(raw) > maxContainersFile {
		return nil
	}
	var doc struct {
		Identities []struct {
			Name          string `json:"name"`
			L10nID        string `json:"l10nID"`
			UserContextID int    `json:"userContextId"`
		} `json:"identities"`
	}
	if json.Unmarshal(raw, &doc) != nil || len(doc.Identities) > maxContainers {
		return nil
	}
	seen := map[int]bool{}
	out := make([]firefoxContainer, 0, len(doc.Identities))
	for _, identity := range doc.Identities {
		if identity.UserContextID <= 0 || identity.UserContextID > 1_000_000 || seen[identity.UserContextID] {
			continue
		}
		seen[identity.UserContextID] = true
		label := strings.TrimSpace(identity.Name)
		if label == "" {
			label = strings.TrimSuffix(strings.TrimPrefix(identity.L10nID, "userContext"), ".label")
		}
		out = append(out, firefoxContainer{id: identity.UserContextID, label: label})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].id < out[j].id })
	return out
}

func safeChildDirectories(root string) []os.DirEntry {
	info, err := os.Lstat(root)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return nil
	}
	entries, err := os.ReadDir(root)
	if err != nil || len(entries) > maxProfiles {
		return nil
	}
	out := make([]os.DirEntry, 0, len(entries))
	for _, entry := range entries {
		if entry.Type()&os.ModeSymlink != 0 || !entry.IsDir() || !safeBasename(entry.Name()) {
			continue
		}
		out = append(out, entry)
	}
	return out
}

func hasRegularCookieDatabase(profileDir string, relatives []string) bool {
	for _, relative := range relatives {
		if regularNoSymlink(filepath.Join(profileDir, relative)) {
			return true
		}
	}
	return false
}

func regularNoSymlink(path string) bool {
	info, err := os.Lstat(path)
	return err == nil && info.Mode().IsRegular() && info.Mode()&os.ModeSymlink == 0
}

func safeBasename(value string) bool {
	return value != "" && value != "." && value != ".." && filepath.Base(value) == value && !strings.ContainsAny(value, `/\\\x00`)
}

func firefoxProfileLabel(name string) string {
	lower := strings.ToLower(name)
	if strings.HasSuffix(lower, ".default") || strings.HasSuffix(lower, ".default-release") {
		return "Default"
	}
	if _, suffix, ok := strings.Cut(name, "."); ok && suffix != "" {
		return safeLabel(suffix, "Profile")
	}
	return safeLabel(name, "Profile")
}

func safeLabel(value, fallback string) string {
	value = strings.ToValidUTF8(strings.TrimSpace(value), "")
	var b strings.Builder
	for _, r := range value {
		if unicode.IsControl(r) || r == '\u2028' || r == '\u2029' {
			b.WriteByte(' ')
			continue
		}
		b.WriteRune(r)
		if b.Len() >= 80 {
			break
		}
	}
	value = strings.Join(strings.Fields(b.String()), " ")
	for len(value) > 80 || !utf8.ValidString(value) {
		value = value[:len(value)-1]
	}
	if value == "" || strings.Contains(value, "://") {
		return fallback
	}
	return value
}

func opaqueRef(parts ...string) string {
	digest := sha256.Sum256([]byte(strings.Join(parts, "\x00")))
	return hex.EncodeToString(digest[:])
}

func validOpaqueRef(value string) bool {
	if len(value) != sha256.Size*2 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

// Error is a bounded, path-free source failure suitable for classification.
type Error struct{ Code string }

func (e *Error) Error() string { return "browser source unavailable" }

func NewError(code string) error {
	code = strings.TrimSpace(code)
	if code == "" {
		code = "source-unavailable"
	}
	return &Error{Code: code}
}

func ErrorCode(err error) (string, bool) {
	var sourceErr *Error
	if !errors.As(err, &sourceErr) {
		return "", false
	}
	return sourceErr.Code, true
}

func (d Descriptor) String() string {
	return fmt.Sprintf("browser source %s", d.Browser)
}
