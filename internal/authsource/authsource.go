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
	BrowserChrome   Browser = "chrome"
	BrowserChromium Browser = "chromium"
	BrowserEdge     Browser = "edge"
	BrowserBrave    Browser = "brave"
	BrowserVivaldi  Browser = "vivaldi"
	BrowserOpera    Browser = "opera"
	BrowserFirefox  Browser = "firefox"
	BrowserSafari   Browser = "safari"
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
	profileRoot string
}

type sourceEnvironment struct {
	platform       string
	home           string
	configHome     string
	localAppData   string
	roamingAppData string
}

type chromiumRoot struct {
	browser    Browser
	label      string
	root       string
	noProfiles bool
}

func SupportedOptions() []Option {
	return supportedOptionsIn(currentEnvironment())
}

func supportedOptions(platform string) []Option {
	return supportedOptionsIn(environmentAt(platform, ""))
}

func supportedOptionsAt(platform, home string) []Option {
	return supportedOptionsIn(environmentAt(platform, home))
}

func supportedOptionsIn(environment sourceEnvironment) []Option {
	if !supportedPlatform(environment.platform) {
		return []Option{}
	}
	sources := discoverSources(environment)
	out := make([]Option, 0, len(sources))
	for _, source := range sources {
		out = append(out, source.option)
	}
	return out
}

// DescriptorForOption resolves only a backend-advertised option. Free-form
// browser names, profiles, containers, paths, and cookie material are rejected.
func DescriptorForOption(optionID string) (Descriptor, error) {
	return descriptorForOptionIn(optionID, currentEnvironment())
}

func descriptorForOption(optionID, platform string) (Descriptor, error) {
	return descriptorForOptionIn(optionID, environmentAt(platform, ""))
}

func descriptorForOptionAt(optionID, platform, home string) (Descriptor, error) {
	return descriptorForOptionIn(optionID, environmentAt(platform, home))
}

func descriptorForOptionIn(optionID string, environment sourceEnvironment) (Descriptor, error) {
	if !supportedPlatform(environment.platform) {
		return Descriptor{}, NewError("unsupported-platform")
	}
	for _, source := range discoverSources(environment) {
		if optionID == source.option.ID {
			return source.descriptor, nil
		}
	}
	return Descriptor{}, NewError("invalid-source")
}

func Label(descriptor Descriptor) string {
	for _, source := range discoverSources(currentEnvironment()) {
		if descriptor == source.descriptor {
			return source.option.Label
		}
	}
	if label := browserLabel(descriptor.Browser); label != "" {
		if descriptor.ProfileRef == "" {
			return label + " — Default"
		}
		return label + " profile"
	}
	return "Browser source"
}

// ValidateDescriptor validates only the durable descriptor shape. Source
// existence is checked at operation time so a removed profile becomes Action
// required instead of making State v2 corrupt during startup.
func ValidateDescriptor(descriptor Descriptor) error {
	if descriptor.SchemaVersion != DescriptorSchemaVersion || !supportedPlatform(descriptor.Platform) || !browserSupported(descriptor.Platform, descriptor.Browser) {
		return NewError("invalid-source")
	}
	switch descriptor.Browser {
	case BrowserFirefox:
		if !validOpaqueRef(descriptor.ProfileRef) || (descriptor.ContainerRef != "" && !validOpaqueRef(descriptor.ContainerRef)) {
			return NewError("invalid-source")
		}
	case BrowserSafari, BrowserOpera:
		if descriptor.ProfileRef != "" || descriptor.ContainerRef != "" {
			return NewError("invalid-source")
		}
	default:
		if descriptor.ContainerRef != "" || (descriptor.ProfileRef != "" && !validOpaqueRef(descriptor.ProfileRef)) {
			return NewError("invalid-source")
		}
	}
	return nil
}

// CookiesFromBrowser derives the sole engine credential input. CookieFile is
// never represented by this package or accepted from callers.
func CookiesFromBrowser(descriptor Descriptor) (string, error) {
	return cookiesFromBrowserIn(descriptor, currentEnvironment())
}

func cookiesFromBrowser(descriptor Descriptor, platform string) (string, error) {
	return cookiesFromBrowserIn(descriptor, environmentAt(platform, ""))
}

