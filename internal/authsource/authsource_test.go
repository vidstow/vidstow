package authsource

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

func TestDescriptorValidationUsesCrossPlatformBrowserMatrix(t *testing.T) {
	valid := []Descriptor{
		{SchemaVersion: DescriptorSchemaVersion, Platform: "darwin", Browser: BrowserChrome},
		{SchemaVersion: DescriptorSchemaVersion, Platform: "darwin", Browser: BrowserSafari},
		{SchemaVersion: DescriptorSchemaVersion, Platform: "linux", Browser: BrowserChromium},
		{SchemaVersion: DescriptorSchemaVersion, Platform: "linux", Browser: BrowserBrave, ProfileRef: opaqueRef("linux-profile")},
		{SchemaVersion: DescriptorSchemaVersion, Platform: "windows", Browser: BrowserEdge},
		{SchemaVersion: DescriptorSchemaVersion, Platform: "windows", Browser: BrowserOpera},
		{SchemaVersion: DescriptorSchemaVersion, Platform: "windows", Browser: BrowserFirefox, ProfileRef: opaqueRef("firefox")},
	}
	for _, descriptor := range valid {
		if err := ValidateDescriptor(descriptor); err != nil {
			t.Fatalf("valid descriptor rejected: %#v: %v", descriptor, err)
		}
	}
	invalid := []Descriptor{
		{SchemaVersion: DescriptorSchemaVersion, Platform: "darwin", Browser: BrowserEdge},
		{SchemaVersion: DescriptorSchemaVersion, Platform: "linux", Browser: BrowserSafari},
		{SchemaVersion: DescriptorSchemaVersion, Platform: "windows", Browser: BrowserSafari},
		{SchemaVersion: DescriptorSchemaVersion, Platform: "freebsd", Browser: BrowserChrome},
		{SchemaVersion: DescriptorSchemaVersion, Platform: "windows", Browser: BrowserChrome, ProfileRef: `C:\Users\private\Default`},
		{SchemaVersion: DescriptorSchemaVersion, Platform: "linux", Browser: BrowserFirefox},
		{SchemaVersion: DescriptorSchemaVersion, Platform: "windows", Browser: BrowserOpera, ProfileRef: opaqueRef("profile")},
		{SchemaVersion: DescriptorSchemaVersion, Platform: "darwin", Browser: BrowserSafari, ProfileRef: opaqueRef("profile")},
	}
	for _, descriptor := range invalid {
		if err := ValidateDescriptor(descriptor); err == nil {
			t.Fatalf("unsafe descriptor accepted: %#v", descriptor)
		}
	}
}

func TestDarwinDiscoveryAndMappingUseOpaqueRefs(t *testing.T) {
	home := t.TempDir()
	writeCookieDatabase(t, filepath.Join(home, "Library", "Application Support", "Google", "Chrome", "Default"))
	writeCookieDatabase(t, filepath.Join(home, "Library", "Application Support", "Google", "Chrome", "Profile 1"))
	firefox := filepath.Join(home, "Library", "Application Support", "Firefox", "Profiles", "abc.default-release")
	writeFirefoxDatabase(t, firefox, true)
	writeRegular(t, filepath.Join(home, "Library", "Cookies", "Cookies.binarycookies"))

	environment := environmentAt("darwin", home)
	assertSourceSpecs(t, environment, map[string]string{
		"Chrome|Default|":      "chrome",
		"Chrome|Profile 1|":    "chrome:Profile 1",
		"Firefox|Default|":     "firefox:abc.default-release::none",
		"Firefox|Default|Work": "firefox:abc.default-release::@7",
		"Safari|Default|":      "safari",
	})
}

