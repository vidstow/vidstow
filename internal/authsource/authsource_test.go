package authsource

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestDescriptorValidationRejectsRendererAuthority(t *testing.T) {
	valid := Descriptor{SchemaVersion: DescriptorSchemaVersion, Platform: "darwin", Browser: BrowserChrome}
	if err := ValidateDescriptor(valid); err != nil {
		t.Fatalf("valid default descriptor: %v", err)
	}
	if spec, err := cookiesFromBrowser(valid, "darwin"); err != nil || spec != "chrome" {
		t.Fatalf("cookiesFromBrowser(darwin) = %q, %v", spec, err)
	}
	for _, descriptor := range []Descriptor{
		{SchemaVersion: DescriptorSchemaVersion, Platform: "darwin", Browser: BrowserChrome, ProfileRef: "/Users/private-canary/Profile"},
		{SchemaVersion: DescriptorSchemaVersion, Platform: "darwin", Browser: BrowserSafari, ProfileRef: "~/Library/Cookies/Cookies.binarycookies"},
		{SchemaVersion: DescriptorSchemaVersion, Platform: "darwin", Browser: BrowserFirefox},
		{SchemaVersion: DescriptorSchemaVersion, Platform: "darwin", Browser: BrowserFirefox, ContainerRef: "COOKIE_CANARY_7f2"},
		{SchemaVersion: DescriptorSchemaVersion, Platform: "darwin", Browser: "edge"},
	} {
		if err := ValidateDescriptor(descriptor); err == nil {
			t.Fatalf("unsafe descriptor accepted: %#v", descriptor)
		}
	}
	for _, option := range supportedOptions("darwin") {
		if option.Browser == BrowserFirefox {
			t.Fatalf("mutable bare Firefox option was advertised: %#v", option)
		}
	}
}

func TestNamedProfileAndFirefoxContainerDiscoveryUsesOpaqueRefs(t *testing.T) {
	home := t.TempDir()
	chrome := filepath.Join(home, "Library", "Application Support", "Google", "Chrome", "Profile 1", "Network")
	firefox := filepath.Join(home, "Library", "Application Support", "Firefox", "Profiles", "abc.default-release")
	if err := os.MkdirAll(chrome, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(chrome, "Cookies"), []byte("fixture"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(firefox, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(firefox, "cookies.sqlite"), []byte("fixture"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(firefox, "containers.json"), []byte(`{"identities":[{"name":"Work","userContextId":7}]}`), 0o600); err != nil {
		t.Fatal(err)
	}

	options := supportedOptionsAt("darwin", home)
	if len(options) != 5 {
		t.Fatalf("options = %#v; want two fixed plus Chrome, Firefox, and Firefox container", options)
	}
	var chromeID, firefoxID, containerID string
	for _, option := range options {
		if strings.Contains(option.ID, "/") || strings.Contains(option.ID, home) {
			t.Fatalf("option leaked path: %#v", option)
		}
		switch {
		case option.Browser == BrowserChrome && option.ProfileLabel == "Profile 1":
			chromeID = option.ID
		case option.Browser == BrowserFirefox && option.ContainerLabel == "":
			firefoxID = option.ID
		case option.Browser == BrowserFirefox && option.ContainerLabel == "Work":
			containerID = option.ID
		}
	}
	for _, test := range []struct {
		id   string
		want string
	}{
		{chromeID, "chrome:Profile 1"},
		{firefoxID, "firefox:abc.default-release::none"},
		{containerID, "firefox:abc.default-release::@7"},
	} {
		if test.id == "" {
			t.Fatalf("missing discovered option for %q: %#v", test.want, options)
		}
		descriptor, err := descriptorForOptionAt(test.id, "darwin", home)
		if err != nil {
			t.Fatalf("descriptorForOptionAt(%q): %v", test.id, err)
		}
		if got, err := cookiesFromBrowserAt(descriptor, "darwin", home); err != nil || got != test.want {
			t.Fatalf("cookiesFromBrowserAt(%q) = %q, %v; want %q", test.id, got, err, test.want)
		}
	}
}

func TestDiscoveryRejectsSymlinksAndRemovedProfiles(t *testing.T) {
	home := t.TempDir()
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "Cookies"), []byte("fixture"), 0o600); err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(home, "Library", "Application Support", "Google", "Chrome")
	if err := os.MkdirAll(root, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "Profile 2")); err != nil {
		t.Fatal(err)
	}
	if got := supportedOptionsAt("darwin", home); len(got) != 2 {
		t.Fatalf("symlinked profile advertised: %#v", got)
	}

	profile := filepath.Join(root, "Profile 3")
	if err := os.MkdirAll(profile, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(profile, "Cookies"), []byte("fixture"), 0o600); err != nil {
		t.Fatal(err)
	}
	options := supportedOptionsAt("darwin", home)
	descriptor, err := descriptorForOptionAt(options[len(options)-1].ID, "darwin", home)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(profile, "Cookies")); err != nil {
		t.Fatal(err)
	}
	if spec, err := cookiesFromBrowserAt(descriptor, "darwin", home); err == nil || spec != "" {
		t.Fatalf("removed profile resolved to %q, %v", spec, err)
	}
}

func TestDarwinOptionMappingIsPureAndRuntimeGateFailsClosed(t *testing.T) {
	for _, test := range []struct {
		option string
		want   string
	}{
		{option: "darwin.chrome.default", want: "chrome"},
		{option: "darwin.safari.default", want: "safari"},
	} {
		descriptor, err := descriptorForOption(test.option, "darwin")
		if err != nil {
			t.Fatalf("descriptorForOption(%q): %v", test.option, err)
		}
		if got, err := cookiesFromBrowser(descriptor, "darwin"); err != nil || got != test.want {
			t.Fatalf("cookiesFromBrowser(%q) = %q, %v; want %q", test.option, got, err, test.want)
		}
	}
	for _, platform := range []string{"linux", "windows", "freebsd"} {
		if options := supportedOptions(platform); len(options) != 0 {
			t.Fatalf("%s options = %#v; want none", platform, options)
		}
		if _, err := descriptorForOption("darwin.chrome.default", platform); err == nil {
			t.Fatalf("%s option mapping succeeded", platform)
		}
		valid := Descriptor{SchemaVersion: DescriptorSchemaVersion, Platform: "darwin", Browser: BrowserChrome}
		if spec, err := cookiesFromBrowser(valid, platform); err == nil || spec != "" {
			t.Fatalf("%s cookie mapping = %q, %v; want unsupported", platform, spec, err)
		}
	}
	valid := Descriptor{SchemaVersion: DescriptorSchemaVersion, Platform: "darwin", Browser: BrowserChrome}
	got, err := CookiesFromBrowser(valid)
	if runtime.GOOS == "darwin" {
		if err != nil || got != "chrome" {
			t.Fatalf("runtime Darwin mapping = %q, %v", got, err)
		}
	} else if err == nil || got != "" {
		t.Fatalf("runtime %s mapping = %q, %v; want unsupported", runtime.GOOS, got, err)
	}
}