func cookiesFromBrowserAt(descriptor Descriptor, platform, home string) (string, error) {
	return cookiesFromBrowserIn(descriptor, environmentAt(platform, home))
}

func cookiesFromBrowserIn(descriptor Descriptor, environment sourceEnvironment) (string, error) {
	if err := ValidateDescriptor(descriptor); err != nil {
		return "", err
	}
	if descriptor.Platform != environment.platform {
		return "", NewError("unsupported-platform")
	}
	for _, source := range discoverSources(environment) {
		if source.descriptor != descriptor {
			continue
		}
		browser := string(descriptor.Browser)
		switch descriptor.Browser {
		case BrowserSafari, BrowserOpera:
			return browser, nil
		case BrowserFirefox:
			container := "none"
			if descriptor.ContainerRef != "" {
				container = "@" + strconv.Itoa(source.containerID)
			}
			return browser + ":" + source.profileName + "::" + container, nil
		default:
			if source.profileName == "" || source.profileName == "Default" {
				return browser, nil
			}
			return browser + ":" + source.profileName, nil
		}
	}
	return "", NewError("source-unavailable")
}

func currentEnvironment() sourceEnvironment {
	home, _ := os.UserHomeDir()
	environment := environmentAt(runtime.GOOS, home)
	if runtime.GOOS == "linux" {
		if configured := os.Getenv("XDG_CONFIG_HOME"); configured != "" {
			environment.configHome = configured
		}
	}
	if runtime.GOOS == "windows" {
		// The engine resolves Windows browser roots from these environment
		// variables, not from the user's home directory. Do not advertise a
		// home-derived source that the operation could later resolve elsewhere.
		environment.localAppData = ""
		environment.roamingAppData = ""
		if local := os.Getenv("LOCALAPPDATA"); filepath.IsAbs(local) {
			environment.localAppData = local
		}
		if roaming := os.Getenv("APPDATA"); filepath.IsAbs(roaming) {
			environment.roamingAppData = roaming
		}
	}
	return environment
}

func environmentAt(platform, home string) sourceEnvironment {
	environment := sourceEnvironment{platform: platform, home: home}
	switch platform {
	case "linux":
		if home != "" {
			environment.configHome = filepath.Join(home, ".config")
		}
	case "windows":
		if home != "" {
			environment.localAppData = filepath.Join(home, "AppData", "Local")
			environment.roamingAppData = filepath.Join(home, "AppData", "Roaming")
		}
	}
	return environment
}

func supportedPlatform(platform string) bool {
	return platform == "darwin" || platform == "linux" || platform == "windows"
}

func browserSupported(platform string, browser Browser) bool {
	switch platform {
	case "darwin":
		return browser == BrowserChrome || browser == BrowserFirefox || browser == BrowserSafari
	case "linux":
		return browser == BrowserChrome || browser == BrowserChromium || browser == BrowserBrave || browser == BrowserFirefox
	case "windows":
		switch browser {
		case BrowserChrome, BrowserChromium, BrowserEdge, BrowserBrave, BrowserVivaldi, BrowserOpera, BrowserFirefox:
			return true
		}
	}
	return false
}

func discoverSources(environment sourceEnvironment) []discoveredSource {
	if !supportedPlatform(environment.platform) {
		return nil
	}
	var out []discoveredSource
	for _, root := range chromiumRoots(environment) {
		out = append(out, discoverChromium(environment.platform, root)...)
	}
	out = append(out, discoverFirefox(environment)...)
	if environment.platform == "darwin" {
		// Safari is an operating-system source. Advertise its fixed default even
		// when privacy controls prevent discovery from probing the cookie file;
		// the explicit source check owns the bounded permission/missing result.
		out = append(out, discoveredSource{
			option:     Option{ID: "darwin.safari.default", Browser: BrowserSafari, Label: "Safari — Default", ProfileLabel: "Default"},
			descriptor: Descriptor{SchemaVersion: DescriptorSchemaVersion, Platform: "darwin", Browser: BrowserSafari},
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].option.Browser != out[j].option.Browser {
			return out[i].option.Browser < out[j].option.Browser
		}
		return out[i].option.Label < out[j].option.Label
	})
	return out
}