func TestLinuxDiscoveryCoversEngineBrowserMatrix(t *testing.T) {
	home := t.TempDir()
	environment := environmentAt("linux", home)
	for _, relative := range []string{
		filepath.Join("google-chrome", "Default"),
		filepath.Join("chromium", "Profile 2"),
		filepath.Join("BraveSoftware", "Brave-Browser", "Default"),
	} {
		writeCookieDatabase(t, filepath.Join(environment.configHome, relative))
	}
	writeFirefoxDatabase(t, filepath.Join(home, ".mozilla", "firefox", "xyz.work"), true)

	assertSourceSpecs(t, environment, map[string]string{
		"Chrome|Default|":     "chrome",
		"Chromium|Profile 2|": "chromium:Profile 2",
		"Brave|Default|":      "brave",
		"Firefox|work|":       "firefox:xyz.work::none",
		"Firefox|work|Work":   "firefox:xyz.work::@7",
	})
	for _, option := range supportedOptionsIn(environment) {
		if option.Browser == BrowserEdge || option.Browser == BrowserVivaldi || option.Browser == BrowserOpera || option.Browser == BrowserSafari {
			t.Fatalf("unsupported Linux browser advertised: %#v", option)
		}
	}
}

func TestWindowsDiscoveryCoversEngineBrowserMatrix(t *testing.T) {
	home := t.TempDir()
	environment := environmentAt("windows", home)
	roots := map[Browser]string{
		BrowserChrome:   filepath.Join(environment.localAppData, "Google", "Chrome", "User Data", "Default"),
		BrowserChromium: filepath.Join(environment.localAppData, "Chromium", "User Data", "Default"),
		BrowserEdge:     filepath.Join(environment.localAppData, "Microsoft", "Edge", "User Data", "Profile 3"),
		BrowserBrave:    filepath.Join(environment.localAppData, "BraveSoftware", "Brave-Browser", "User Data", "Default"),
		BrowserVivaldi:  filepath.Join(environment.localAppData, "Vivaldi", "User Data", "Default"),
		BrowserOpera:    filepath.Join(environment.roamingAppData, "Opera Software", "Opera Stable"),
	}
	for _, root := range roots {
		writeCookieDatabase(t, root)
	}
	writeFirefoxDatabase(t, filepath.Join(environment.roamingAppData, "Mozilla", "Firefox", "Profiles", "win.default-release"), false)

	assertSourceSpecs(t, environment, map[string]string{
		"Chrome|Default|":   "chrome",
		"Chromium|Default|": "chromium",
		"Edge|Profile 3|":   "edge:Profile 3",
		"Brave|Default|":    "brave",
		"Vivaldi|Default|":  "vivaldi",
		"Opera|Default|":    "opera",
		"Firefox|Default|":  "firefox:win.default-release::none",
	})
}

func TestFirefoxAmbiguousProfileNameIsNotAdvertised(t *testing.T) {
	home := t.TempDir()
	environment := environmentAt("linux", home)
	writeFirefoxDatabase(t, filepath.Join(home, ".mozilla", "firefox", "same.default"), false)
	writeFirefoxDatabase(t, filepath.Join(home, ".var", "app", "org.mozilla.firefox", ".mozilla", "firefox", "same.default"), false)
	for _, option := range supportedOptionsIn(environment) {
		if option.Browser == BrowserFirefox {
			t.Fatalf("ambiguous Firefox profile advertised: %#v", option)
		}
	}
}

func TestDiscoveryRejectsSymlinksAndRemovedProfiles(t *testing.T) {
	home := t.TempDir()
	environment := environmentAt("linux", home)
	outside := t.TempDir()
	writeCookieDatabase(t, outside)
	root := filepath.Join(environment.configHome, "google-chrome")
	if err := os.MkdirAll(root, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "Profile 2")); err != nil {
		t.Fatal(err)
	}
	if options := supportedOptionsIn(environment); len(options) != 0 {
		t.Fatalf("symlinked profile advertised: %#v", options)
	}

	profile := filepath.Join(root, "Profile 3")
	writeCookieDatabase(t, profile)
	options := supportedOptionsIn(environment)
	if len(options) != 1 {
		t.Fatalf("options = %#v; want one profile", options)
	}
	descriptor, err := descriptorForOptionIn(options[0].ID, environment)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(profile, "Network", "Cookies")); err != nil {
		t.Fatal(err)
	}
	if spec, err := cookiesFromBrowserIn(descriptor, environment); err == nil || spec != "" {
		t.Fatalf("removed profile resolved to %q, %v", spec, err)
	}
}

