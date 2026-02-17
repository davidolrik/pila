package core

import "testing"

func TestResolveVersion_WithLdflags(t *testing.T) {
	original := version
	defer func() { version = original }()

	version = "1.2.3"
	got := resolveVersion()
	if got != "1.2.3" {
		t.Errorf("resolveVersion() = %q, want %q", got, "1.2.3")
	}
}

func TestResolveVersion_FallbackToBuildInfo(t *testing.T) {
	original := version
	defer func() { version = original }()

	version = ""
	got := resolveVersion()
	// In test binaries, debug.ReadBuildInfo() returns (devel) as the version,
	// so resolveVersion should return the fallback.
	if got != "(devel)" {
		t.Errorf("resolveVersion() = %q, want %q", got, "(devel)")
	}
}