func chromiumRoots(environment sourceEnvironment) []chromiumRoot {
	switch environment.platform {
	case "darwin":
		return []chromiumRoot{{BrowserChrome, "Chrome", joinRoot(environment.home, "Library", "Application Support", "Google", "Chrome"), false}}
	case "linux":
		return []chromiumRoot{
			{BrowserChrome, "Chrome", joinRoot(environment.configHome, "google-chrome"), false},
			{BrowserChromium, "Chromium", joinRoot(environment.configHome, "chromium"), false},
			{BrowserBrave, "Brave", joinRoot(environment.configHome, "BraveSoftware", "Brave-Browser"), false},
		}
	case "windows":
		return []chromiumRoot{
			{BrowserChrome, "Chrome", joinRoot(environment.localAppData, "Google", "Chrome", "User Data"), false},
			{BrowserChromium, "Chromium", joinRoot(environment.localAppData, "Chromium", "User Data"), false},
			{BrowserEdge, "Edge", joinRoot(environment.localAppData, "Microsoft", "Edge", "User Data"), false},
			{BrowserBrave, "Brave", joinRoot(environment.localAppData, "BraveSoftware", "Brave-Browser", "User Data"), false},
			{BrowserVivaldi, "Vivaldi", joinRoot(environment.localAppData, "Vivaldi", "User Data"), false},
			{BrowserOpera, "Opera", joinRoot(environment.roamingAppData, "Opera Software", "Opera Stable"), true},
		}
	}
	return nil
}

func discoverChromium(platform string, root chromiumRoot) []discoveredSource {
	if root.root == "" {
		return nil
	}
	if root.noProfiles {
		if !hasRegularCookieDatabase(root.root, []string{filepath.Join("Network", "Cookies"), "Cookies"}) {
			return nil
		}
		return []discoveredSource{{
			option:     Option{ID: platform + "." + string(root.browser) + ".default", Browser: root.browser, Label: root.label + " — Default", ProfileLabel: "Default"},
			descriptor: Descriptor{SchemaVersion: DescriptorSchemaVersion, Platform: platform, Browser: root.browser}, profileRoot: root.root,
		}}
	}
	entries := safeChildDirectories(root.root)
	out := make([]discoveredSource, 0, len(entries))
	for _, entry := range entries {
		name := entry.Name()
		if !hasRegularCookieDatabase(filepath.Join(root.root, name), []string{filepath.Join("Network", "Cookies"), "Cookies"}) {
			continue
		}
		profileLabel := safeLabel(name, "Profile")
		descriptor := Descriptor{SchemaVersion: DescriptorSchemaVersion, Platform: platform, Browser: root.browser}
		optionID := platform + "." + string(root.browser) + ".default"
		profileName := "Default"
		if name != "Default" {
			ref := opaqueRef(platform, string(root.browser), name)
			descriptor.ProfileRef = ref
			optionID = platform + "." + string(root.browser) + "." + ref
			profileName = name
		}
		out = append(out, discoveredSource{
			option:     Option{ID: optionID, Browser: root.browser, Label: root.label + " — " + profileLabel, ProfileLabel: profileLabel},
			descriptor: descriptor, profileName: profileName, profileRoot: filepath.Join(root.root, name),
		})
	}
	return out
}