func TestDiscoveryIgnoresNonProfileClutterWithinProfileBound(t *testing.T) {
	home := t.TempDir()
	environment := environmentAt("linux", home)
	root := filepath.Join(environment.configHome, "google-chrome")
	writeCookieDatabase(t, filepath.Join(root, "Default"))
	for index := 0; index < maxProfiles+20; index++ {
		writeRegular(t, filepath.Join(root, "state-"+strconv.Itoa(index)))
	}
	options := supportedOptionsIn(environment)
	if len(options) != 1 || options[0].Browser != BrowserChrome || options[0].ProfileLabel != "Default" {
		t.Fatalf("profile hidden by unrelated browser files: %#v", options)
	}
}

func TestMissingOrRelativePlatformDirectoriesNeverUseRelativeDiscovery(t *testing.T) {
	for _, environment := range []sourceEnvironment{
		{platform: "linux"},
		{platform: "linux", home: "relative-home", configHome: "relative-config"},
		{platform: "windows"},
		{platform: "windows", home: "relative-home", localAppData: "relative-local", roamingAppData: "relative-roaming"},
	} {
		if options := supportedOptionsIn(environment); len(options) != 0 {
			t.Fatalf("%s unsafe environment advertised relative sources: %#v", environment.platform, options)
		}
	}
	options := supportedOptionsIn(sourceEnvironment{platform: "darwin"})
	if len(options) != 1 || options[0].Browser != BrowserSafari {
		t.Fatalf("Darwin empty environment options = %#v; want only fixed Safari source", options)
	}
}

func TestRuntimePlatformDoesNotAcceptAnotherPlatformBinding(t *testing.T) {
	other := "linux"
	if runtime.GOOS == other {
		other = "windows"
	}
	descriptor := Descriptor{SchemaVersion: DescriptorSchemaVersion, Platform: other, Browser: BrowserChrome}
	if spec, err := CookiesFromBrowser(descriptor); err == nil || spec != "" {
		t.Fatalf("runtime %s accepted %s descriptor: %q, %v", runtime.GOOS, other, spec, err)
	}
	if options := supportedOptions("freebsd"); len(options) != 0 {
		t.Fatalf("unsupported platform options = %#v", options)
	}
}

func assertSourceSpecs(t *testing.T, environment sourceEnvironment, expected map[string]string) {
	t.Helper()
	options := supportedOptionsIn(environment)
	got := make(map[string]string, len(options))
	for _, option := range options {
		if strings.Contains(option.ID, environment.home) || strings.ContainsAny(option.ID, `/\\`) {
			t.Fatalf("option ID leaked path: %#v", option)
		}
		descriptor, err := descriptorForOptionIn(option.ID, environment)
		if err != nil {
			t.Fatalf("descriptorForOptionIn(%q): %v", option.ID, err)
		}
		encoded, err := jsonDescriptor(descriptor)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(encoded, environment.home) || strings.Contains(encoded, "Cookies") {
			t.Fatalf("descriptor leaked source material: %s", encoded)
		}
		spec, err := cookiesFromBrowserIn(descriptor, environment)
		if err != nil {
			t.Fatalf("cookiesFromBrowserIn(%q): %v", option.ID, err)
		}
		key := browserLabel(option.Browser) + "|" + option.ProfileLabel + "|" + option.ContainerLabel
		got[key] = spec
	}
	if len(got) != len(expected) {
		t.Fatalf("source specs = %#v; want %#v", got, expected)
	}
	for key, want := range expected {
		if got[key] != want {
			t.Fatalf("source %q = %q; want %q (all: %#v)", key, got[key], want, got)
		}
	}
}

func jsonDescriptor(descriptor Descriptor) (string, error) {
	raw, err := json.Marshal(descriptor)
	return string(raw), err
}

func writeCookieDatabase(t *testing.T, profile string) {
	t.Helper()
	writeRegular(t, filepath.Join(profile, "Network", "Cookies"))
}

func writeFirefoxDatabase(t *testing.T, profile string, container bool) {
	t.Helper()
	writeRegular(t, filepath.Join(profile, "cookies.sqlite"))
	if container {
		if err := os.WriteFile(filepath.Join(profile, "containers.json"), []byte(`{"identities":[{"name":"Work","userContextId":7}]}`), 0o600); err != nil {
			t.Fatal(err)
		}
	}
}

func writeRegular(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("fixture"), 0o600); err != nil {
		t.Fatal(err)
	}
}