func discoverFirefox(environment sourceEnvironment) []discoveredSource {
	type candidate struct {
		root string
		name string
	}
	var candidates []candidate
	counts := make(map[string]int)
	for _, root := range firefoxRoots(environment) {
		for _, entry := range safeChildDirectories(root) {
			name := entry.Name()
			if !regularNoSymlink(filepath.Join(root, name, "cookies.sqlite")) {
				continue
			}
			candidates = append(candidates, candidate{root: root, name: name})
			counts[name]++
		}
	}
	var out []discoveredSource
	for _, candidate := range candidates {
		// The public engine selector accepts a profile name rather than a path.
		// Do not advertise an ambiguous name that could select another install.
		if counts[candidate.name] != 1 {
			continue
		}
		profileDir := filepath.Join(candidate.root, candidate.name)
		profileRef := opaqueRef(environment.platform, string(BrowserFirefox), candidate.root, candidate.name)
		profileLabel := firefoxProfileLabel(candidate.name)
		baseDescriptor := Descriptor{SchemaVersion: DescriptorSchemaVersion, Platform: environment.platform, Browser: BrowserFirefox, ProfileRef: profileRef}
		prefix := environment.platform + ".firefox." + profileRef
		out = append(out, discoveredSource{
			option:     Option{ID: prefix, Browser: BrowserFirefox, Label: "Firefox — " + profileLabel, ProfileLabel: profileLabel},
			descriptor: baseDescriptor, profileName: candidate.name, profileRoot: profileDir,
		})
		for _, container := range discoverFirefoxContainers(profileDir) {
			containerRef := opaqueRef(environment.platform, string(BrowserFirefox), candidate.root, candidate.name, "container", strconv.Itoa(container.id))
			containerLabel := safeLabel(container.label, "Container")
			out = append(out, discoveredSource{
				option:      Option{ID: prefix + "." + containerRef, Browser: BrowserFirefox, Label: "Firefox — " + profileLabel + " · " + containerLabel, ProfileLabel: profileLabel, ContainerLabel: containerLabel},
				descriptor:  Descriptor{SchemaVersion: DescriptorSchemaVersion, Platform: environment.platform, Browser: BrowserFirefox, ProfileRef: profileRef, ContainerRef: containerRef},
				profileName: candidate.name, containerID: container.id, profileRoot: profileDir,
			})
		}
	}
	return out
}

func firefoxRoots(environment sourceEnvironment) []string {
	switch environment.platform {
	case "darwin":
		return compactRoots(joinRoot(environment.home, "Library", "Application Support", "Firefox", "Profiles"))
	case "windows":
		return compactRoots(
			joinRoot(environment.roamingAppData, "Mozilla", "Firefox", "Profiles"),
			joinRoot(environment.localAppData, "Packages", "Mozilla.Firefox_n80bbvh6b1yt2", "LocalCache", "Roaming", "Mozilla", "Firefox", "Profiles"),
		)
	case "linux":
		return compactRoots(
			joinRoot(environment.configHome, "mozilla", "firefox"),
			joinRoot(environment.home, ".mozilla", "firefox"),
			joinRoot(environment.home, ".var", "app", "org.mozilla.firefox", "config", "mozilla", "firefox"),
			joinRoot(environment.home, ".var", "app", "org.mozilla.firefox", ".mozilla", "firefox"),
			joinRoot(environment.home, "snap", "firefox", "common", ".mozilla", "firefox"),
		)
	}
	return nil
}

func joinRoot(base string, elements ...string) string {
	if base == "" || !filepath.IsAbs(base) {
		return ""
	}
	return filepath.Join(append([]string{base}, elements...)...)
}

func compactRoots(roots ...string) []string {
	out := make([]string, 0, len(roots))
	for _, root := range roots {
		if root != "" {
			out = append(out, root)
		}
	}
	return out
}

func browserLabel(browser Browser) string {
	switch browser {
	case BrowserChrome:
		return "Chrome"
	case BrowserChromium:
		return "Chromium"
	case BrowserEdge:
		return "Edge"
	case BrowserBrave:
		return "Brave"
	case BrowserVivaldi:
		return "Vivaldi"
	case BrowserOpera:
		return "Opera"
	case BrowserFirefox:
		return "Firefox"
	case BrowserSafari:
		return "Safari"
	default:
		return ""
	}
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
	if strings.TrimSpace(root) == "" {
		return nil
	}
	info, err := os.Lstat(root)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return nil
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil
	}
	out := make([]os.DirEntry, 0, min(len(entries), maxProfiles))
	for _, entry := range entries {
		if entry.Type()&os.ModeSymlink != 0 || !entry.IsDir() || !safeBasename(entry.Name()) {
			continue
		}
		out = append(out, entry)
		if len(out) > maxProfiles {
			return nil
		}
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
	if strings.TrimSpace(path) == "" {
		return false
	}
	info, err := os.Lstat(path)
	return err == nil && info.Mode().IsRegular() && info.Mode()&os.ModeSymlink == 0
}

func safeBasename(value string) bool {
	return value != "" && value != "." && value != ".." && filepath.Base(value) == value && !strings.ContainsAny(value, "/\\\\\x00")
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
